package resource

import (
	"context"
	"fmt"
)

// NewParams holds parameters for the New operation.
type NewParams struct {
	Branch  string
	Base    string
	NoHooks bool
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

	branches, err := s.git.ListBranches()
	if err != nil {
		return nil, fmt.Errorf("listing branches: %w", err)
	}
	branchSet := toSet(branches)
	_, branchExists := branchSet[p.Branch]

	if branchExists && p.Base != "" {
		return nil, fmt.Errorf("cannot specify base branch for existing branch '%s'", p.Branch)
	}

	rb := newRollback(s)
	defer rb.execute()

	var wtPath string
	var wtCreated bool

	if branchExists {
		wtPath, wtCreated, err = s.ensureWorktree(p.Branch)
		if err != nil {
			return nil, fmt.Errorf("ensuring worktree: %w", err)
		}
		if wtCreated {
			rb.add("RemoveWorktree", func() error { return s.git.RemoveWorktree(wtPath) })
		}
	} else {
		base := p.Base
		if base == "" {
			base = s.params.DefaultBranch
		}
		if _, ok := branchSet[base]; !ok {
			return nil, &BranchNotFoundError{Branch: base}
		}

		wtPath = s.params.WorktreePath(p.Branch)
		if err := s.addWorktreeNewBranch(wtPath, p.Branch, base); err != nil {
			return nil, fmt.Errorf("creating worktree: %w", err)
		}
		wtCreated = true
		rb.add("RemoveWorktree", func() error { return s.git.RemoveWorktree(wtPath) })
		rb.add("DeleteBranch", func() error { return s.git.DeleteBranch(p.Branch) })
	}

	// Copy files before creating tmux (hooks may depend on them)
	if wtCreated {
		if err := s.copyFiles(wtPath); err != nil {
			return nil, err
		}
	}

	// Ensure tmux (best-effort rollback on failure)
	initCmd := s.buildInitCmd(wtCreated, p.NoHooks)
	if err := s.ensureTmux(s.params.SessionName, p.Branch, wtPath, initCmd); err != nil {
		return nil, err
	}

	rb.disarm()
	return s.finalizeOperation(OpNew, p.Branch, wtPath, wtCreated)
}
