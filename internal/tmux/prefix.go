package tmux

import "strings"

// DefaultPrefix is the default prefix added to tmux session and window names
// to distinguish hashi-managed resources from others.
const DefaultPrefix = "hs/"

var _ Client = (*prefixedClient)(nil)

// prefixedClient wraps a Client and transparently adds/strips a prefix
// on session and window names.
type prefixedClient struct {
	inner  Client
	prefix string
}

// NewPrefixedClient returns a Client that prepends prefix to all session
// and window names on outgoing calls, and strips the prefix from window
// names returned by ListWindows.
func NewPrefixedClient(inner Client, prefix string) Client {
	if prefix == "" {
		return inner
	}
	return &prefixedClient{inner: inner, prefix: prefix}
}

func (p *prefixedClient) add(name string) string {
	return p.prefix + name
}

func (p *prefixedClient) strip(name string) string {
	return strings.TrimPrefix(name, p.prefix)
}

// Session operations

func (p *prefixedClient) HasSession(name string) (bool, error) {
	return p.inner.HasSession(p.add(name))
}

func (p *prefixedClient) NewSession(name, windowName, dir, initCmd string) error {
	return p.inner.NewSession(p.add(name), p.add(windowName), dir, initCmd)
}

func (p *prefixedClient) KillSession(name string) error {
	return p.inner.KillSession(p.add(name))
}

// Window operations

func (p *prefixedClient) ListWindows(session string) ([]Window, error) {
	windows, err := p.inner.ListWindows(p.add(session))
	if err != nil {
		return nil, err
	}
	for i := range windows {
		windows[i].Name = p.strip(windows[i].Name)
	}
	return windows, nil
}

func (p *prefixedClient) NewWindow(session, name, dir, initCmd string) error {
	return p.inner.NewWindow(p.add(session), p.add(name), dir, initCmd)
}

func (p *prefixedClient) KillWindow(session, window string) error {
	return p.inner.KillWindow(p.add(session), p.add(window))
}

func (p *prefixedClient) RenameWindow(session, old, new string) error {
	return p.inner.RenameWindow(p.add(session), p.add(old), p.add(new))
}

func (p *prefixedClient) SendKeys(session, window string, keys ...string) error {
	return p.inner.SendKeys(p.add(session), p.add(window), keys...)
}

func (p *prefixedClient) PaneCurrentCommand(session, window string) (string, error) {
	return p.inner.PaneCurrentCommand(p.add(session), p.add(window))
}

// Connection

func (p *prefixedClient) AttachSession(session, window string) error {
	return p.inner.AttachSession(p.add(session), p.add(window))
}

func (p *prefixedClient) SwitchClient(session, window string) error {
	return p.inner.SwitchClient(p.add(session), p.add(window))
}

// Environment

func (p *prefixedClient) IsInsideTmux() bool {
	return p.inner.IsInsideTmux()
}
