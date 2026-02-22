package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hashiexec "github.com/wasabi0522/hashi/internal/exec"
)

func TestCompletionCommand(t *testing.T) {
	app := NewApp()
	rootCmd := app.BuildRootCmd()
	// Find the completion subcommand
	var compCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "completion <bash|zsh|fish>" {
			compCmd = c
			break
		}
	}
	require.NotNil(t, compCmd, "completion command not found")

	t.Run("bash", func(t *testing.T) {
		var buf bytes.Buffer
		compCmd.SetOut(&buf)
		err := compCmd.RunE(compCmd, []string{"bash"})
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "bash")
	})

	t.Run("zsh", func(t *testing.T) {
		var buf bytes.Buffer
		compCmd.SetOut(&buf)
		err := compCmd.RunE(compCmd, []string{"zsh"})
		require.NoError(t, err)
		assert.NotEmpty(t, buf.String())
	})

	t.Run("fish", func(t *testing.T) {
		var buf bytes.Buffer
		compCmd.SetOut(&buf)
		err := compCmd.RunE(compCmd, []string{"fish"})
		require.NoError(t, err)
		assert.NotEmpty(t, buf.String())
	})

	t.Run("unsupported shell", func(t *testing.T) {
		err := compCmd.RunE(compCmd, []string{"powershell"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported shell")
	})
}

func TestCompleteBranchesWithExec(t *testing.T) {
	t.Run("git not found", func(t *testing.T) {
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				return fmt.Errorf("not found")
			},
		}
		branches, directive := completeBranchesWithExec(e)
		assert.Nil(t, branches)
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})

	t.Run("ListBranches error", func(t *testing.T) {
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				return nil
			},
			OutputFunc: func(name string, args ...string) (string, error) {
				return "", fmt.Errorf("git error")
			},
		}
		branches, directive := completeBranchesWithExec(e)
		assert.Nil(t, branches)
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})

	t.Run("success", func(t *testing.T) {
		e := &hashiexec.ExecutorMock{
			LookPathFunc: func(name string) error {
				return nil
			},
			OutputFunc: func(name string, args ...string) (string, error) {
				return "main\nfeature\nbugfix", nil
			},
		}
		branches, directive := completeBranchesWithExec(e)
		assert.Equal(t, []string{"main", "feature", "bugfix"}, branches)
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})
}
