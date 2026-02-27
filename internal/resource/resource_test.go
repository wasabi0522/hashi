package resource

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusIsHealthy(t *testing.T) {
	assert.True(t, StatusOK.IsHealthy())
	assert.False(t, StatusWorktreeMissing.IsHealthy())
	assert.False(t, StatusOrphanedWindow.IsHealthy())
	assert.False(t, StatusOrphanedWorktree.IsHealthy())
}

func TestStatusLabel(t *testing.T) {
	assert.Equal(t, "", StatusOK.Label())
	assert.Equal(t, "worktree missing", StatusWorktreeMissing.Label())
	assert.Equal(t, "orphaned window", StatusOrphanedWindow.Label())
	assert.Equal(t, "orphaned worktree", StatusOrphanedWorktree.Label())
}

func TestStatusSuggestedCommand(t *testing.T) {
	assert.Equal(t, "", StatusOK.SuggestedCommand())
	assert.Equal(t, "new", StatusWorktreeMissing.SuggestedCommand())
	assert.Equal(t, "remove", StatusOrphanedWindow.SuggestedCommand())
	assert.Equal(t, "remove", StatusOrphanedWorktree.SuggestedCommand())
}

func TestStatusString(t *testing.T) {
	assert.Equal(t, "ok", StatusOK.String())
	assert.Equal(t, "worktree_missing", StatusWorktreeMissing.String())
	assert.Equal(t, "orphaned_window", StatusOrphanedWindow.String())
	assert.Equal(t, "orphaned_worktree", StatusOrphanedWorktree.String())
}

func TestStatusMarshalJSON(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusOK, `"ok"`},
		{StatusWorktreeMissing, `"worktree_missing"`},
		{StatusOrphanedWindow, `"orphaned_window"`},
		{StatusOrphanedWorktree, `"orphaned_worktree"`},
	}
	for _, tt := range tests {
		data, err := json.Marshal(tt.status)
		require.NoError(t, err)
		assert.Equal(t, tt.want, string(data))
	}
}

func TestStatusUnmarshalJSON(t *testing.T) {
	t.Run("valid statuses", func(t *testing.T) {
		tests := []struct {
			input string
			want  Status
		}{
			{`"ok"`, StatusOK},
			{`"worktree_missing"`, StatusWorktreeMissing},
			{`"orphaned_window"`, StatusOrphanedWindow},
			{`"orphaned_worktree"`, StatusOrphanedWorktree},
		}
		for _, tt := range tests {
			var got Status
			err := json.Unmarshal([]byte(tt.input), &got)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		}
	})

	t.Run("unknown status", func(t *testing.T) {
		var s Status
		err := json.Unmarshal([]byte(`"invalid"`), &s)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown status")
	})
}

func TestStatusJSONRoundTrip(t *testing.T) {
	for _, s := range []Status{StatusOK, StatusWorktreeMissing, StatusOrphanedWindow, StatusOrphanedWorktree} {
		data, err := json.Marshal(s)
		require.NoError(t, err)
		var got Status
		require.NoError(t, json.Unmarshal(data, &got))
		assert.Equal(t, s, got)
	}
}
