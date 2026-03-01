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

func TestRenameCmd(t *testing.T) {
	app := &App{}
	cmd := app.renameCmd(nil)
	assert.Equal(t, []string{"mv"}, cmd.Aliases)
}

func TestRunRename(t *testing.T) {
	t.Run("renames branch successfully", func(t *testing.T) {
		repoRoot := t.TempDir()
		app := appWithDeps(&deps{
			git: &git.ClientMock{
				ListBranchesFunc: func() ([]string, error) {
					return []string{"old"}, nil
				},
				RenameBranchFunc: func(old string, newName string) error {
					return nil
				},
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return nil, nil
				},
				AddWorktreeFunc: func(path string, branch string) error {
					return nil
				},
			},
			tmux: &tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) {
					return false, nil
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
		err := app.runRename(cmd, []string{"old", "new"})
		require.NoError(t, err)
	})

	t.Run("invalid old name", func(t *testing.T) {
		app := appWithDeps(&deps{})
		_, err := executeCommand(t, app, "rename", "", "new")
		assert.Error(t, err)
	})

	t.Run("invalid new name", func(t *testing.T) {
		app := appWithDeps(&deps{})
		_, err := executeCommand(t, app, "rename", "old", "")
		assert.Error(t, err)
	})

	t.Run("deps error", func(t *testing.T) {
		app := appWithDepsError(fmt.Errorf("no deps"))

		cmd := &cobra.Command{}
		err := app.runRename(cmd, []string{"old", "new"})
		assert.Error(t, err)
	})

	t.Run("rename error", func(t *testing.T) {
		app := appWithDeps(&deps{
			git: &git.ClientMock{
				ListBranchesFunc: func() ([]string, error) {
					return []string{"old", "existing"}, nil
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
		err := app.runRename(cmd, []string{"old", "existing"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}
