package resource

import (
	"fmt"
	"strings"

	"github.com/wasabi0522/hashi/internal/tmux"
)

// ensureTmux ensures the tmux session and window exist.
// Creates session if missing, creates window if missing, updates directory if window exists.
// initCmd, if non-empty, is passed to tmux new-session/new-window as the initial shell command.
func (s *Service) ensureTmux(sessionName, windowName, dir, initCmd string) error {
	ok, err := s.tmux.HasSession(sessionName)
	if err != nil {
		return fmt.Errorf("checking session: %w", err)
	}
	if !ok {
		return s.tmux.NewSession(sessionName, windowName, dir, initCmd)
	}

	windows, err := s.tmux.ListWindows(sessionName)
	if err != nil {
		return fmt.Errorf("listing windows: %w", err)
	}

	if w := findWindow(windows, windowName); w != nil {
		s.sendCd(sessionName, windowName, dir)
		return nil
	}

	return s.tmux.NewWindow(sessionName, windowName, dir, initCmd)
}

// listWindowsSafe returns the tmux windows for the given session.
// Returns nil if the session does not exist or ListWindows fails.
func (s *Service) listWindowsSafe(sessionName string) []tmux.Window {
	ok, err := s.tmux.HasSession(sessionName)
	s.bestEffort("HasSession", err)
	if !ok {
		return nil
	}
	windows, err := s.tmux.ListWindows(sessionName)
	s.bestEffort("ListWindows", err)
	return windows
}

// sendCd sends a cd command to the tmux pane if it is running a shell.
// Skips if the pane is running a non-shell process (e.g. vim).
func (s *Service) sendCd(session, window, dir string) {
	cmd, err := s.tmux.PaneCurrentCommand(session, window)
	if err != nil {
		s.bestEffort("PaneCurrentCommand", err)
		return
	}
	if !s.isShellCommand(cmd) {
		return
	}
	s.bestEffort("SendKeys", s.tmux.SendKeys(session, window, "C-u", "cd "+shellQuote(dir), "Enter"))
}

// DefaultShellCommands is the default set of commands recognized as interactive shells.
var DefaultShellCommands = map[string]struct{}{
	"bash": {}, "zsh": {}, "fish": {}, "sh": {},
	"dash": {}, "ksh": {}, "tcsh": {}, "csh": {},
}

func (s *Service) isShellCommand(cmd string) bool {
	_, ok := s.shellCommands[cmd]
	return ok
}

// connect attaches or switches to the tmux session/window.
func (s *Service) connect(sessionName, windowName string) error {
	if s.tmux.IsInsideTmux() {
		return s.tmux.SwitchClient(sessionName, windowName)
	}
	return s.tmux.AttachSession(sessionName, windowName)
}

// finalizeOperation connects to tmux and returns the result.
func (s *Service) finalizeOperation(op OperationType, branch, wtPath string, wtCreated bool) (*OperationResult, error) {
	if err := s.connect(s.params.SessionName, branch); err != nil {
		return nil, err
	}
	return &OperationResult{Operation: op, Branch: branch, WorktreePath: wtPath, Created: wtCreated}, nil
}

// buildInitCmd builds the tmux initial command string for post_new hooks.
// Each hook runs in its own sh -c subshell chained with && for fail-fast behavior.
// The user's login shell is taken from CommonParams.Shell; falls back to "sh" if empty.
// Returns "" if no hooks, worktree was not created, or hooks are disabled.
func (s *Service) buildInitCmd(wtCreated, noHooks bool) string {
	if !wtCreated || noHooks || len(s.params.PostNewHooks) == 0 {
		return ""
	}
	shell := s.params.Shell
	if shell == "" {
		shell = "sh"
	}
	parts := make([]string, 0, len(s.params.PostNewHooks))
	for _, h := range s.params.PostNewHooks {
		parts = append(parts, fmt.Sprintf("sh -c %s", shellQuote(h)))
	}
	return strings.Join(parts, " && ") + "; exec " + shellQuote(shell)
}
