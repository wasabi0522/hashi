package tmux

import (
	"fmt"
	osexec "os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/exec"
)

func mockExec() *exec.ExecutorMock {
	return &exec.ExecutorMock{}
}

func TestNewClient(t *testing.T) {
	c := NewClient(mockExec())
	assert.NotNil(t, c)
}

func TestClientHasSession(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error { return nil }
		c := NewClient(e)
		ok, err := c.HasSession("sess")
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not exists", func(t *testing.T) {
		e := mockExec()
		// tmux has-session exits with code 1 when session is not found.
		exitErr := osexec.Command("false").Run()
		e.RunFunc = func(name string, args ...string) error { return exitErr }
		c := NewClient(e)
		ok, err := c.HasSession("sess")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("unexpected error", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error { return fmt.Errorf("unexpected") }
		c := NewClient(e)
		ok, err := c.HasSession("sess")
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestClientNewSession(t *testing.T) {
	t.Run("without initCmd", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error {
			assert.Equal(t, "tmux", name)
			assert.Equal(t, []string{"new-session", "-d", "-s", "sess", "-n", "win", "-c", "/dir"}, args)
			return nil
		}
		c := NewClient(e)
		require.NoError(t, c.NewSession("sess", "win", "/dir", ""))
	})

	t.Run("with initCmd", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error {
			assert.Equal(t, "tmux", name)
			assert.Equal(t, []string{"new-session", "-d", "-s", "sess", "-n", "win", "-c", "/dir", "echo hello; exec zsh"}, args)
			return nil
		}
		c := NewClient(e)
		require.NoError(t, c.NewSession("sess", "win", "/dir", "echo hello; exec zsh"))
	})
}

func TestClientKillSession(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"kill-session", "-t", "sess"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.KillSession("sess"))
}

func TestClientListWindows(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "main\t1\nfeat\t0", nil
		}
		c := NewClient(e)
		ws, err := c.ListWindows("sess")
		require.NoError(t, err)
		require.Len(t, ws, 2)
		assert.Equal(t, "main", ws[0].Name)
		assert.True(t, ws[0].Active)
	})

	t.Run("error", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "", fmt.Errorf("fail")
		}
		c := NewClient(e)
		_, err := c.ListWindows("sess")
		assert.Error(t, err)
	})
}

func TestClientNewWindow(t *testing.T) {
	t.Run("without initCmd", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error {
			assert.Equal(t, []string{"new-window", "-a", "-t", "sess", "-n", "win", "-c", "/dir"}, args)
			return nil
		}
		c := NewClient(e)
		require.NoError(t, c.NewWindow("sess", "win", "/dir", ""))
	})

	t.Run("with initCmd", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error {
			assert.Equal(t, []string{"new-window", "-a", "-t", "sess", "-n", "win", "-c", "/dir", "npm install; exec bash"}, args)
			return nil
		}
		c := NewClient(e)
		require.NoError(t, c.NewWindow("sess", "win", "/dir", "npm install; exec bash"))
	})
}

func TestClientKillWindow(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"kill-window", "-t", "sess:win"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.KillWindow("sess", "win"))
}

func TestClientRenameWindow(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"rename-window", "-t", "sess:old", "new"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.RenameWindow("sess", "old", "new"))
}

func TestClientSendKeys(t *testing.T) {
	t.Run("single key", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error {
			assert.Equal(t, []string{"send-keys", "-t", "sess:win", "C-u"}, args)
			return nil
		}
		c := NewClient(e)
		require.NoError(t, c.SendKeys("sess", "win", "C-u"))
	})

	t.Run("multiple keys", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error {
			assert.Equal(t, []string{"send-keys", "-t", "sess:win", "cd /dir", "Enter"}, args)
			return nil
		}
		c := NewClient(e)
		require.NoError(t, c.SendKeys("sess", "win", "cd /dir", "Enter"))
	})
}

func TestClientPaneCurrentCommand(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			assert.Equal(t, []string{"display-message", "-t", "sess:win", "-p", "#{pane_current_command}"}, args)
			return "zsh", nil
		}
		c := NewClient(e)
		cmd, err := c.PaneCurrentCommand("sess", "win")
		require.NoError(t, err)
		assert.Equal(t, "zsh", cmd)
	})

	t.Run("error", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "", fmt.Errorf("tmux error")
		}
		c := NewClient(e)
		_, err := c.PaneCurrentCommand("sess", "win")
		assert.Error(t, err)
	})
}

func TestClientAttachSession(t *testing.T) {
	e := mockExec()
	e.RunInteractiveFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"attach-session", "-t", "sess:win"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.AttachSession("sess", "win"))
}

func TestClientSwitchClient(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"switch-client", "-t", "sess:win"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.SwitchClient("sess", "win"))
}

func TestClientIsInsideTmux(t *testing.T) {
	t.Run("inside", func(t *testing.T) {
		t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
		c := NewClient(mockExec())
		assert.True(t, c.IsInsideTmux())
	})

	t.Run("outside", func(t *testing.T) {
		t.Setenv("TMUX", "")
		c := NewClient(mockExec())
		assert.False(t, c.IsInsideTmux())
	})
}

func TestParseWindowList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Window
	}{
		{name: "empty", input: "", want: nil},
		{
			name: "single active window", input: "main\t1",
			want: []Window{{Name: "main", Active: true}},
		},
		{
			name: "multiple windows", input: "main\t1\nfeature-login\t0\nfix-bug\t0",
			want: []Window{
				{Name: "main", Active: true},
				{Name: "feature-login", Active: false},
				{Name: "fix-bug", Active: false},
			},
		},
		{
			name: "slash in window name", input: "main\t0\nfeat/auth\t1",
			want: []Window{
				{Name: "main", Active: false},
				{Name: "feat/auth", Active: true},
			},
		},
		{
			name: "malformed line ignored", input: "main\t1\nbadline\nfeat\t0",
			want: []Window{
				{Name: "main", Active: true},
				{Name: "feat", Active: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWindowList(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
