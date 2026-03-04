package resource

import (
	"fmt"
	"os"
	"path/filepath"
)

// requireNotDefaultBranch returns a DefaultBranchError if branch is the default branch.
func (s *Service) requireNotDefaultBranch(branch, action string) error {
	if branch == s.params.DefaultBranch {
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

// ensureWorktree ensures a worktree exists for the given branch.
// Returns (path, wasCreated, error).
// For the default branch, verifies the repo root has the correct branch checked out,
// switching automatically if clean, or returning an error if uncommitted changes exist.
func (s *Service) ensureWorktree(branch string) (string, bool, error) {
	if branch == s.params.DefaultBranch {
		if err := s.ensureDefaultBranchCheckout(); err != nil {
			return "", false, err
		}
		return s.params.RepoRoot, false, nil
	}
	return s.findOrCreateWorktree(branch)
}

// ensureDefaultBranchCheckout verifies that the repo root has the default branch checked out.
// If a different branch is checked out and the working tree is clean, it switches automatically.
// If uncommitted changes exist, it returns a RepoRootBranchMismatchError.
func (s *Service) ensureDefaultBranchCheckout() error {
	current, err := s.git.CurrentBranch(s.params.RepoRoot)
	if err != nil {
		return fmt.Errorf("checking current branch at repo root: %w", err)
	}
	if current == s.params.DefaultBranch {
		return nil
	}

	dirty, err := s.git.HasUncommittedChanges(s.params.RepoRoot)
	if err != nil {
		return fmt.Errorf("checking uncommitted changes at repo root: %w", err)
	}
	if dirty {
		return &RepoRootBranchMismatchError{Expected: s.params.DefaultBranch, Actual: current}
	}

	if err := s.git.SwitchBranch(s.params.RepoRoot, s.params.DefaultBranch); err != nil {
		return fmt.Errorf("switching repo root to %s: %w", s.params.DefaultBranch, err)
	}
	return nil
}

// findOrCreateWorktree returns the existing worktree for branch, or creates one.
// Returns (path, wasCreated, error).
func (s *Service) findOrCreateWorktree(branch string) (string, bool, error) {
	worktrees, err := s.git.ListWorktrees()
	if err != nil {
		return "", false, fmt.Errorf("listing worktrees: %w", err)
	}
	if wt := findNonMainWorktree(worktrees, branch); wt != nil {
		return wt.Path, false, nil
	}

	path := s.params.WorktreePath(branch)
	if err := s.addWorktree(path, branch); err != nil {
		return "", false, fmt.Errorf("adding worktree: %w", err)
	}
	return path, true, nil
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
	base := filepath.Join(s.params.RepoRoot, s.params.WorktreeDir)
	if parent != base {
		s.bestEffort("cleanWorktreeParent", os.Remove(parent))
	}
}
