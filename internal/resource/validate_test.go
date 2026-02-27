package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBranchName(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		for _, name := range []string{"feature", "fix/login", "my-branch", "v1.0"} {
			require.NoError(t, ValidateBranchName(name), "should accept %q", name)
		}
	})

	t.Run("empty", func(t *testing.T) {
		assert.Error(t, ValidateBranchName(""))
	})

	t.Run("whitespace", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("foo bar"))
	})

	t.Run("control characters", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("foo\x00bar"))
	})

	t.Run("invalid characters", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("foo~bar"))
	})

	t.Run("colon", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("foo:bar"))
	})

	t.Run("double dot", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("foo..bar"))
	})

	t.Run("at-brace", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("foo@{bar"))
	})

	t.Run("starts with dash", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("-feature"))
	})

	t.Run("starts with dot", func(t *testing.T) {
		assert.Error(t, ValidateBranchName(".hidden"))
	})

	t.Run("ends with dot", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("feature."))
	})

	t.Run("ends with slash", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("feature/"))
	})

	t.Run("consecutive slashes", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("feature//fix"))
	})

	t.Run(".lock suffix", func(t *testing.T) {
		assert.Error(t, ValidateBranchName("feature.lock"))
	})
}
