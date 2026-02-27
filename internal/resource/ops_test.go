package resource

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

func TestEnsureWorktree(t *testing.T) {
	t.Run("default branch returns repo root", func(t *testing.T) {
		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main"}
		svc := newTestSvc(&git.ClientMock{}, stubTmux(), WithCommonParams(cp))

		path, created, err := svc.ensureWorktree("main")
		require.NoError(t, err)
		assert.Equal(t, "/repo", path)
		assert.False(t, created)
	})

	t.Run("existing worktree returns its path", func(t *testing.T) {
		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main"}
		svc := newTestSvc(&git.ClientMock{
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return []git.Worktree{
					{Path: "/repo/.worktrees/feature", Branch: "feature"},
				}, nil
			},
		}, stubTmux(), WithCommonParams(cp))

		path, created, err := svc.ensureWorktree("feature")
		require.NoError(t, err)
		assert.Equal(t, "/repo/.worktrees/feature", path)
		assert.False(t, created)
	})

	t.Run("creates worktree if missing", func(t *testing.T) {
		repoRoot := t.TempDir()
		var addedPath, addedBranch string
		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main"}
		svc := newTestSvc(&git.ClientMock{
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			AddWorktreeFunc: func(path string, branch string) error {
				addedPath = path
				addedBranch = branch
				return nil
			},
		}, stubTmux(), WithCommonParams(cp))

		path, created, err := svc.ensureWorktree("feature")
		require.NoError(t, err)
		assert.Contains(t, path, ".worktrees/feature")
		assert.True(t, created)
		assert.Equal(t, path, addedPath)
		assert.Equal(t, "feature", addedBranch)
	})

	t.Run("error from ListWorktrees", func(t *testing.T) {
		cp := CommonParams{RepoRoot: "/repo", WorktreeDir: ".worktrees", DefaultBranch: "main"}
		svc := newTestSvc(&git.ClientMock{
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, fmt.Errorf("git error")
			},
		}, stubTmux(), WithCommonParams(cp))

		_, _, err := svc.ensureWorktree("feature")
		assert.Error(t, err)
	})

	t.Run("error from AddWorktree", func(t *testing.T) {
		repoRoot := t.TempDir()
		cp := CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main"}
		svc := newTestSvc(&git.ClientMock{
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			AddWorktreeFunc: func(path string, branch string) error {
				return fmt.Errorf("add failed")
			},
		}, stubTmux(), WithCommonParams(cp))

		_, _, err := svc.ensureWorktree("feature")
		assert.Error(t, err)
	})
}

func TestEnsureTmux(t *testing.T) {
	t.Run("creates session when none exists", func(t *testing.T) {
		var sessionName, windowName, dir, capturedInitCmd string
		svc := NewService(nil, nil, &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return false, nil
			},
			NewSessionFunc: func(name string, wName string, d string, initCmd string) error {
				sessionName = name
				windowName = wName
				dir = d
				capturedInitCmd = initCmd
				return nil
			},
		})

		err := svc.ensureTmux("org/repo", "feature", "/repo/.worktrees/feature", "echo hi")
		require.NoError(t, err)
		assert.Equal(t, "org/repo", sessionName)
		assert.Equal(t, "feature", windowName)
		assert.Equal(t, "/repo/.worktrees/feature", dir)
		assert.Equal(t, "echo hi", capturedInitCmd)
	})

	t.Run("returns error when HasSession fails", func(t *testing.T) {
		svc := NewService(nil, nil, &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return false, fmt.Errorf("tmux not running")
			},
		})

		err := svc.ensureTmux("org/repo", "feature", "/path", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checking session")
	})

	t.Run("creates window when session exists but window does not", func(t *testing.T) {
		var newWindowName, capturedInitCmd string
		svc := NewService(nil, nil, &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return []tmux.Window{{Name: "main", Active: true}}, nil
			},
			NewWindowFunc: func(session string, name string, dir string, initCmd string) error {
				newWindowName = name
				capturedInitCmd = initCmd
				return nil
			},
		})

		err := svc.ensureTmux("org/repo", "feature", "/repo/.worktrees/feature", "npm install")
		require.NoError(t, err)
		assert.Equal(t, "feature", newWindowName)
		assert.Equal(t, "npm install", capturedInitCmd)
	})

	t.Run("sends cd when window exists and pane runs shell", func(t *testing.T) {
		var allKeys [][]string
		svc := NewService(nil, nil, &tmux.ClientMock{
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
				allKeys = append(allKeys, keys)
				return nil
			},
		})

		err := svc.ensureTmux("org/repo", "feature", "/repo/.worktrees/feature", "")
		require.NoError(t, err)
		require.Len(t, allKeys, 1)
		assert.Equal(t, "C-u", allKeys[0][0])
		assert.Contains(t, allKeys[0][1], "cd")
		assert.Equal(t, "Enter", allKeys[0][2])
	})

	t.Run("skips cd when pane runs non-shell process", func(t *testing.T) {
		var sendKeysCalled bool
		svc := NewService(nil, nil, &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return []tmux.Window{{Name: "feature", Active: false}}, nil
			},
			PaneCurrentCommandFunc: func(session string, window string) (string, error) {
				return "vim", nil
			},
			SendKeysFunc: func(session string, window string, keys ...string) error {
				sendKeysCalled = true
				return nil
			},
		})

		err := svc.ensureTmux("org/repo", "feature", "/repo/.worktrees/feature", "")
		require.NoError(t, err)
		assert.False(t, sendKeysCalled)
	})

	t.Run("skips cd when PaneCurrentCommand errors", func(t *testing.T) {
		var sendKeysCalled bool
		svc := NewService(nil, nil, &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return []tmux.Window{{Name: "feature", Active: false}}, nil
			},
			PaneCurrentCommandFunc: func(session string, window string) (string, error) {
				return "", fmt.Errorf("pane error")
			},
			SendKeysFunc: func(session string, window string, keys ...string) error {
				sendKeysCalled = true
				return nil
			},
		})

		err := svc.ensureTmux("org/repo", "feature", "/repo/.worktrees/feature", "")
		require.NoError(t, err)
		assert.False(t, sendKeysCalled)
	})

	t.Run("error from ListWindows", func(t *testing.T) {
		svc := NewService(nil, nil, &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWindowsFunc: func(session string) ([]tmux.Window, error) {
				return nil, fmt.Errorf("list error")
			},
		})

		err := svc.ensureTmux("org/repo", "feature", "/path", "")
		assert.Error(t, err)
	})
}

func TestIsShellCommand(t *testing.T) {
	svc := NewService(nil, nil, nil)
	shells := []string{"bash", "zsh", "fish", "sh", "dash", "ksh", "tcsh", "csh"}
	for _, s := range shells {
		assert.True(t, svc.isShellCommand(s), "should be shell: %s", s)
	}
	nonShells := []string{"vim", "nvim", "python", "node", "go", ""}
	for _, s := range nonShells {
		assert.False(t, svc.isShellCommand(s), "should not be shell: %s", s)
	}
}

func TestConnect(t *testing.T) {
	t.Run("switch client when inside tmux", func(t *testing.T) {
		var switched bool
		svc := NewService(nil, nil, &tmux.ClientMock{
			IsInsideTmuxFunc: func() bool { return true },
			SwitchClientFunc: func(session string, window string) error {
				switched = true
				return nil
			},
		})

		err := svc.connect("org/repo", "feature")
		require.NoError(t, err)
		assert.True(t, switched)
	})

	t.Run("attach session when outside tmux", func(t *testing.T) {
		var attached bool
		svc := NewService(nil, nil, &tmux.ClientMock{
			IsInsideTmuxFunc: func() bool { return false },
			AttachSessionFunc: func(session string, window string) error {
				attached = true
				return nil
			},
		})

		err := svc.connect("org/repo", "feature")
		require.NoError(t, err)
		assert.True(t, attached)
	})
}

func TestBuildInitCmd(t *testing.T) {
	t.Run("builds for loop with sh -c and appends exec shell", func(t *testing.T) {
		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			PostNewHooks: []string{"npm install", "echo done"},
		}))
		cmd := svc.buildInitCmd(true, "/bin/zsh")
		assert.Equal(t, "for __cmd in 'npm install' 'echo done'; do sh -c \"$__cmd\" || exit 1; done; exec '/bin/zsh'", cmd)
	})

	t.Run("single hook", func(t *testing.T) {
		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			PostNewHooks: []string{"npm install"},
		}))
		cmd := svc.buildInitCmd(true, "/bin/bash")
		assert.Equal(t, "for __cmd in 'npm install'; do sh -c \"$__cmd\" || exit 1; done; exec '/bin/bash'", cmd)
	})

	t.Run("empty hooks returns empty string", func(t *testing.T) {
		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			PostNewHooks: nil,
		}))
		cmd := svc.buildInitCmd(true, "/bin/zsh")
		assert.Empty(t, cmd)
	})

	t.Run("not created returns empty string", func(t *testing.T) {
		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			PostNewHooks: []string{"echo hello"},
		}))
		cmd := svc.buildInitCmd(false, "/bin/zsh")
		assert.Empty(t, cmd)
	})

	t.Run("falls back to sh when shell is empty", func(t *testing.T) {
		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			PostNewHooks: []string{"echo hello"},
		}))
		cmd := svc.buildInitCmd(true, "")
		assert.Equal(t, "for __cmd in 'echo hello'; do sh -c \"$__cmd\" || exit 1; done; exec 'sh'", cmd)
	})

	t.Run("escapes single quotes in hooks", func(t *testing.T) {
		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			PostNewHooks: []string{"echo 'hello'"},
		}))
		cmd := svc.buildInitCmd(true, "/bin/zsh")
		assert.Equal(t, "for __cmd in 'echo '\\''hello'\\'''; do sh -c \"$__cmd\" || exit 1; done; exec '/bin/zsh'", cmd)
	})
}

func TestCopyFiles(t *testing.T) {
	t.Run("copies file", func(t *testing.T) {
		repoRoot := t.TempDir()
		wtPath := filepath.Join(repoRoot, ".worktrees", "feat")
		require.NoError(t, os.MkdirAll(wtPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(repoRoot, ".env"), []byte("SECRET=1"), 0644))

		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			RepoRoot:  repoRoot,
			CopyFiles: []string{".env"},
		}))

		err := svc.copyFiles(wtPath)
		require.NoError(t, err)

		got, err := os.ReadFile(filepath.Join(wtPath, ".env"))
		require.NoError(t, err)
		assert.Equal(t, "SECRET=1", string(got))
	})

	t.Run("copies directory recursively", func(t *testing.T) {
		repoRoot := t.TempDir()
		wtPath := filepath.Join(repoRoot, ".worktrees", "feat")
		require.NoError(t, os.MkdirAll(wtPath, 0755))

		// Create .claude/settings.json in repo root
		claudeDir := filepath.Join(repoRoot, ".claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{"key":"val"}`), 0644))

		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			RepoRoot:  repoRoot,
			CopyFiles: []string{".claude"},
		}))

		err := svc.copyFiles(wtPath)
		require.NoError(t, err)

		got, err := os.ReadFile(filepath.Join(wtPath, ".claude", "settings.json"))
		require.NoError(t, err)
		assert.Equal(t, `{"key":"val"}`, string(got))
	})

	t.Run("skips non-existent entries", func(t *testing.T) {
		repoRoot := t.TempDir()
		wtPath := filepath.Join(repoRoot, ".worktrees", "feat")
		require.NoError(t, os.MkdirAll(wtPath, 0755))

		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			RepoRoot:  repoRoot,
			CopyFiles: []string{".env", "nonexistent.txt"},
		}))

		err := svc.copyFiles(wtPath)
		require.NoError(t, err)
	})

	t.Run("empty list does nothing", func(t *testing.T) {
		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			RepoRoot:  t.TempDir(),
			CopyFiles: nil,
		}))

		err := svc.copyFiles(t.TempDir())
		require.NoError(t, err)
	})

	t.Run("preserves file permissions", func(t *testing.T) {
		repoRoot := t.TempDir()
		wtPath := filepath.Join(repoRoot, ".worktrees", "feat")
		require.NoError(t, os.MkdirAll(wtPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "script.sh"), []byte("#!/bin/sh"), 0755))

		svc := NewService(nil, nil, nil, WithCommonParams(CommonParams{
			RepoRoot:  repoRoot,
			CopyFiles: []string{"script.sh"},
		}))

		err := svc.copyFiles(wtPath)
		require.NoError(t, err)

		info, err := os.Stat(filepath.Join(wtPath, "script.sh"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	})
}

func TestShellQuote(t *testing.T) {
	assert.Equal(t, "'/simple/path'", shellQuote("/simple/path"))
	assert.Equal(t, "'/path/with spaces'", shellQuote("/path/with spaces"))
	assert.Equal(t, "'/it'\\''s'", shellQuote("/it's"))
}

type testLogger struct {
	warnings []string
}

func (l *testLogger) Warn(msg string, args ...any) {
	l.warnings = append(l.warnings, fmt.Sprintf("%s %v", msg, args))
}

func TestBestEffort(t *testing.T) {
	t.Run("nil error does nothing", func(t *testing.T) {
		log := &testLogger{}
		svc := NewService(nil, nil, nil, WithLogger(log))
		svc.bestEffort("op", nil)
		assert.Empty(t, log.warnings)
	})

	t.Run("nil logger does not panic", func(t *testing.T) {
		svc := NewService(nil, nil, nil)
		assert.NotPanics(t, func() {
			svc.bestEffort("op", fmt.Errorf("fail"))
		})
	})

	t.Run("logs warning on error", func(t *testing.T) {
		log := &testLogger{}
		svc := NewService(nil, nil, nil, WithLogger(log))
		svc.bestEffort("TestOp", fmt.Errorf("something failed"))
		require.Len(t, log.warnings, 1)
		assert.Contains(t, log.warnings[0], "TestOp")
	})
}
