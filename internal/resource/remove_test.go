package resource

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

func TestPrepareRemove(t *testing.T) {
	t.Run("rejects default branch", func(t *testing.T) {
		svc := newTestSvc(&git.ClientMock{}, stubTmux(), WithCommonParams(defaultCP()))

		_, err := svc.PrepareRemove(context.Background(), "main")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot remove default branch")
	})

	t.Run("returns error when no resources exist", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				BranchExistsFunc:  mockBranchExists(),
				ListWorktreesFunc: func() ([]git.Worktree, error) { return nil, nil },
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
			},
			WithCommonParams(defaultCP()),
		)

		_, err := svc.PrepareRemove(context.Background(), "ghost")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("detects all resource states", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				BranchExistsFunc: mockBranchExists("feature"),
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return []git.Worktree{
						{Path: "/repo/.worktrees/feature", Branch: "feature"},
					}, nil
				},
				HasUncommittedChangesFunc: func(path string) (bool, error) { return true, nil },
				IsMergedFunc:              func(branch string, base string) (bool, error) { return false, nil },
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return true, nil },
				ListWindowsFunc: func(session string) ([]tmux.Window, error) {
					return []tmux.Window{{Name: "feature", Active: true}}, nil
				},
			},
			WithCommonParams(defaultCP()),
		)

		check, err := svc.PrepareRemove(context.Background(), "feature")
		require.NoError(t, err)
		assert.True(t, check.HasBranch)
		assert.True(t, check.HasWorktree)
		assert.Equal(t, "/repo/.worktrees/feature", check.WorktreePath)
		assert.True(t, check.HasWindow)
		assert.True(t, check.IsActive)
		assert.True(t, check.HasUncommitted)
		assert.True(t, check.IsUnmerged)
	})

	t.Run("orphaned window only", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				BranchExistsFunc:  mockBranchExists(),
				ListWorktreesFunc: func() ([]git.Worktree, error) { return nil, nil },
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return true, nil },
				ListWindowsFunc: func(session string) ([]tmux.Window, error) {
					return []tmux.Window{{Name: "orphan", Active: false}}, nil
				},
			},
			WithCommonParams(defaultCP()),
		)

		check, err := svc.PrepareRemove(context.Background(), "orphan")
		require.NoError(t, err)
		assert.False(t, check.HasBranch)
		assert.False(t, check.HasWorktree)
		assert.True(t, check.HasWindow)
	})

	t.Run("BranchExists error", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					return false, fmt.Errorf("git error")
				},
			},
			stubTmux(),
			WithCommonParams(defaultCP()),
		)

		_, err := svc.PrepareRemove(context.Background(), "feat")
		assert.Error(t, err)
	})

	t.Run("branch exists without worktree (merged)", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				BranchExistsFunc:  mockBranchExists("feat"),
				ListWorktreesFunc: func() ([]git.Worktree, error) { return nil, nil },
				IsMergedFunc:      func(branch string, base string) (bool, error) { return true, nil },
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
			},
			WithCommonParams(defaultCP()),
		)

		check, err := svc.PrepareRemove(context.Background(), "feat")
		require.NoError(t, err)
		assert.True(t, check.HasBranch)
		assert.False(t, check.HasWorktree)
		assert.False(t, check.IsUnmerged)
	})
}

func TestExecuteRemove(t *testing.T) {
	t.Run("removes all resources", func(t *testing.T) {
		var killedWindow, removedWT, deletedBranch bool
		svc := newTestSvc(
			&git.ClientMock{
				RemoveWorktreeFunc:   func(path string) error { removedWT = true; return nil },
				DeleteBranchFromFunc: func(dir string, name string) error { deletedBranch = true; return nil },
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return true, nil },
				ListWindowsFunc: func(session string) ([]tmux.Window, error) {
					return []tmux.Window{{Name: "main", Active: true}}, nil
				},
				KillWindowFunc: func(session string, window string) error { killedWindow = true; return nil },
			},
			WithCommonParams(defaultCP()),
		)

		check := RemoveCheck{
			Branch:       "feature",
			HasBranch:    true,
			HasWorktree:  true,
			WorktreePath: "/repo/.worktrees/feature",
			HasWindow:    true,
			IsActive:     false,
		}

		result, err := svc.ExecuteRemove(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, killedWindow)
		assert.True(t, removedWT)
		assert.True(t, deletedBranch)
		assert.True(t, result.WindowKilled)
		assert.True(t, result.WorktreeRemoved)
		assert.True(t, result.BranchDeleted)
	})

	t.Run("switches away from active window before removal", func(t *testing.T) {
		var ensureTmuxCalled bool
		svc := newTestSvc(
			&git.ClientMock{
				RemoveWorktreeFunc:   func(path string) error { return nil },
				DeleteBranchFromFunc: func(dir string, name string) error { return nil },
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return true, nil },
				ListWindowsFunc: func(session string) ([]tmux.Window, error) {
					return []tmux.Window{{Name: "main", Active: false}}, nil
				},
				PaneCurrentCommandFunc: func(session string, window string) (string, error) {
					return "zsh", nil
				},
				SendKeysFunc: func(session string, window string, keys ...string) error {
					if window == "main" {
						ensureTmuxCalled = true
					}
					return nil
				},
				IsInsideTmuxFunc: func() bool { return true },
				SwitchClientFunc: func(session string, window string) error { return nil },
				KillWindowFunc:   func(session string, window string) error { return nil },
			},
			WithCommonParams(defaultCP()),
		)

		check := RemoveCheck{
			Branch:       "feature",
			HasBranch:    true,
			HasWorktree:  true,
			WorktreePath: "/repo/.worktrees/feature",
			HasWindow:    true,
			IsActive:     true,
		}

		_, err := svc.ExecuteRemove(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, ensureTmuxCalled)
	})

	t.Run("kills session when no windows remain", func(t *testing.T) {
		var sessionKilled bool
		svc := newTestSvc(
			&git.ClientMock{
				RemoveWorktreeFunc:   func(path string) error { return nil },
				DeleteBranchFromFunc: func(dir string, name string) error { return nil },
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return true, nil },
				ListWindowsFunc: func(session string) ([]tmux.Window, error) {
					return nil, nil
				},
				KillWindowFunc:  func(session string, window string) error { return nil },
				KillSessionFunc: func(name string) error { sessionKilled = true; return nil },
			},
			WithCommonParams(defaultCP()),
		)

		check := RemoveCheck{
			Branch:       "feature",
			HasBranch:    true,
			HasWorktree:  true,
			WorktreePath: "/repo/.worktrees/feature",
			HasWindow:    true,
		}

		result, err := svc.ExecuteRemove(context.Background(), check)
		require.NoError(t, err)
		assert.True(t, sessionKilled)
		assert.True(t, result.SessionKilled)
	})

	t.Run("error from RemoveWorktree", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				RemoveWorktreeFunc: func(path string) error { return fmt.Errorf("remove failed") },
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
				KillWindowFunc: func(session string, window string) error { return nil },
			},
			WithCommonParams(defaultCP()),
		)

		check := RemoveCheck{
			Branch:       "feature",
			HasBranch:    true,
			HasWorktree:  true,
			WorktreePath: "/repo/.worktrees/feature",
		}

		_, err := svc.ExecuteRemove(context.Background(), check)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "removing worktree")
	})

	t.Run("error from DeleteBranch", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				DeleteBranchFromFunc: func(dir string, name string) error { return fmt.Errorf("delete failed") },
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
			},
			WithCommonParams(defaultCP()),
		)

		check := RemoveCheck{
			Branch:    "feature",
			HasBranch: true,
		}

		_, err := svc.ExecuteRemove(context.Background(), check)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deleting branch")
	})

	t.Run("EnsureTmux error on active window", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return true, nil },
				ListWindowsFunc: func(session string) ([]tmux.Window, error) {
					return nil, fmt.Errorf("tmux error")
				},
			},
			WithCommonParams(defaultCP()),
		)

		check := RemoveCheck{
			Branch:   "feature",
			IsActive: true,
		}

		_, err := svc.ExecuteRemove(context.Background(), check)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "switching to default branch")
	})

	t.Run("KillWindow error", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
				KillWindowFunc: func(session string, window string) error {
					return fmt.Errorf("kill failed")
				},
			},
			WithCommonParams(defaultCP()),
		)

		check := RemoveCheck{
			Branch:    "feature",
			HasWindow: true,
		}

		_, err := svc.ExecuteRemove(context.Background(), check)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "killing window")
	})

	t.Run("skips missing resources", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
			},
			WithCommonParams(defaultCP()),
		)

		check := RemoveCheck{Branch: "feature"}

		result, err := svc.ExecuteRemove(context.Background(), check)
		require.NoError(t, err)
		assert.False(t, result.BranchDeleted)
		assert.False(t, result.WorktreeRemoved)
		assert.False(t, result.WindowKilled)
	})
}
