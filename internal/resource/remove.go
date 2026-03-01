package resource

import (
	"context"
	"fmt"
)

// RemoveCheck holds the state information for a branch removal.
type RemoveCheck struct {
	Branch         string
	HasBranch      bool
	HasWorktree    bool
	WorktreePath   string
	HasWindow      bool
	IsActive       bool
	HasUncommitted bool
	IsUnmerged     bool
}

// HasResources reports whether any managed resource exists for this branch.
func (c RemoveCheck) HasResources() bool {
	return c.HasBranch || c.HasWorktree || c.HasWindow
}

// NeedsWarning reports whether the removal should warn the user about data loss.
func (c RemoveCheck) NeedsWarning() bool {
	return c.HasUncommitted || c.IsUnmerged
}

// PrepareRemove checks the state of a branch for removal.
// Returns an error if no resources exist for the branch.
func (s *Service) PrepareRemove(ctx context.Context, branch string) (RemoveCheck, error) {
	if err := ValidateBranchName(branch); err != nil {
		return RemoveCheck{}, err
	}
	if err := s.requireNotDefaultBranch(branch, "remove"); err != nil {
		return RemoveCheck{}, err
	}

	check := RemoveCheck{Branch: branch}

	exists, err := s.git.BranchExists(branch)
	if err != nil {
		return RemoveCheck{}, fmt.Errorf("checking branch: %w", err)
	}
	check.HasBranch = exists

	worktrees, err := s.git.ListWorktrees()
	if err != nil {
		return RemoveCheck{}, fmt.Errorf("listing worktrees: %w", err)
	}
	if wt := findWorktree(worktrees, branch); wt != nil {
		check.HasWorktree = true
		check.WorktreePath = wt.Path
	}

	if w := findWindow(s.listWindowsSafe(s.cp.SessionName), branch); w != nil {
		check.HasWindow = true
		check.IsActive = w.Active
	}

	if !check.HasResources() {
		return RemoveCheck{}, &BranchNotFoundError{Branch: branch}
	}

	if check.HasBranch && check.HasWorktree {
		// Defaults to false on failure: safe because the confirmation prompt
		// still protects the user even without this warning.
		check.HasUncommitted, _ = s.git.HasUncommittedChanges(check.WorktreePath)
	}
	if check.HasBranch {
		// Defaults to unmerged=true on failure (via !merged where merged=false):
		// this is the safe side, warning the user even when the check itself fails.
		merged, _ := s.git.IsMerged(branch, s.cp.DefaultBranch)
		check.IsUnmerged = !merged
	}

	return check, nil
}

// RemoveResult holds the result of a branch removal.
type RemoveResult struct {
	BranchDeleted   bool
	WorktreeRemoved bool
	WindowKilled    bool
	SessionKilled   bool
}

// ExecuteRemove removes the resources for a branch.
func (s *Service) ExecuteRemove(ctx context.Context, check RemoveCheck) (*RemoveResult, error) {
	result := &RemoveResult{}

	// Switch from active window if needed
	if check.IsActive {
		if err := s.ensureTmux(s.cp.SessionName, s.cp.DefaultBranch, s.cp.RepoRoot, ""); err != nil {
			return nil, fmt.Errorf("switching to default branch: %w", err)
		}
		if s.tmux.IsInsideTmux() {
			s.bestEffort("SwitchClient", s.tmux.SwitchClient(s.cp.SessionName, s.cp.DefaultBranch))
		}
	}

	// Remove worktree and delete branch before killing the window.
	// When the user runs "hashi remove" from the active window,
	// KillWindow sends SIGHUP to this process, so git operations must complete first.
	if check.HasWorktree {
		if err := s.git.RemoveWorktree(check.WorktreePath); err != nil {
			return nil, fmt.Errorf("removing worktree: %w", err)
		}
		result.WorktreeRemoved = true
		s.cleanWorktreeParent(check.WorktreePath)
	}

	if check.HasBranch {
		// Use DeleteBranchFrom with repo root to avoid depending on CWD,
		// which may no longer exist after worktree removal.
		if err := s.git.DeleteBranchFrom(s.cp.RepoRoot, check.Branch); err != nil {
			return nil, fmt.Errorf("deleting branch: %w", err)
		}
		result.BranchDeleted = true
	}

	// Kill window last: may terminate this process via SIGHUP if it was the active window.
	if check.HasWindow {
		if err := s.tmux.KillWindow(s.cp.SessionName, check.Branch); err != nil {
			return nil, fmt.Errorf("killing window: %w", err)
		}
		result.WindowKilled = true
	}

	// Best-effort: kill session if no windows remain.
	if ok, _ := s.tmux.HasSession(s.cp.SessionName); ok {
		windows, lErr := s.tmux.ListWindows(s.cp.SessionName)
		s.bestEffort("ListWindows", lErr)
		if len(windows) == 0 {
			if err := s.tmux.KillSession(s.cp.SessionName); err == nil {
				result.SessionKilled = true
			}
		}
	}

	return result, nil
}
