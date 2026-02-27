package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/config"
	hashicontext "github.com/wasabi0522/hashi/internal/context"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/resource"
	"github.com/wasabi0522/hashi/internal/tmux"
	"github.com/wasabi0522/hashi/internal/ui"
)

func TestRemoveCmd(t *testing.T) {
	app := &App{}
	cmd := app.removeCmd(nil)
	assert.Equal(t, []string{"rm"}, cmd.Aliases)
}

func TestBuildRemovePrompt(t *testing.T) {
	ui.SetNoColor(true)
	t.Cleanup(func() { ui.SetNoColor(false) })

	tests := []struct {
		name     string
		check    resource.RemoveCheck
		contains []string
		exact    string
	}{
		{
			name:  "branch only",
			check: resource.RemoveCheck{Branch: "feature", HasBranch: true},
			exact: "Remove 'feature'? (branch)",
		},
		{
			name:  "all resources",
			check: resource.RemoveCheck{Branch: "feature", HasBranch: true, HasWorktree: true, HasWindow: true},
			exact: "Remove 'feature'? (branch, worktree, window)",
		},
		{
			name:     "uncommitted changes warning on separate line",
			check:    resource.RemoveCheck{Branch: "feature", HasBranch: true, HasWorktree: true, HasUncommitted: true},
			contains: []string{"(branch, worktree)", "\n  ⚠ has uncommitted changes"},
		},
		{
			name:     "unmerged warning on separate line",
			check:    resource.RemoveCheck{Branch: "feature", HasBranch: true, IsUnmerged: true},
			contains: []string{"(branch)", "\n  ⚠ has unmerged commits"},
		},
		{
			name:  "both warnings on separate lines",
			check: resource.RemoveCheck{Branch: "feature", HasBranch: true, HasWorktree: true, HasWindow: true, HasUncommitted: true, IsUnmerged: true},
			contains: []string{
				"(branch, worktree, window)",
				"\n  ⚠ has uncommitted changes",
				"\n  ⚠ has unmerged commits",
			},
		},
		{
			name:  "orphaned worktree and window",
			check: resource.RemoveCheck{Branch: "orphan", HasWorktree: true, HasWindow: true},
			exact: "Remove 'orphan'? (worktree, window)",
		},
		{
			name:  "orphaned window only",
			check: resource.RemoveCheck{Branch: "orphan", HasWindow: true},
			exact: "Remove 'orphan'? (window)",
		},
		{
			name:  "orphaned worktree only",
			check: resource.RemoveCheck{Branch: "orphan", HasWorktree: true},
			exact: "Remove 'orphan'? (worktree)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := buildRemovePrompt(tt.check)
			if tt.exact != "" {
				assert.Equal(t, tt.exact, msg)
			}
			for _, s := range tt.contains {
				assert.Contains(t, msg, s)
			}
		})
	}
}

func TestConfirmPrompt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"y", "y\n", true},
		{"yes", "yes\n", true},
		{"YES (case insensitive)", "YES\n", true},
		{"no", "n\n", false},
		{"empty input", "\n", false},
		{"EOF (no input)", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(tt.input))
			var buf bytes.Buffer
			cmd.SetErr(&buf)
			assert.Equal(t, tt.want, confirmPrompt(cmd, "Delete?"))
			assert.Contains(t, buf.String(), "y/N [N]")
		})
	}
}

func defaultRemoveDeps(t *testing.T) *deps {
	t.Helper()
	return &deps{
		git: &git.ClientMock{
			BranchExistsFunc: func(name string) (bool, error) {
				return true, nil
			},
			ListWorktreesFunc: func() ([]git.Worktree, error) {
				return nil, nil
			},
			IsMergedFunc: func(branch string, base string) (bool, error) {
				return true, nil
			},
			DeleteBranchFromFunc: func(dir string, name string) error {
				return nil
			},
		},
		tmux: &tmux.ClientMock{
			HasSessionFunc: func(name string) (bool, error) {
				return false, nil
			},
		},
		ctx: &hashicontext.Context{
			RepoRoot:      t.TempDir(),
			DefaultBranch: "main",
			SessionName:   "org/repo",
		},
		cfg: &config.Config{WorktreeDir: ".worktrees"},
	}
}

func TestRunRemove(t *testing.T) {
	t.Run("force remove single branch", func(t *testing.T) {
		d := defaultRemoveDeps(t)
		app := appWithDeps(d)

		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)
		err := app.runRemove(cmd, []string{"feature"}, true)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Removed")
	})

	t.Run("non-force confirm yes", func(t *testing.T) {
		d := defaultRemoveDeps(t)
		app := appWithDeps(d)

		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader("y\n"))
		err := app.runRemove(cmd, []string{"feature"}, false)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Removed")
	})

	t.Run("non-force confirm no skips", func(t *testing.T) {
		d := defaultRemoveDeps(t)
		app := appWithDeps(d)

		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader("n\n"))
		err := app.runRemove(cmd, []string{"feature"}, false)
		require.NoError(t, err)
		assert.NotContains(t, buf.String(), "Removed")
	})

	t.Run("invalid branch name", func(t *testing.T) {
		d := defaultRemoveDeps(t)
		app := appWithDeps(d)
		_, err := executeCommand(t, app, "remove", "-f", "")
		assert.Error(t, err)
	})

	t.Run("cannot remove default branch", func(t *testing.T) {
		d := defaultRemoveDeps(t)
		app := appWithDeps(d)

		cmd := &cobra.Command{}
		err := app.runRemove(cmd, []string{"main"}, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot remove default branch")
	})

	t.Run("deps error", func(t *testing.T) {
		app := appWithDepsError(fmt.Errorf("no git"))

		cmd := &cobra.Command{}
		err := app.runRemove(cmd, []string{"feature"}, true)
		assert.Error(t, err)
	})

	t.Run("PrepareRemove error", func(t *testing.T) {
		app := appWithDeps(&deps{
			git: &git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					return false, fmt.Errorf("git error")
				},
			},
			tmux: &tmux.ClientMock{},
			ctx:  &hashicontext.Context{DefaultBranch: "main"},
			cfg:  &config.Config{},
		})

		cmd := &cobra.Command{}
		err := app.runRemove(cmd, []string{"feature"}, true)
		assert.Error(t, err)
	})

	t.Run("ExecuteRemove error", func(t *testing.T) {
		app := appWithDeps(&deps{
			git: &git.ClientMock{
				BranchExistsFunc: func(name string) (bool, error) {
					return true, nil
				},
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return nil, nil
				},
				IsMergedFunc: func(branch string, base string) (bool, error) {
					return true, nil
				},
				DeleteBranchFromFunc: func(dir string, name string) error {
					return fmt.Errorf("delete failed")
				},
			},
			tmux: &tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) {
					return false, nil
				},
			},
			ctx: &hashicontext.Context{
				RepoRoot:      t.TempDir(),
				DefaultBranch: "main",
				SessionName:   "org/repo",
			},
			cfg: &config.Config{WorktreeDir: ".worktrees"},
		})

		cmd := &cobra.Command{}
		err := app.runRemove(cmd, []string{"feature"}, true)
		assert.Error(t, err)
	})
}
