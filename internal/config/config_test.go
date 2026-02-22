package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("defaults when no file", func(t *testing.T) {
		cfg, err := Load("/nonexistent/.hashi.yaml")
		require.NoError(t, err)
		assert.Equal(t, ".worktrees", cfg.WorktreeDir)
		assert.Empty(t, cfg.Hooks.PostNew)
	})

	t.Run("from yaml file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".hashi.yaml")
		content := "worktree_dir: custom_dir\nhooks:\n  copy_files:\n    - .env\n    - .claude\n  post_new:\n    - npm install\n"
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))

		cfg, err := Load(path)
		require.NoError(t, err)
		assert.Equal(t, "custom_dir", cfg.WorktreeDir)
		assert.Equal(t, []string{".env", ".claude"}, cfg.Hooks.CopyFiles)
		assert.Equal(t, []string{"npm install"}, cfg.Hooks.PostNew)
	})

	t.Run("env var overrides file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".hashi.yaml")
		require.NoError(t, os.WriteFile(path, []byte("worktree_dir: from_file\n"), 0644))

		t.Setenv("HASHI_WORKTREE_DIR", "from_env")

		cfg, err := Load(path)
		require.NoError(t, err)
		assert.Equal(t, "from_env", cfg.WorktreeDir)
	})

	t.Run("env var overrides default", func(t *testing.T) {
		t.Setenv("HASHI_WORKTREE_DIR", "env_dir")

		cfg, err := Load("/nonexistent/.hashi.yaml")
		require.NoError(t, err)
		assert.Equal(t, "env_dir", cfg.WorktreeDir)
	})

	t.Run("absolute worktree_dir rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".hashi.yaml")
		require.NoError(t, os.WriteFile(path, []byte("worktree_dir: /absolute/path\n"), 0644))

		_, err := Load(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "relative path")
	})

	t.Run("worktree_dir dot rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".hashi.yaml")
		require.NoError(t, os.WriteFile(path, []byte("worktree_dir: \".\"\n"), 0644))

		_, err := Load(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be '.'")
	})

	t.Run("worktree_dir with .. rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".hashi.yaml")
		require.NoError(t, os.WriteFile(path, []byte("worktree_dir: ../escape\n"), 0644))

		_, err := Load(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "..")
	})

	t.Run("bare keys without values", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".hashi.yaml")
		content := "hooks:\n  copy_files:\n  post_new:\n"
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))

		cfg, err := Load(path)
		require.NoError(t, err)
		assert.Empty(t, cfg.Hooks.CopyFiles)
		assert.Empty(t, cfg.Hooks.PostNew)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".hashi.yaml")
		require.NoError(t, os.WriteFile(path, []byte("invalid: [yaml: broken"), 0644))

		_, err := Load(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading config")
	})
}

func TestLoadFromReader(t *testing.T) {
	t.Run("reads valid yaml", func(t *testing.T) {
		r := strings.NewReader("worktree_dir: custom\nhooks:\n  copy_files:\n    - .env\n  post_new:\n    - npm install\n")
		cfg, err := LoadFromReader(r)
		require.NoError(t, err)
		assert.Equal(t, "custom", cfg.WorktreeDir)
		assert.Equal(t, []string{".env"}, cfg.Hooks.CopyFiles)
		assert.Equal(t, []string{"npm install"}, cfg.Hooks.PostNew)
	})

	t.Run("uses defaults", func(t *testing.T) {
		r := strings.NewReader("")
		cfg, err := LoadFromReader(r)
		require.NoError(t, err)
		assert.Equal(t, ".worktrees", cfg.WorktreeDir)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		r := strings.NewReader("invalid: [yaml: broken")
		_, err := LoadFromReader(r)
		require.Error(t, err)
	})

	t.Run("validation error", func(t *testing.T) {
		r := strings.NewReader("worktree_dir: /absolute")
		_, err := LoadFromReader(r)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "relative path")
	})
}
