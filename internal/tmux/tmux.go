package tmux

//go:generate moq -out tmux_mock.go . Client

// Client abstracts tmux operations for testing.
type Client interface {
	// Session operations
	HasSession(name string) (bool, error)
	NewSession(name, windowName, dir, initCmd string) error
	KillSession(name string) error

	// Window operations
	ListWindows(session string) ([]Window, error)
	NewWindow(session, name, dir, initCmd string) error
	KillWindow(session, window string) error
	RenameWindow(session, old, new string) error
	SendKeys(session, window string, keys ...string) error
	PaneCurrentCommand(session, window string) (string, error)

	// Connection
	AttachSession(session, window string) error
	SwitchClient(session, window string) error

	// Environment
	IsInsideTmux() bool
}

// Window represents a tmux window entry.
type Window struct {
	Name   string
	Active bool
}
