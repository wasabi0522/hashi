package tmux

import (
	"os"
	"strings"

	"github.com/wasabi0522/hashi/internal/exec"
)

func target(session, window string) string {
	return session + ":" + window
}

var _ Client = (*client)(nil)

type client struct {
	exec exec.Executor
}

// NewClient creates a tmux Client backed by the given Executor.
func NewClient(exec exec.Executor) Client {
	return &client{exec: exec}
}

func (c *client) HasSession(name string) (bool, error) {
	err := c.exec.Run("tmux", "has-session", "-t", name)
	if err == nil {
		return true, nil
	}
	if exec.IsExitCode(err, 1) {
		return false, nil
	}
	return false, err
}

func (c *client) NewSession(name, windowName, dir, initCmd string) error {
	args := []string{"new-session", "-d", "-s", name, "-n", windowName, "-c", dir}
	if initCmd != "" {
		args = append(args, initCmd)
	}
	return c.exec.Run("tmux", args...)
}

func (c *client) KillSession(name string) error {
	return c.exec.Run("tmux", "kill-session", "-t", name)
}

func (c *client) ListWindows(session string) ([]Window, error) {
	out, err := c.exec.Output("tmux", "list-windows", "-t", session, "-F", "#{window_name}\t#{window_active}")
	if err != nil {
		return nil, err
	}
	return parseWindowList(out), nil
}

func (c *client) NewWindow(session, name, dir, initCmd string) error {
	args := []string{"new-window", "-a", "-t", session, "-n", name, "-c", dir}
	if initCmd != "" {
		args = append(args, initCmd)
	}
	return c.exec.Run("tmux", args...)
}

func (c *client) KillWindow(session, window string) error {
	return c.exec.Run("tmux", "kill-window", "-t", target(session, window))
}

func (c *client) RenameWindow(session, old, new string) error {
	return c.exec.Run("tmux", "rename-window", "-t", target(session, old), new)
}

func (c *client) SendKeys(session, window string, keys ...string) error {
	args := []string{"send-keys", "-t", target(session, window)}
	args = append(args, keys...)
	return c.exec.Run("tmux", args...)
}

func (c *client) PaneCurrentCommand(session, window string) (string, error) {
	return c.exec.Output("tmux", "display-message", "-t", target(session, window), "-p", "#{pane_current_command}")
}

func (c *client) AttachSession(session, window string) error {
	return c.exec.RunInteractive("tmux", "attach-session", "-t", target(session, window))
}

func (c *client) SwitchClient(session, window string) error {
	return c.exec.Run("tmux", "switch-client", "-t", target(session, window))
}

func (c *client) IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// tmuxActiveFlag is the value tmux uses in #{window_active} to indicate the active window.
const tmuxActiveFlag = "1"

// parseWindowList parses the output of `tmux list-windows -F '#{window_name}\t#{window_active}'`.
func parseWindowList(output string) []Window {
	if output == "" {
		return nil
	}

	var windows []Window
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		windows = append(windows, Window{
			Name:   parts[0],
			Active: parts[1] == tmuxActiveFlag,
		})
	}

	return windows
}
