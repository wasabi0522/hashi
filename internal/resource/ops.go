package resource

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/wasabi0522/hashi/internal/tmux"
)

// bestEffort logs a warning if a best-effort operation fails.
// Does nothing if err is nil.
func (s *Service) bestEffort(op string, err error) {
	if err == nil {
		return
	}
	s.logger.Warn("best-effort operation failed", "op", op, "error", err)
}

// requireNotDefaultBranch returns a DefaultBranchError if branch is the default branch.
func (s *Service) requireNotDefaultBranch(branch, action string) error {
	if branch == s.cp.DefaultBranch {
		return &DefaultBranchError{Action: action}
	}
	return nil
}

// requireBranchExists returns a BranchNotFoundError if the branch does not exist.
func (s *Service) requireBranchExists(branch string) error {
	exists, err := s.git.BranchExists(branch)
	if err != nil {
		return fmt.Errorf("checking branch %q: %w", branch, err)
	}
	if !exists {
		return &BranchNotFoundError{Branch: branch}
	}
	return nil
}

// requireBranchNotExists returns a BranchExistsError if the branch already exists.
func (s *Service) requireBranchNotExists(branch string) error {
	exists, err := s.git.BranchExists(branch)
	if err != nil {
		return fmt.Errorf("checking branch %q: %w", branch, err)
	}
	if exists {
		return &BranchExistsError{Branch: branch}
	}
	return nil
}

// ensureWorktree ensures a worktree exists for the given branch.
// Returns (path, wasCreated, error).
func (s *Service) ensureWorktree(branch string) (string, bool, error) {
	if branch == s.cp.DefaultBranch {
		return s.cp.RepoRoot, false, nil
	}
	return s.findOrCreateWorktree(branch)
}

// findOrCreateWorktree returns the existing worktree for branch, or creates one.
// Returns (path, wasCreated, error).
func (s *Service) findOrCreateWorktree(branch string) (string, bool, error) {
	worktrees, err := s.git.ListWorktrees()
	if err != nil {
		return "", false, fmt.Errorf("listing worktrees: %w", err)
	}
	if wt := findWorktree(worktrees, branch); wt != nil {
		return wt.Path, false, nil
	}

	path := s.cp.WorktreePath(branch)
	if err := s.addWorktree(path, branch); err != nil {
		return "", false, fmt.Errorf("adding worktree: %w", err)
	}
	return path, true, nil
}

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
	if err := s.connect(s.cp.SessionName, branch); err != nil {
		return nil, err
	}
	return &OperationResult{Operation: op, Branch: branch, WorktreePath: wtPath, Created: wtCreated}, nil
}

// buildInitCmd builds the tmux initial command string for post_new hooks.
// Each hook runs in its own sh -c subshell with fail-fast behavior.
// shell is the user's login shell (e.g. from $SHELL); falls back to "sh" if empty.
// Returns "" if no hooks or worktree was not created.
func (s *Service) buildInitCmd(wtCreated bool, shell string) string {
	if !wtCreated || len(s.cp.PostNewHooks) == 0 {
		return ""
	}
	if shell == "" {
		shell = "sh"
	}
	var quoted []string
	for _, h := range s.cp.PostNewHooks {
		quoted = append(quoted, shellQuote(h))
	}
	return fmt.Sprintf("for __cmd in %s; do sh -c \"$__cmd\" || exit 1; done; exec %s",
		strings.Join(quoted, " "), shellQuote(shell))
}

// copyFiles copies configured files and directories from repo root to the worktree.
// Entries that do not exist in the repo root are silently skipped.
func (s *Service) copyFiles(wtPath string) error {
	for _, rel := range s.cp.CopyFiles {
		src := filepath.Join(s.cp.RepoRoot, rel)
		dst := filepath.Join(wtPath, rel)

		info, err := os.Lstat(src)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("stat %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue // skip symlinks to prevent following links outside the repo
		}

		if info.IsDir() {
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("copying directory %s: %w", rel, err)
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("copying file %s: %w", rel, err)
			}
		}
	}
	return nil
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil // skip symlinks to prevent following links outside the repo
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

// copyFile copies a single file, preserving its permissions.
func copyFile(src, dst string) (retErr error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}

// rollback collects best-effort undo operations to run on failure.
type rollback struct {
	svc     *Service
	actions []func()
	armed   bool
}

// newRollback creates a new armed rollback tied to the given service.
func newRollback(svc *Service) *rollback {
	return &rollback{svc: svc, armed: true}
}

// add registers an undo operation with a label.
func (rb *rollback) add(label string, fn func() error) {
	rb.actions = append(rb.actions, func() {
		rb.svc.bestEffort(label+" rollback", fn())
	})
}

// execute runs all registered undo operations in reverse order, if still armed.
func (rb *rollback) execute() {
	if !rb.armed {
		return
	}
	for i := len(rb.actions) - 1; i >= 0; i-- {
		rb.actions[i]()
	}
}

// disarm prevents the rollback from executing (call on success).
func (rb *rollback) disarm() {
	rb.armed = false
}

// ensureParentDir creates parent directories for the given path.
func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}

// addWorktree creates parent directories and adds a worktree.
func (s *Service) addWorktree(path, branch string) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	return s.git.AddWorktree(path, branch)
}

// addWorktreeNewBranch creates parent directories and adds a worktree for a new branch.
func (s *Service) addWorktreeNewBranch(path, branch, base string) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	return s.git.AddWorktreeNewBranch(path, branch, base)
}

// cleanWorktreeParent removes the worktree's parent directory if it is empty
// and is not the worktree base directory itself.
func (s *Service) cleanWorktreeParent(wtPath string) {
	parent := filepath.Dir(wtPath)
	base := filepath.Join(s.cp.RepoRoot, s.cp.WorktreeDir)
	if parent != base {
		s.bestEffort("cleanWorktreeParent", os.Remove(parent))
	}
}
