package tmux

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMock() *ClientMock {
	return &ClientMock{}
}

func TestNewPrefixedClient_emptyPrefix(t *testing.T) {
	inner := newMock()
	got := NewPrefixedClient(inner, "")
	assert.Same(t, inner, got, "empty prefix should return inner client as-is")
}

func TestPrefixedClient_HasSession(t *testing.T) {
	inner := newMock()
	inner.HasSessionFunc = func(name string) (bool, error) {
		assert.Equal(t, "hs/sess", name)
		return true, nil
	}
	c := NewPrefixedClient(inner, "hs/")
	ok, err := c.HasSession("sess")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestPrefixedClient_NewSession(t *testing.T) {
	inner := newMock()
	inner.NewSessionFunc = func(name, windowName, dir, initCmd string) error {
		assert.Equal(t, "hs/sess", name)
		assert.Equal(t, "hs/win", windowName)
		assert.Equal(t, "/dir", dir)
		assert.Equal(t, "cmd", initCmd)
		return nil
	}
	c := NewPrefixedClient(inner, "hs/")
	require.NoError(t, c.NewSession("sess", "win", "/dir", "cmd"))
}

func TestPrefixedClient_KillSession(t *testing.T) {
	inner := newMock()
	inner.KillSessionFunc = func(name string) error {
		assert.Equal(t, "hs/sess", name)
		return nil
	}
	c := NewPrefixedClient(inner, "hs/")
	require.NoError(t, c.KillSession("sess"))
}

func TestPrefixedClient_ListWindows(t *testing.T) {
	inner := newMock()
	inner.ListWindowsFunc = func(session string) ([]Window, error) {
		assert.Equal(t, "hs/sess", session)
		return []Window{
			{Name: "hs/main", Active: true},
			{Name: "hs/feat", Active: false},
		}, nil
	}
	c := NewPrefixedClient(inner, "hs/")
	ws, err := c.ListWindows("sess")
	require.NoError(t, err)
	require.Len(t, ws, 2)
	assert.Equal(t, "main", ws[0].Name, "prefix should be stripped")
	assert.True(t, ws[0].Active)
	assert.Equal(t, "feat", ws[1].Name, "prefix should be stripped")
	assert.False(t, ws[1].Active)
}

func TestPrefixedClient_NewWindow(t *testing.T) {
	inner := newMock()
	inner.NewWindowFunc = func(session, name, dir, initCmd string) error {
		assert.Equal(t, "hs/sess", session)
		assert.Equal(t, "hs/win", name)
		assert.Equal(t, "/dir", dir)
		return nil
	}
	c := NewPrefixedClient(inner, "hs/")
	require.NoError(t, c.NewWindow("sess", "win", "/dir", ""))
}

func TestPrefixedClient_KillWindow(t *testing.T) {
	inner := newMock()
	inner.KillWindowFunc = func(session, window string) error {
		assert.Equal(t, "hs/sess", session)
		assert.Equal(t, "hs/win", window)
		return nil
	}
	c := NewPrefixedClient(inner, "hs/")
	require.NoError(t, c.KillWindow("sess", "win"))
}

func TestPrefixedClient_RenameWindow(t *testing.T) {
	inner := newMock()
	inner.RenameWindowFunc = func(session, old, new string) error {
		assert.Equal(t, "hs/sess", session)
		assert.Equal(t, "hs/old", old)
		assert.Equal(t, "hs/new", new)
		return nil
	}
	c := NewPrefixedClient(inner, "hs/")
	require.NoError(t, c.RenameWindow("sess", "old", "new"))
}

func TestPrefixedClient_SendKeys(t *testing.T) {
	inner := newMock()
	inner.SendKeysFunc = func(session, window string, keys ...string) error {
		assert.Equal(t, "hs/sess", session)
		assert.Equal(t, "hs/win", window)
		assert.Equal(t, []string{"C-u", "cd /dir", "Enter"}, keys)
		return nil
	}
	c := NewPrefixedClient(inner, "hs/")
	require.NoError(t, c.SendKeys("sess", "win", "C-u", "cd /dir", "Enter"))
}

func TestPrefixedClient_PaneCurrentCommand(t *testing.T) {
	inner := newMock()
	inner.PaneCurrentCommandFunc = func(session, window string) (string, error) {
		assert.Equal(t, "hs/sess", session)
		assert.Equal(t, "hs/win", window)
		return "zsh", nil
	}
	c := NewPrefixedClient(inner, "hs/")
	cmd, err := c.PaneCurrentCommand("sess", "win")
	require.NoError(t, err)
	assert.Equal(t, "zsh", cmd)
}

func TestPrefixedClient_AttachSession(t *testing.T) {
	inner := newMock()
	inner.AttachSessionFunc = func(session, window string) error {
		assert.Equal(t, "hs/sess", session)
		assert.Equal(t, "hs/win", window)
		return nil
	}
	c := NewPrefixedClient(inner, "hs/")
	require.NoError(t, c.AttachSession("sess", "win"))
}

func TestPrefixedClient_SwitchClient(t *testing.T) {
	inner := newMock()
	inner.SwitchClientFunc = func(session, window string) error {
		assert.Equal(t, "hs/sess", session)
		assert.Equal(t, "hs/win", window)
		return nil
	}
	c := NewPrefixedClient(inner, "hs/")
	require.NoError(t, c.SwitchClient("sess", "win"))
}

func TestPrefixedClient_IsInsideTmux(t *testing.T) {
	inner := newMock()
	inner.IsInsideTmuxFunc = func() bool { return true }
	c := NewPrefixedClient(inner, "hs/")
	assert.True(t, c.IsInsideTmux())
}
