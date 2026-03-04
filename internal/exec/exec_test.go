package exec

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultExecutor(t *testing.T) {
	e := NewDefaultExecutor()
	assert.NotNil(t, e)
}

func TestLookPath(t *testing.T) {
	e := NewDefaultExecutor()

	t.Run("existing command", func(t *testing.T) {
		err := e.LookPath("git")
		require.NoError(t, err)
	})

	t.Run("missing command", func(t *testing.T) {
		err := e.LookPath("nonexistent-command-xyz-12345")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "command not found")
	})
}

func TestOutput(t *testing.T) {
	e := NewDefaultExecutor()

	t.Run("success", func(t *testing.T) {
		out, err := e.Output("echo", "hello")
		require.NoError(t, err)
		assert.Equal(t, "hello", out)
	})

	t.Run("error with stderr", func(t *testing.T) {
		_, err := e.Output("sh", "-c", "echo fail >&2; exit 1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fail")
	})

	t.Run("error without stderr", func(t *testing.T) {
		_, err := e.Output("sh", "-c", "exit 1")
		assert.Error(t, err)
	})
}

func TestRun(t *testing.T) {
	e := NewDefaultExecutor()

	t.Run("success", func(t *testing.T) {
		err := e.Run("true")
		require.NoError(t, err)
	})

	t.Run("error with stderr", func(t *testing.T) {
		err := e.Run("sh", "-c", "echo errmsg >&2; exit 1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "errmsg")
	})

	t.Run("error without stderr", func(t *testing.T) {
		err := e.Run("false")
		assert.Error(t, err)
	})
}

func TestIsExitCode(t *testing.T) {
	t.Run("matching exit code", func(t *testing.T) {
		e := NewDefaultExecutor()
		err := e.Run("sh", "-c", "exit 2")
		require.Error(t, err)
		assert.True(t, IsExitCode(err, 2))
	})

	t.Run("non-matching exit code", func(t *testing.T) {
		e := NewDefaultExecutor()
		err := e.Run("sh", "-c", "exit 3")
		require.Error(t, err)
		assert.False(t, IsExitCode(err, 1))
	})

	t.Run("non-exit error", func(t *testing.T) {
		assert.False(t, IsExitCode(fmt.Errorf("plain error"), 1))
	})

	t.Run("nil error", func(t *testing.T) {
		assert.False(t, IsExitCode(nil, 0))
	})
}

func TestRunInteractive(t *testing.T) {
	e := NewDefaultExecutor()

	t.Run("success", func(t *testing.T) {
		err := e.RunInteractive("true")
		require.NoError(t, err)
	})

	t.Run("failure", func(t *testing.T) {
		err := e.RunInteractive("false")
		assert.Error(t, err)
	})
}

func TestResolveShell(t *testing.T) {
	t.Run("returns SHELL when absolute", func(t *testing.T) {
		t.Setenv("SHELL", "/bin/zsh")
		assert.Equal(t, "/bin/zsh", ResolveShell())
	})

	t.Run("falls back to sh when SHELL is relative", func(t *testing.T) {
		t.Setenv("SHELL", "zsh")
		assert.Equal(t, "sh", ResolveShell())
	})

	t.Run("falls back to sh when SHELL is empty", func(t *testing.T) {
		t.Setenv("SHELL", "")
		assert.Equal(t, "sh", ResolveShell())
	})
}

func TestWrapExecError(t *testing.T) {
	baseErr := fmt.Errorf("exit status 1")

	t.Run("returns base error when stderr is empty", func(t *testing.T) {
		err := wrapExecError(baseErr, "")
		assert.Equal(t, baseErr, err)
	})

	t.Run("wraps error with stderr message", func(t *testing.T) {
		err := wrapExecError(baseErr, "something went wrong")
		assert.Contains(t, err.Error(), "something went wrong")
		assert.ErrorIs(t, err, baseErr)
	})

	t.Run("truncates long stderr", func(t *testing.T) {
		long := strings.Repeat("x", 600)
		err := wrapExecError(baseErr, long)
		assert.Contains(t, err.Error(), "... (truncated)")
		// The error message should not contain the full 600-char string
		assert.Less(t, len(err.Error()), 600)
	})

	t.Run("masks credentials in URLs", func(t *testing.T) {
		stderr := "fatal: unable to access 'https://user:token@github.com/org/repo.git/': could not resolve host"
		err := wrapExecError(baseErr, stderr)
		assert.NotContains(t, err.Error(), "user:token")
		assert.Contains(t, err.Error(), "://***@")
	})

	t.Run("trims whitespace", func(t *testing.T) {
		err := wrapExecError(baseErr, "  error message  \n")
		assert.Contains(t, err.Error(), "error message")
	})
}
