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

func TestNew(t *testing.T) {
	t.Run("creates new branch with worktree and window", func(t *testing.T) {
		repoRoot := t.TempDir()
		var addedWT, addedBranch, addedBase string
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("main"),
			AddWorktreeNewBranchFunc: func(path string, branch string, base string) error {
				addedWT = path
				addedBranch = branch
				addedBase = base
				return nil
			},
		}
		tm := stubTmuxInside()

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
		})
		require.NoError(t, err)
		assert.Contains(t, addedWT, ".worktrees/feature")
		assert.Equal(t, "feature", addedBranch)
		assert.Equal(t, "main", addedBase)
	})

	t.Run("creates new branch with explicit base", func(t *testing.T) {
		repoRoot := t.TempDir()
		var addedBase string
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("main", "develop"),
			AddWorktreeNewBranchFunc: func(path string, branch string, base string) error {
				addedBase = base
				return nil
			},
		}
		tm := stubTmuxInside()

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
			Base:   "develop",
		})
		require.NoError(t, err)
		assert.Equal(t, "develop", addedBase)
	})

	t.Run("errors when base specified for existing branch", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("feature", "develop"),
		}

		cp := CommonParams{DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
			Base:   "develop",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify base branch")
	})

	t.Run("errors when base branch does not exist", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists(), // nothing exists
		}

		cp := CommonParams{DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
			Base:   "nonexistent",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("BranchExists error", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc: func(name string) (bool, error) {
				return false, fmt.Errorf("git error")
			},
		}

		cp := CommonParams{DefaultBranch: "main"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
		})
		assert.Error(t, err)
	})

	t.Run("rolls back on tmux failure for new branch", func(t *testing.T) {
		repoRoot := t.TempDir()
		var removedWT, deletedBranch string
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("main"),
			AddWorktreeNewBranchFunc: func(path string, branch string, base string) error {
				return nil
			},
			RemoveWorktreeFunc: func(path string) error {
				removedWT = path
				return nil
			},
			DeleteBranchFunc: func(name string) error {
				deletedBranch = name
				return nil
			},
		}
		tm := &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return false, nil
			},
			NewSessionFunc: func(name string, windowName string, dir string, initCmd string) error {
				return fmt.Errorf("tmux error")
			},
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
		})
		assert.Error(t, err)
		assert.Contains(t, removedWT, "feature")
		assert.Equal(t, "feature", deletedBranch)
	})

	t.Run("existing branch ensures worktree and tmux", func(t *testing.T) {
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("feature", "main"),
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
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
		})
		require.NoError(t, err)
	})

	t.Run("existing branch with tmux failure does not rollback branch", func(t *testing.T) {
		repoRoot := t.TempDir()
		var removedWT bool
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("feature", "main"),
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			AddWorktreeFunc: func(path string, branch string) error {
				return nil
			},
			RemoveWorktreeFunc: func(path string) error {
				removedWT = true
				return nil
			},
		}
		tm := &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return false, nil
			},
			NewSessionFunc: func(name string, windowName string, dir string, initCmd string) error {
				return fmt.Errorf("tmux error")
			},
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
		})
		assert.Error(t, err)
		assert.True(t, removedWT, "worktree should be rolled back")
	})

	t.Run("passes initCmd to tmux when worktree created for new branch", func(t *testing.T) {
		repoRoot := t.TempDir()
		t.Setenv("SHELL", "/bin/zsh")
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("main"),
			AddWorktreeNewBranchFunc: func(path string, branch string, base string) error {
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

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo", PostNewHooks: []string{"echo hello"}}
		svc := NewService(nil, g, tm, WithCommonParams(cp))
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
		})
		require.NoError(t, err)

		assert.Contains(t, capturedInitCmd, "echo hello")
		assert.Contains(t, capturedInitCmd, "sh -c")
		assert.Contains(t, capturedInitCmd, "exec '/bin/zsh'")
	})

	t.Run("AddWorktreeNewBranch error", func(t *testing.T) {
		repoRoot := t.TempDir()
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("main"),
			AddWorktreeNewBranchFunc: func(path string, branch string, base string) error {
				return fmt.Errorf("worktree add failed")
			},
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.New(context.Background(), NewParams{
			Branch: "feature",
		})
		assert.Error(t, err)
	})
}
