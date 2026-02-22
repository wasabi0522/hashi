package resource

import (
	"context"

	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

// classifyWorktreeStatus returns the status of a worktree entry.
func classifyWorktreeStatus(wt git.Worktree, branchSet map[string]struct{}) Status {
	if wt.IsMain {
		return StatusOK
	}
	if _, ok := branchSet[wt.Branch]; !ok {
		return StatusOrphanedWorktree
	}
	return StatusOK
}

// classifyWindowOnlyStatus returns the status of a window that has no matching worktree.
func classifyWindowOnlyStatus(name string, branchSet map[string]struct{}) Status {
	if _, ok := branchSet[name]; ok {
		return StatusWorktreeMissing
	}
	return StatusOrphanedWindow
}

// CollectState gathers the combined state of worktrees and tmux windows.
// It assumes that the main worktree always has a branch (never detached HEAD)
// and that its branch appears in the branch list. Tmux session/window lookup
// is best-effort: if the session does not exist, all windows are treated as absent.
func (s *Service) CollectState(ctx context.Context) ([]State, error) {
	worktrees, err := s.git.ListWorktrees()
	if err != nil {
		return nil, err
	}

	branches, err := s.git.ListBranches()
	if err != nil {
		return nil, err
	}
	branchSet := toSet(branches)

	windows := s.listWindowsSafe(s.cp.SessionName)
	winMap := toMap(windows, func(w tmux.Window) string { return w.Name })

	seen := make(map[string]struct{})
	var states []State

	// Process worktrees
	for _, wt := range worktrees {
		if wt.Detached {
			continue // skip detached HEAD
		}
		name := wt.Branch
		seen[name] = struct{}{}

		win, hasWin := winMap[name]

		states = append(states, State{
			Branch:    name,
			Worktree:  wt.Path,
			Window:    hasWin,
			Active:    hasWin && win.Active,
			IsDefault: name == s.cp.DefaultBranch,
			Status:    classifyWorktreeStatus(wt, branchSet),
		})
	}

	// Process windows without worktrees
	for _, w := range windows {
		if _, ok := seen[w.Name]; ok {
			continue
		}

		states = append(states, State{
			Branch: w.Name,
			Window: true,
			Active: w.Active,
			Status: classifyWindowOnlyStatus(w.Name, branchSet),
		})
	}

	return states, nil
}
