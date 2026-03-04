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

func (p *prefixedClient) addPrefix(name string) string {
	return p.prefix + name
}

func (p *prefixedClient) stripPrefix(name string) string {
	return strings.TrimPrefix(name, p.prefix)
}

// Session operations

func (p *prefixedClient) HasSession(name string) (bool, error) {
	return p.inner.HasSession(p.addPrefix(name))
}

func (p *prefixedClient) NewSession(name, windowName, dir, initCmd string) error {
	return p.inner.NewSession(p.addPrefix(name), p.addPrefix(windowName), dir, initCmd)
}

func (p *prefixedClient) KillSession(name string) error {
	return p.inner.KillSession(p.addPrefix(name))
}

// Window operations

func (p *prefixedClient) ListWindows(session string) ([]Window, error) {
	windows, err := p.inner.ListWindows(p.addPrefix(session))
	if err != nil {
		return nil, err
	}
	managed := windows[:0] // reuse backing array to filter in-place
	for _, w := range windows {
		if !strings.HasPrefix(w.Name, p.prefix) {
			continue
		}
		w.Name = p.stripPrefix(w.Name)
		managed = append(managed, w)
	}
	return managed, nil
}

func (p *prefixedClient) NewWindow(session, name, dir, initCmd string) error {
	return p.inner.NewWindow(p.addPrefix(session), p.addPrefix(name), dir, initCmd)
}

func (p *prefixedClient) KillWindow(session, window string) error {
	return p.inner.KillWindow(p.addPrefix(session), p.addPrefix(window))
}

func (p *prefixedClient) RenameWindow(session, oldName, newName string) error {
	return p.inner.RenameWindow(p.addPrefix(session), p.addPrefix(oldName), p.addPrefix(newName))
}

func (p *prefixedClient) SendKeys(session, window string, keys ...string) error {
	return p.inner.SendKeys(p.addPrefix(session), p.addPrefix(window), keys...)
}

func (p *prefixedClient) PaneCurrentCommand(session, window string) (string, error) {
	return p.inner.PaneCurrentCommand(p.addPrefix(session), p.addPrefix(window))
}

// Connection

func (p *prefixedClient) AttachSession(session, window string) error {
	return p.inner.AttachSession(p.addPrefix(session), p.addPrefix(window))
}

func (p *prefixedClient) SwitchClient(session, window string) error {
	return p.inner.SwitchClient(p.addPrefix(session), p.addPrefix(window))
}

// Environment

func (p *prefixedClient) IsInsideTmux() bool {
	return p.inner.IsInsideTmux()
}
