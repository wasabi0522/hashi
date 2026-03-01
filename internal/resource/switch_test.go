package resource

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

func TestSwitch(t *testing.T) {
	t.Run("switches to existing branch with worktree", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("feature"),
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return []git.Worktree{
					{Path: "/repo/.worktrees/feature", Branch: "feature"},
				}, nil
			},
		}
		tm := &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return []tmux.Window{{Name: "feature", Active: false}}, nil
			},
			PaneCurrentCommandFunc: func(session string, window string) (string, error) {
				return "zsh", nil
			},
			SendKeysFunc: func(session string, window string, keys ...string) error {
				return nil
			},
			IsInsideTmuxFunc: func() bool { return true },
			SwitchClientFunc: func(session string, window string) error { return nil },
		}

		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{
			Branch: "feature",
		})
		require.NoError(t, err)
	})

	t.Run("errors when branch does not exist", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists(), // nothing exists
		}

		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{
			Branch: "nonexistent",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("creates worktree if missing", func(t *testing.T) {
		repoRoot := t.TempDir()
		var addedBranch string
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("feature"),
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			AddWorktreeFunc: func(path string, branch string) error {
				addedBranch = branch
				return nil
			},
		}
		tm := stubTmuxInside()

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{
			Branch: "feature",
		})
		require.NoError(t, err)
		assert.Equal(t, "feature", addedBranch)
	})

	t.Run("BranchExists error", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc: func(name string) (bool, error) {
				return false, fmt.Errorf("git error")
			},
		}

		svc := newTestSvc(g, stubTmux())
		_, err := svc.Switch(context.Background(), SwitchParams{
			Branch: "feature",
		})
		assert.Error(t, err)
	})

	t.Run("passes initCmd to tmux when worktree created", func(t *testing.T) {
		repoRoot := t.TempDir()
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("feature"),
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			AddWorktreeFunc: func(path string, branch string) error {
				_ = os.MkdirAll(path, 0755)
				return nil
			},
		}
		var capturedInitCmd string
		tm := &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) { return false, nil },
			NewSessionFunc: func(name string, windowName string, dir string, initCmd string) error {
				capturedInitCmd = initCmd
				return nil
			},
			IsInsideTmuxFunc: func() bool { return true },
			SwitchClientFunc: func(session string, window string) error { return nil },
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo", Shell: "/bin/bash", PostNewHooks: []string{"echo hello"}}
		svc := NewService(g, tm, WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{
			Branch: "feature",
		})
		require.NoError(t, err)

		assert.Contains(t, capturedInitCmd, "echo hello")
		assert.Contains(t, capturedInitCmd, "sh -c")
		assert.Contains(t, capturedInitCmd, "exec '/bin/bash'")
	})

	t.Run("EnsureTmux error", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("feature"),
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return []git.Worktree{
					{Path: "/repo/.worktrees/feature", Branch: "feature"},
				}, nil
			},
		}
		tm := &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return nil, fmt.Errorf("tmux error")
			},
		}

		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{
			Branch: "feature",
		})
		assert.Error(t, err)
	})

	t.Run("switches to default branch", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc:  mockBranchExists("main"),
			CurrentBranchFunc: func(dir string) (string, error) { return "main", nil },
		}
		tm := &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return []tmux.Window{{Name: "main", Active: false}}, nil
			},
			PaneCurrentCommandFunc: func(session string, window string) (string, error) {
				return "bash", nil
			},
			SendKeysFunc: func(session string, window string, keys ...string) error {
				return nil
			},
			IsInsideTmuxFunc: func() bool { return true },
			SwitchClientFunc: func(session string, window string) error { return nil },
		}

		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{
			Branch: "main",
		})
		require.NoError(t, err)
	})

	t.Run("auto-switches repo root to default branch when clean", func(t *testing.T) {
		var switchedTo string
		g := &git.ClientMock{
			BranchExistsFunc:  mockBranchExists("main"),
			CurrentBranchFunc: func(dir string) (string, error) { return "feature", nil },
			HasUncommittedChangesFunc: func(worktreePath string) (bool, error) {
				return false, nil
			},
			SwitchBranchFunc: func(dir, branch string) error {
				switchedTo = branch
				return nil
			},
		}
		tm := &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) { return true, nil },
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return []tmux.Window{{Name: "main", Active: false}}, nil
			},
			PaneCurrentCommandFunc: func(session, window string) (string, error) { return "zsh", nil },
			SendKeysFunc:           func(session, window string, keys ...string) error { return nil },
			IsInsideTmuxFunc:       func() bool { return true },
			SwitchClientFunc:       func(session, window string) error { return nil },
		}

		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{Branch: "main"})
		require.NoError(t, err)
		assert.Equal(t, "main", switchedTo)
	})

	t.Run("errors when repo root has uncommitted changes on wrong branch", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc:  mockBranchExists("main"),
			CurrentBranchFunc: func(dir string) (string, error) { return "feature", nil },
			HasUncommittedChangesFunc: func(worktreePath string) (bool, error) {
				return true, nil
			},
		}

		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{Branch: "main"})
		require.Error(t, err)

		var mismatchErr *RepoRootBranchMismatchError
		assert.ErrorAs(t, err, &mismatchErr)
		assert.Equal(t, "main", mismatchErr.Expected)
		assert.Equal(t, "feature", mismatchErr.Actual)
	})

	t.Run("errors when git switch fails on clean repo root", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc:  mockBranchExists("main"),
			CurrentBranchFunc: func(dir string) (string, error) { return "feature", nil },
			HasUncommittedChangesFunc: func(worktreePath string) (bool, error) {
				return false, nil
			},
			SwitchBranchFunc: func(dir, branch string) error {
				return fmt.Errorf("switch failed")
			},
		}

		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{Branch: "main"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "switching repo root")
	})

	t.Run("EnsureWorktree error", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("feature"),
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, fmt.Errorf("worktree error")
			},
		}

		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.Switch(context.Background(), SwitchParams{
			Branch: "feature",
		})
		assert.Error(t, err)
	})
}
