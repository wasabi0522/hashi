package cmd

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/config"
	hashicontext "github.com/wasabi0522/hashi/internal/context"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

func TestNewCmd(t *testing.T) {
	app := &App{}
	cmd := app.newCmd(nil)
	assert.Equal(t, []string{"n"}, cmd.Aliases)
}

func TestRunNew(t *testing.T) {
	t.Run("creates new branch", func(t *testing.T) {
		repoRoot := t.TempDir()
		app := appWithDeps(&deps{
			git: &git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					if name == "feature" {
						return false, nil
					}
					return true, nil
				},
				AddWorktreeNewBranchFunc: func(path string, branch string, base string) error {
					return nil
				},
			},
			tmux: &tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) {
					return false, nil
				},
				NewSessionFunc: func(name string, windowName string, dir string, initCmd string) error {
					return nil
				},
				IsInsideTmuxFunc: func() bool { return true },
				SwitchClientFunc: func(session string, window string) error {
					return nil
				},
			},
			ctx: &hashicontext.Context{
				RepoRoot:      repoRoot,
				DefaultBranch: "main",
				SessionName:   "org/repo",
			},
			cfg: &config.Config{WorktreeDir: ".worktrees"},
		})

		cmd := &cobra.Command{}
		err := app.runNew(cmd, []string{"feature"})
		require.NoError(t, err)
	})

	t.Run("with explicit base", func(t *testing.T) {
		repoRoot := t.TempDir()
		var usedBase string
		app := appWithDeps(&deps{
			git: &git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					if name == "feature" {
						return false, nil
					}
					return true, nil
				},
				AddWorktreeNewBranchFunc: func(path string, branch string, base string) error {
					usedBase = base
					return nil
				},
			},
			tmux: &tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) {
					return false, nil
				},
				NewSessionFunc: func(name string, windowName string, dir string, initCmd string) error {
					return nil
				},
				IsInsideTmuxFunc: func() bool { return false },
				AttachSessionFunc: func(session string, window string) error {
					return nil
				},
			},
			ctx: &hashicontext.Context{
				RepoRoot:      repoRoot,
				DefaultBranch: "main",
				SessionName:   "org/repo",
			},
			cfg: &config.Config{WorktreeDir: ".worktrees"},
		})

		cmd := &cobra.Command{}
		err := app.runNew(cmd, []string{"feature", "develop"})
		require.NoError(t, err)
		assert.Equal(t, "develop", usedBase)
	})

	t.Run("deps error", func(t *testing.T) {
		app := appWithDepsError(fmt.Errorf("git not found"))

		cmd := &cobra.Command{}
		err := app.runNew(cmd, []string{"feature"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "git not found")
	})

	t.Run("invalid branch name", func(t *testing.T) {
		app := appWithDeps(&deps{})
		_, err := executeCommand(t, app, "new", "")
		assert.Error(t, err)
	})

	t.Run("invalid base name", func(t *testing.T) {
		app := appWithDeps(&deps{})
		_, err := executeCommand(t, app, "new", "feature", "")
		assert.Error(t, err)
	})

	t.Run("resource.New error", func(t *testing.T) {
		app := appWithDeps(&deps{
			git: &git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					return false, fmt.Errorf("git error")
				},
			},
			tmux: &tmux.ClientMock{},
			ctx: &hashicontext.Context{
				RepoRoot:      t.TempDir(),
				DefaultBranch: "main",
				SessionName:   "org/repo",
			},
			cfg: &config.Config{WorktreeDir: ".worktrees"},
		})

		cmd := &cobra.Command{}
		err := app.runNew(cmd, []string{"feature"})
		assert.Error(t, err)
	})
}
