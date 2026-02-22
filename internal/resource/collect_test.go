package resource

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

func TestCollectState(t *testing.T) {
	t.Run("all resources present", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return []git.Worktree{
						{Path: "/repo", Branch: "main", IsMain: true},
						{Path: "/repo/.worktrees/feature", Branch: "feature"},
					}, nil
				},
				ListBranchesFunc: func() ([]string, error) {
					return []string{"main", "feature"}, nil
				},
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return true, nil },
				ListWindowsFunc: func(session string) ([]tmux.Window, error) {
					return []tmux.Window{
						{Name: "main", Active: true},
						{Name: "feature", Active: false},
					}, nil
				},
			},
			WithCommonParams(CommonParams{SessionName: "org/repo"}),
		)

		states, err := svc.CollectState(context.Background())
		require.NoError(t, err)
		require.Len(t, states, 2)

		assert.Equal(t, "main", states[0].Branch)
		assert.Equal(t, StatusOK, states[0].Status)
		assert.True(t, states[0].Active)

		assert.Equal(t, "feature", states[1].Branch)
		assert.Equal(t, StatusOK, states[1].Status)
		assert.False(t, states[1].Active)
	})

	t.Run("no tmux session", func(t *testing.T) {
		svc := newTestSvc(
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
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
			},
			WithCommonParams(CommonParams{SessionName: "org/repo"}),
		)

		states, err := svc.CollectState(context.Background())
		require.NoError(t, err)
		require.Len(t, states, 1)
		assert.Equal(t, "main", states[0].Branch)
		assert.False(t, states[0].Window)
		assert.False(t, states[0].Active)
	})

	t.Run("worktree missing (window + branch exist)", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return []git.Worktree{
						{Path: "/repo", Branch: "main", IsMain: true},
					}, nil
				},
				ListBranchesFunc: func() ([]string, error) {
					return []string{"main", "fix-bug"}, nil
				},
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return true, nil },
				ListWindowsFunc: func(session string) ([]tmux.Window, error) {
					return []tmux.Window{
						{Name: "main", Active: true},
						{Name: "fix-bug", Active: false},
					}, nil
				},
			},
			WithCommonParams(CommonParams{SessionName: "org/repo"}),
		)

		states, err := svc.CollectState(context.Background())
		require.NoError(t, err)
		require.Len(t, states, 2)
		assert.Equal(t, "fix-bug", states[1].Branch)
		assert.Equal(t, StatusWorktreeMissing, states[1].Status)
		assert.Empty(t, states[1].Worktree)
	})

	t.Run("orphaned window (window exists, no branch)", func(t *testing.T) {
		svc := newTestSvc(
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
				HasSessionFunc: func(name string) (bool, error) { return true, nil },
				ListWindowsFunc: func(session string) ([]tmux.Window, error) {
					return []tmux.Window{
						{Name: "main", Active: true},
						{Name: "orphan-x", Active: false},
					}, nil
				},
			},
			WithCommonParams(CommonParams{SessionName: "org/repo"}),
		)

		states, err := svc.CollectState(context.Background())
		require.NoError(t, err)
		require.Len(t, states, 2)
		assert.Equal(t, "orphan-x", states[1].Branch)
		assert.Equal(t, StatusOrphanedWindow, states[1].Status)
	})

	t.Run("orphaned worktree (worktree exists but branch deleted)", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return []git.Worktree{
						{Path: "/repo", Branch: "main", IsMain: true},
						{Path: "/repo/.worktrees/stale", Branch: "stale"},
					}, nil
				},
				ListBranchesFunc: func() ([]string, error) {
					return []string{"main"}, nil
				},
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
			},
			WithCommonParams(CommonParams{SessionName: "org/repo"}),
		)

		states, err := svc.CollectState(context.Background())
		require.NoError(t, err)
		require.Len(t, states, 2)
		assert.Equal(t, "stale", states[1].Branch)
		assert.Equal(t, StatusOrphanedWorktree, states[1].Status)
	})

	t.Run("ListBranches error", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return []git.Worktree{
						{Path: "/repo/.worktrees/feat", Branch: "feat"},
					}, nil
				},
				ListBranchesFunc: func() ([]string, error) {
					return nil, fmt.Errorf("git error")
				},
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
			},
			WithCommonParams(CommonParams{SessionName: "org/repo"}),
		)

		_, err := svc.CollectState(context.Background())
		assert.Error(t, err)
	})

	t.Run("ListWorktrees error", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return nil, fmt.Errorf("git error")
				},
			},
			stubTmux(),
			WithCommonParams(CommonParams{SessionName: "org/repo"}),
		)

		_, err := svc.CollectState(context.Background())
		assert.Error(t, err)
	})

	t.Run("detached HEAD worktree skipped", func(t *testing.T) {
		svc := newTestSvc(
			&git.ClientMock{
				ListWorktreesFunc: func() ([]git.Worktree, error) {
					return []git.Worktree{
						{Path: "/repo", Branch: "main", IsMain: true},
						{Path: "/repo/.worktrees/detached", Branch: "", Detached: true},
					}, nil
				},
				ListBranchesFunc: func() ([]string, error) {
					return []string{"main"}, nil
				},
			},
			&tmux.ClientMock{
				HasSessionFunc: func(name string) (bool, error) { return false, nil },
			},
			WithCommonParams(CommonParams{SessionName: "org/repo"}),
		)

		states, err := svc.CollectState(context.Background())
		require.NoError(t, err)
		require.Len(t, states, 1)
		assert.Equal(t, "main", states[0].Branch)
	})
}
