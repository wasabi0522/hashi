package exec

import (
	"fmt"
	"os"
	"path/filepath"
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

func TestRunShell(t *testing.T) {
	e := NewDefaultExecutor()

	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		err := e.RunShell("touch testfile.txt", dir)
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(dir, "testfile.txt"))
		require.NoError(t, err)
	})

	t.Run("failure", func(t *testing.T) {
		err := e.RunShell("exit 1", t.TempDir())
		assert.Error(t, err)
	})

	t.Run("runs in specified directory", func(t *testing.T) {
		dir := t.TempDir()
		err := e.RunShell("pwd > result.txt", dir)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(dir, "result.txt"))
		require.NoError(t, err)
		assert.Contains(t, string(data), filepath.Base(dir))
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
