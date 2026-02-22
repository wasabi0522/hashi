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

func TestSwitchCmd(t *testing.T) {
	app := &App{}
	cmd := app.switchCmd(nil)
	assert.Equal(t, []string{"sw"}, cmd.Aliases)
}

func TestRunSwitch(t *testing.T) {
	t.Run("switches to existing branch", func(t *testing.T) {
		app := appWithDeps(&deps{
			git: &git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					return true, nil
				},
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return []git.Worktree{
						{Path: "/repo/.worktrees/feature", Branch: "feature"},
					}, nil
				},
			},
			tmux: &tmux.ClientMock{
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
				SwitchClientFunc: func(session string, window string) error {
					return nil
				},
			},
			ctx: &hashicontext.Context{
				RepoRoot:      "/repo",
				DefaultBranch: "main",
				SessionName:   "org/repo",
			},
			cfg: &config.Config{WorktreeDir: ".worktrees"},
		})

		cmd := &cobra.Command{}
		err := app.runSwitch(cmd, []string{"feature"})
		require.NoError(t, err)
	})

	t.Run("invalid branch name", func(t *testing.T) {
		app := appWithDeps(&deps{})
		_, err := executeCommand(t, app, "switch", "")
		assert.Error(t, err)
	})

	t.Run("deps error", func(t *testing.T) {
		app := appWithDepsError(fmt.Errorf("tmux not found"))

		cmd := &cobra.Command{}
		err := app.runSwitch(cmd, []string{"feature"})
		assert.Error(t, err)
	})

	t.Run("branch not found", func(t *testing.T) {
		app := appWithDeps(&deps{
			git: &git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					return false, nil
				},
			},
			tmux: &tmux.ClientMock{},
			ctx: &hashicontext.Context{
				RepoRoot:      "/repo",
				DefaultBranch: "main",
				SessionName:   "org/repo",
			},
			cfg: &config.Config{WorktreeDir: ".worktrees"},
		})

		cmd := &cobra.Command{}
		err := app.runSwitch(cmd, []string{"nonexistent"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}
