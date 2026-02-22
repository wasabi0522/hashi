package resource

import (
	"context"
	"fmt"
	"os"
)

// RenameParams holds parameters for the Rename operation.
type RenameParams struct {
	Old string
	New string
}

// Rename renames a branch along with its worktree and tmux window.
func (s *Service) Rename(ctx context.Context, p RenameParams) (*OperationResult, error) {
	// Validation
	if err := s.requireNotDefaultBranch(p.Old, "rename"); err != nil {
		return nil, err
	}
	if err := s.requireBranchExists(p.Old); err != nil {
		return nil, err
	}
	if err := s.requireBranchNotExists(p.New); err != nil {
		return nil, err
	}

	// Rename branch
	if err := s.git.RenameBranch(p.Old, p.New); err != nil {
		return nil, fmt.Errorf("renaming branch: %w", err)
	}

	rb := newRollback(s)
	defer rb.execute()
	rb.add("RenameBranch", func() error { return s.git.RenameBranch(p.New, p.Old) })

	// Handle worktree
	wtPath, wtCreated, err := s.renameWorktree(p)
	if err != nil {
		return nil, fmt.Errorf("renaming worktree: %w", err)
	}

	if wtCreated {
		if err := s.copyFiles(wtPath); err != nil {
			return nil, err
		}
	}

	// Handle tmux
	initCmd := s.buildInitCmd(wtCreated)
	s.renameTmuxWindow(p, wtPath, initCmd)

	// Best-effort connect to the renamed window (aligns with New/Switch behavior)
	s.bestEffort("connect", s.connect(s.cp.SessionName, p.New))

	rb.disarm()
	return &OperationResult{Operation: OpRename, Branch: p.New, WorktreePath: wtPath, Created: wtCreated}, nil
}

// renameWorktree moves or creates the worktree for the renamed branch.
// It searches for p.New (not p.Old) because the git branch has already been
// renamed at this point, so git reports the worktree under the new branch name.
// Returns (path, wasCreated, error).
func (s *Service) renameWorktree(p RenameParams) (string, bool, error) {
	worktrees, err := s.git.ListWorktrees()
	if err != nil {
		return "", false, err
	}
	if wt := findWorktree(worktrees, p.New); wt != nil {
		wtPath, err := s.moveWorktree(p, wt.Path)
		return wtPath, false, err
	}
	return s.findOrCreateWorktree(p.New)
}

// moveWorktree moves the worktree directory from its current location to the new path.
func (s *Service) moveWorktree(p RenameParams, oldPath string) (string, error) {
	newPath := s.cp.WorktreePath(p.New)

	if err := ensureParentDir(newPath); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", fmt.Errorf("moving worktree: %w", err)
	}
	if err := s.git.RepairWorktrees(); err != nil {
		s.bestEffort("os.Rename rollback", os.Rename(newPath, oldPath))
		return "", fmt.Errorf("repairing worktrees: %w", err)
	}

	s.cleanWorktreeParent(oldPath)

	return newPath, nil
}

// renameTmuxWindow updates the tmux window for the renamed branch.
// All tmux operations are best-effort: failures are silently ignored.
func (s *Service) renameTmuxWindow(p RenameParams, wtPath, initCmd string) {
	windows := s.listWindowsSafe(s.cp.SessionName)
	if windows == nil {
		return
	}
	if findWindow(windows, p.Old) != nil {
		s.bestEffort("RenameWindow", s.tmux.RenameWindow(s.cp.SessionName, p.Old, p.New))
		s.sendCd(s.cp.SessionName, p.New, wtPath)
		return
	}
	s.bestEffort("NewWindow", s.tmux.NewWindow(s.cp.SessionName, p.New, wtPath, initCmd))
}
