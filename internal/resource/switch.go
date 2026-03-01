package resource

import (
	"context"
	"fmt"
)

// SwitchParams holds parameters for the Switch operation.
type SwitchParams struct {
	Branch string
}

// Switch switches to an existing branch, creating worktree and tmux resources as needed.
func (s *Service) Switch(ctx context.Context, p SwitchParams) (*OperationResult, error) {
	if err := ValidateBranchName(p.Branch); err != nil {
		return nil, err
	}
	if err := s.requireBranchExists(p.Branch); err != nil {
		return nil, err
	}

	wtPath, wtCreated, err := s.ensureWorktree(p.Branch)
	if err != nil {
		return nil, fmt.Errorf("ensuring worktree: %w", err)
	}

	// Copy files before creating tmux (hooks may depend on them).
	// No rollback: Switch does not own the worktree lifecycle.
	if wtCreated {
		if err := s.copyFiles(wtPath); err != nil {
			return nil, err
		}
	}

	initCmd := s.buildInitCmd(wtCreated)
	if err := s.ensureTmux(s.cp.SessionName, p.Branch, wtPath, initCmd); err != nil {
		return nil, fmt.Errorf("ensuring tmux: %w", err)
	}

	return s.finalizeOperation(OpSwitch, p.Branch, wtPath, wtCreated)
}
