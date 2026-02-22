package resource

import (
	"context"
	"fmt"
)

// NewParams holds parameters for the New operation.
type NewParams struct {
	Branch string
	Base   string
}

// New creates or switches to a branch with its worktree and tmux window.
func (s *Service) New(ctx context.Context, p NewParams) (*OperationResult, error) {
	if err := ValidateBranchName(p.Branch); err != nil {
		return nil, err
	}
	if p.Base != "" {
		if err := ValidateBranchName(p.Base); err != nil {
			return nil, fmt.Errorf("invalid base branch: %w", err)
		}
	}

	branchExists, err := s.git.BranchExists(p.Branch)
	if err != nil {
		return nil, fmt.Errorf("checking branch: %w", err)
	}

	if branchExists && p.Base != "" {
		return nil, fmt.Errorf("cannot specify base branch for existing branch '%s'", p.Branch)
	}

	var wtPath string
	var wtCreated bool
	var branchCreated bool

	if branchExists {
		wtPath, wtCreated, err = s.ensureWorktree(p.Branch)
		if err != nil {
			return nil, fmt.Errorf("ensuring worktree: %w", err)
		}
	} else {
		base := p.Base
		if base == "" {
			base = s.cp.DefaultBranch
		}
		if err := s.requireBranchExists(base); err != nil {
			return nil, err
		}

		wtPath = s.cp.WorktreePath(p.Branch)
		if err := s.addWorktreeNewBranch(wtPath, p.Branch, base); err != nil {
			return nil, fmt.Errorf("creating worktree: %w", err)
		}
		wtCreated = true
		branchCreated = true
	}

	// Copy files before creating tmux (hooks may depend on them)
	if wtCreated {
		if err := s.copyFiles(wtPath); err != nil {
			s.rollbackNew(wtCreated, branchCreated, wtPath, p.Branch)
			return nil, err
		}
	}

	// Ensure tmux (best-effort rollback on failure)
	initCmd := s.buildInitCmd(wtCreated)
	if err := s.ensureTmux(s.cp.SessionName, p.Branch, wtPath, initCmd); err != nil {
		s.rollbackNew(wtCreated, branchCreated, wtPath, p.Branch)
		return nil, err
	}

	return s.finalizeOperation(OpNew, p.Branch, wtPath, wtCreated)
}

// rollbackNew performs best-effort cleanup of newly created resources.
func (s *Service) rollbackNew(wtCreated, branchCreated bool, wtPath, branch string) {
	if wtCreated {
		s.bestEffort("RemoveWorktree", s.git.RemoveWorktree(wtPath))
	}
	if branchCreated {
		s.bestEffort("DeleteBranch", s.git.DeleteBranch(branch))
	}
}
