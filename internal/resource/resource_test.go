package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
