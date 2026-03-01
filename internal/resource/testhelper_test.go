package resource

import (
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

// defaultCP returns a CommonParams with sensible defaults for testing.
func defaultCP() CommonParams {
	return CommonParams{
		RepoRoot:      "/repo",
		WorktreeDir:   ".worktrees",
		DefaultBranch: "main",
		SessionName:   "org/repo",
	}
}

// mockBranchExists returns a BranchExistsFunc that returns true for the given branches.
func mockBranchExists(existing ...string) func(string) (bool, error) {
	set := make(map[string]bool, len(existing))
	for _, b := range existing {
		set[b] = true
	}
	return func(name string) (bool, error) {
		return set[name], nil
	}
}

// mockListBranches returns a ListBranchesFunc that returns the given branches.
func mockListBranches(existing ...string) func() ([]string, error) {
	return func() ([]string, error) {
		return existing, nil
	}
}

// stubTmux returns a tmux.ClientMock with no-op session/window funcs.
func stubTmux() *tmux.ClientMock {
	return &tmux.ClientMock{
		HasSessionFunc: func(name string) (bool, error) {
			return false, nil
		},
		IsInsideTmuxFunc:  func() bool { return false },
		AttachSessionFunc: func(session string, window string) error { return nil },
		SendKeysFunc:      func(session string, window string, keys ...string) error { return nil },
	}
}

// stubTmuxInside returns a tmux.ClientMock that acts as if inside tmux.
func stubTmuxInside() *tmux.ClientMock {
	return &tmux.ClientMock{
		HasSessionFunc: func(name string) (bool, error) {
			return false, nil
		},
		NewSessionFunc:   func(name string, windowName string, dir string, initCmd string) error { return nil },
		IsInsideTmuxFunc: func() bool { return true },
		SwitchClientFunc: func(session string, window string) error { return nil },
		SendKeysFunc:     func(session string, window string, keys ...string) error { return nil },
	}
}

// newTestSvc creates a Service with mock git and tmux clients using NewService.
func newTestSvc(g git.Client, tm tmux.Client, opts ...Option) *Service {
	return NewService(g, tm, opts...)
}
