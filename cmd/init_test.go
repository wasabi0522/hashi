package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hashicontext "github.com/wasabi0522/hashi/internal/context"
	"github.com/wasabi0522/hashi/internal/git"
)

func TestRunInit(t *testing.T) {
	t.Run("creates config file", func(t *testing.T) {
		repoRoot := t.TempDir()
		app := &App{
			resolveGitDeps: func() (*gitDeps, error) {
				return &gitDeps{
					git: &git.ClientMock{},
					ctx: &hashicontext.Context{RepoRoot: repoRoot},
				}, nil
			},
		}

		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)
		err := app.runInit(cmd, nil)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(repoRoot, ".hashi.yaml"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "worktree_dir")
		assert.Contains(t, buf.String(), "Created")
	})

	t.Run("errors when config already exists", func(t *testing.T) {
		repoRoot := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(repoRoot, ".hashi.yaml"), []byte("existing"), 0644))

		app := &App{
			resolveGitDeps: func() (*gitDeps, error) {
				return &gitDeps{
					git: &git.ClientMock{},
					ctx: &hashicontext.Context{RepoRoot: repoRoot},
				}, nil
			},
		}

		cmd := &cobra.Command{}
		err := app.runInit(cmd, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("deps error", func(t *testing.T) {
		app := &App{
			resolveGitDeps: func() (*gitDeps, error) {
				return nil, fmt.Errorf("no git")
			},
		}

		cmd := &cobra.Command{}
		err := app.runInit(cmd, nil)
		assert.Error(t, err)
	})
}
