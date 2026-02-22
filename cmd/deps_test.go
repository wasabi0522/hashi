package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hashiexec "github.com/wasabi0522/hashi/internal/exec"
)

func TestResolveDepsWithExec(t *testing.T) {
	t.Run("git not found", func(t *testing.T) {
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				return fmt.Errorf("not found: %s", name)
			},
		}
		_, err := resolveDepsWithExec(e)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git")
	})

	t.Run("tmux not found", func(t *testing.T) {
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				if name == "tmux" {
					return fmt.Errorf("not found: tmux")
				}
				return nil
			},
			OutputFunc: func(name string, args ...string) (string, error) {
				return "", nil
			},
		}
		_, err := resolveDepsWithExec(e)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tmux")
	})

	t.Run("context resolve error", func(t *testing.T) {
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				return nil
			},
			OutputFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("not a git repo")
			},
		}
		_, err := resolveDepsWithExec(e)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		repoRoot := t.TempDir()
		gitDir := repoRoot + "/.git"
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				return nil
			},
			OutputFunc: func(name string, args ...string) (string, error) {
				if len(args) > 0 {
					switch args[0] {
					case "rev-parse":
						return gitDir, nil
					case "symbolic-ref":
						return "refs/remotes/origin/HEAD", nil
					case "remote":
						return "https://github.com/org/repo.git", nil
					}
				}
				return "", nil
			},
		}
		d, err := resolveDepsWithExec(e)
		require.NoError(t, err)
		assert.NotNil(t, d.git)
		assert.NotNil(t, d.tmux)
		assert.Equal(t, repoRoot, d.ctx.RepoRoot)
	})
}

func TestResolveGitDepsWithExec(t *testing.T) {
	t.Run("git not found", func(t *testing.T) {
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				return fmt.Errorf("not found")
			},
		}
		_, err := resolveGitDepsWithExec(e)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git")
	})

	t.Run("context resolve error", func(t *testing.T) {
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				return nil
			},
			OutputFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("not a git repo")
			},
		}
		_, err := resolveGitDepsWithExec(e)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		repoRoot := t.TempDir()
		gitDir := repoRoot + "/.git"
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				return nil
			},
			OutputFunc: func(name string, args ...string) (string, error) {
				if len(args) > 0 {
					switch args[0] {
					case "rev-parse":
						return gitDir, nil
					case "symbolic-ref":
						return "refs/remotes/origin/HEAD", nil
					case "remote":
						return "https://github.com/org/repo.git", nil
					}
				}
				return "", nil
			},
		}
		d, err := resolveGitDepsWithExec(e)
		require.NoError(t, err)
		assert.NotNil(t, d.git)
		assert.Equal(t, repoRoot, d.ctx.RepoRoot)
	})
}
