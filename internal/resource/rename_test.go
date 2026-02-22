package resource

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

func TestRename(t *testing.T) {
	t.Run("errors when renaming default branch", func(t *testing.T) {
		cp := CommonParams{DefaultBranch: "main"}
		svc := newTestSvc(&git.ClientMock{}, stubTmux(), WithCommonParams(cp))

		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "main",
			New: "trunk",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot rename default branch")
	})

	t.Run("errors when old branch does not exist", func(t *testing.T) {
		cp := CommonParams{DefaultBranch: "main"}
		svc := newTestSvc(
			&git.ClientMock{BranchExistsFunc: mockBranchExists()},
			stubTmux(),
			WithCommonParams(cp),
		)

		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("errors when new branch already exists", func(t *testing.T) {
		cp := CommonParams{DefaultBranch: "main"}
		svc := newTestSvc(
			&git.ClientMock{BranchExistsFunc: mockBranchExists("old", "existing")},
			stubTmux(),
			WithCommonParams(cp),
		)

		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "existing",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("BranchExists error on old", func(t *testing.T) {
		cp := CommonParams{DefaultBranch: "main"}
		svc := newTestSvc(
			&git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					return false, fmt.Errorf("git error")
				},
			},
			stubTmux(),
			WithCommonParams(cp),
		)

		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		assert.Error(t, err)
	})

	t.Run("BranchExists error on new", func(t *testing.T) {
		callCount := 0
		cp := CommonParams{DefaultBranch: "main"}
		svc := newTestSvc(
			&git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					callCount++
					if callCount == 1 {
						return true, nil // old exists
					}
					return false, fmt.Errorf("git error")
				},
			},
			stubTmux(),
			WithCommonParams(cp),
		)

		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		assert.Error(t, err)
	})

	t.Run("renames branch and creates worktree when none exists", func(t *testing.T) {
		repoRoot := t.TempDir()
		var renamedOld, renamedNew string
		var addedWT string
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("old"),
			RenameBranchFunc: func(old string, newName string) error {
				renamedOld = old
				renamedNew = newName
				return nil
			},
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			AddWorktreeFunc: func(path string, branch string) error {
				addedWT = path
				return nil
			},
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		require.NoError(t, err)
		assert.Equal(t, "old", renamedOld)
		assert.Equal(t, "new", renamedNew)
		assert.Contains(t, addedWT, ".worktrees/new")
	})

	t.Run("moves existing worktree", func(t *testing.T) {
		repoRoot := t.TempDir()
		oldPath := filepath.Join(repoRoot, ".worktrees", "old")
		require.NoError(t, os.MkdirAll(oldPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(oldPath, "marker.txt"), []byte("x"), 0644))

		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("old"),
			RenameBranchFunc: func(old string, newName string) error {
				return nil
			},
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return []git.Worktree{
					{Path: oldPath, Branch: "new"},
				}, nil
			},
			RepairWorktreesFunc: func() error {
				return nil
			},
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		require.NoError(t, err)

		newPath := filepath.Join(repoRoot, ".worktrees", "new")
		_, err = os.Stat(filepath.Join(newPath, "marker.txt"))
		require.NoError(t, err, "marker file should exist in new path")

		_, err = os.Stat(oldPath)
		assert.True(t, os.IsNotExist(err), "old path should not exist")
	})

	t.Run("rolls back worktree move on repair failure", func(t *testing.T) {
		repoRoot := t.TempDir()
		oldPath := filepath.Join(repoRoot, ".worktrees", "old")
		require.NoError(t, os.MkdirAll(oldPath, 0755))

		var branchRolledBack bool
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("old"),
			RenameBranchFunc: func(old string, newName string) error {
				if old == "new" && newName == "old" {
					branchRolledBack = true
				}
				return nil
			},
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return []git.Worktree{
					{Path: oldPath, Branch: "new"},
				}, nil
			},
			RepairWorktreesFunc: func() error {
				return fmt.Errorf("repair failed")
			},
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repairing worktrees")
		assert.True(t, branchRolledBack)
	})

	t.Run("rolls back branch rename on worktree add failure", func(t *testing.T) {
		repoRoot := t.TempDir()
		var rolledBack bool
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("old"),
			RenameBranchFunc: func(old string, newName string) error {
				if old == "new" && newName == "old" {
					rolledBack = true
				}
				return nil
			},
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			AddWorktreeFunc: func(path string, branch string) error {
				return fmt.Errorf("worktree add failed")
			},
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, stubTmux(), WithCommonParams(cp))
		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		assert.Error(t, err)
		assert.True(t, rolledBack)
	})

	t.Run("renames tmux window when session and window exist", func(t *testing.T) {
		repoRoot := t.TempDir()
		var renamedWindow bool
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("old"),
			RenameBranchFunc: func(old string, newName string) error {
				return nil
			},
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			AddWorktreeFunc: func(path string, branch string) error {
				return nil
			},
		}
		tm := &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return []tmux.Window{{Name: "old", Active: false}}, nil
			},
			RenameWindowFunc: func(session string, old string, newName string) error {
				renamedWindow = true
				return nil
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

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		require.NoError(t, err)
		assert.True(t, renamedWindow)
	})

	t.Run("creates new tmux window when old window not found", func(t *testing.T) {
		repoRoot := t.TempDir()
		var newWindowCreated bool
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("old"),
			RenameBranchFunc: func(old string, newName string) error {
				return nil
			},
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			AddWorktreeFunc: func(path string, branch string) error {
				return nil
			},
		}
		tm := &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return []tmux.Window{{Name: "main", Active: true}}, nil
			},
			NewWindowFunc: func(session string, name string, dir string, initCmd string) error {
				newWindowCreated = true
				return nil
			},
			IsInsideTmuxFunc: func() bool { return true },
			SwitchClientFunc: func(session string, window string) error { return nil },
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo"}
		svc := newTestSvc(g, tm, WithCommonParams(cp))
		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		require.NoError(t, err)
		assert.True(t, newWindowCreated)
	})

	t.Run("RenameBranch error", func(t *testing.T) {
		cp := CommonParams{DefaultBranch: "main"}
		svc := newTestSvc(
			&git.ClientMock{
				BranchExistsFunc: mockBranchExists("old"),
				RenameBranchFunc: func(old string, newName string) error {
					return fmt.Errorf("rename failed")
				},
			},
			stubTmux(),
			WithCommonParams(cp),
		)

		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		assert.Error(t, err)
	})

	t.Run("passes initCmd to tmux when worktree newly created", func(t *testing.T) {
		repoRoot := t.TempDir()
		t.Setenv("SHELL", "/bin/zsh")
		g := &git.ClientMock{
			BranchExistsFunc: mockBranchExists("old"),
			RenameBranchFunc: func(old string, newName string) error {
				return nil
			},
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
			HasSessionFunc: func(name string) (bool, error) { return true, nil },
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return []tmux.Window{{Name: "main", Active: true}}, nil
			},
			NewWindowFunc: func(session string, name string, dir string, initCmd string) error {
				capturedInitCmd = initCmd
				return nil
			},
			IsInsideTmuxFunc: func() bool { return true },
			SwitchClientFunc: func(session string, window string) error { return nil },
		}

		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "org/repo", PostNewHooks: []string{"echo hello"}}
		svc := NewService(nil, g, tm, WithCommonParams(cp))
		_, err := svc.Rename(context.Background(), RenameParams{
			Old: "old",
			New: "new",
		})
		require.NoError(t, err)

		assert.Contains(t, capturedInitCmd, "echo hello")
		assert.Contains(t, capturedInitCmd, "exec /bin/zsh")
	})
}
