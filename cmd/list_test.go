package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/config"
	hashicontext "github.com/wasabi0522/hashi/internal/context"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/resource"
	"github.com/wasabi0522/hashi/internal/tmux"
)

func TestListCmd(t *testing.T) {
	app := &App{}
	cmd := app.listCmd()
	assert.Equal(t, []string{"ls"}, cmd.Aliases)
}

func TestPrintJSON(t *testing.T) {
	states := []resource.State{
		{Branch: "feature", Worktree: "/repo/.worktrees/feature", Window: true, Active: true, Status: resource.StatusOK},
		{Branch: "orphan", Window: true, Active: false, Status: resource.StatusOrphanedWindow},
	}

	var buf bytes.Buffer
	err := printJSON(&buf, states)
	require.NoError(t, err)

	var decoded []resource.State
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Len(t, decoded, 2)
	assert.Equal(t, "feature", decoded[0].Branch)
	assert.Equal(t, resource.StatusOrphanedWindow, decoded[1].Status)
}

func TestPrintTable(t *testing.T) {
	t.Run("all status types", func(t *testing.T) {
		states := []resource.State{
			{Branch: "main", Worktree: "/repo/.worktrees/main", Window: true, Active: true, Status: resource.StatusOK},
			{Branch: "feat", Window: true, Active: false, Status: resource.StatusWorktreeMissing},
			{Branch: "orphan-win", Window: true, Active: false, Status: resource.StatusOrphanedWindow},
			{Branch: "orphan-wt", Worktree: "/repo/.worktrees/main", Window: false, Active: false, Status: resource.StatusOrphanedWorktree},
		}

		var buf bytes.Buffer
		printTable(&buf, states)
		out := buf.String()
		assert.Contains(t, out, "main")
		assert.Contains(t, out, "feat")
		assert.Contains(t, out, "orphan-win")
		assert.Contains(t, out, "orphan-wt")
	})

	t.Run("empty states", func(t *testing.T) {
		var buf bytes.Buffer
		printTable(&buf, nil)
		assert.Contains(t, buf.String(), "BRANCH")
	})
}

func newListDeps(g git.Client, tm tmux.Client, ctx *hashicontext.Context) *deps {
	return &deps{
		git:  g,
		tmux: tm,
		ctx:  ctx,
		cfg:  &config.Config{WorktreeDir: ".worktrees"},
	}
}

func TestRunList(t *testing.T) {
	t.Run("success with table output", func(t *testing.T) {
		d := newListDeps(
			&git.ClientMock{
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return []git.Worktree{
						{Path: "/repo", Branch: "main", IsMain: true},
					}, nil
				},
				ListBranchesFunc: func() ([]string, error) {
					return []string{"main"}, nil
				},
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) {
					return false, nil
				},
			},
			&hashicontext.Context{
				RepoRoot:      "/repo",
				DefaultBranch: "main",
				SessionName:   "org/repo",
			},
		)
		app := appWithDeps(d)

		cmd := &cobra.Command{}
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		err := app.runList(cmd, false)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "main")
	})

	t.Run("success with json output", func(t *testing.T) {
		d := newListDeps(
			&git.ClientMock{
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return []git.Worktree{
						{Path: "/repo", Branch: "main", IsMain: true},
					}, nil
				},
				ListBranchesFunc: func() ([]string, error) {
					return []string{"main"}, nil
				},
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) {
					return false, nil
				},
			},
			&hashicontext.Context{
				RepoRoot:      "/repo",
				DefaultBranch: "main",
				SessionName:   "org/repo",
			},
		)
		app := appWithDeps(d)

		cmd := &cobra.Command{}
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		err := app.runList(cmd, true)
		require.NoError(t, err)

		var decoded []resource.State
		require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
		assert.Len(t, decoded, 1)
		assert.Equal(t, "main", decoded[0].Branch)
	})

	t.Run("CollectState error", func(t *testing.T) {
		d := newListDeps(
			&git.ClientMock{
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return nil, fmt.Errorf("list error")
				},
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) {
					return false, nil
				},
			},
			&hashicontext.Context{SessionName: "org/repo"},
		)
		app := appWithDeps(d)

		cmd := &cobra.Command{}
		err := app.runList(cmd, false)
		assert.Error(t, err)
	})

	t.Run("deps error", func(t *testing.T) {
		app := appWithDepsError(fmt.Errorf("no git"))

		cmd := &cobra.Command{}
		err := app.runList(cmd, false)
		assert.Error(t, err)
	})
}
