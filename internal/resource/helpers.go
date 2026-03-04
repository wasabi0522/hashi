package resource

import (
	"path/filepath"
	"strings"

	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

// findBy returns a pointer to the first element where key(elem) == target, or nil.
func findBy[T any](items []T, key func(T) string, target string) *T {
	for i := range items {
		if key(items[i]) == target {
			return &items[i]
		}
	}
	return nil
}

// findWindow returns the window matching the given name, or nil.
func findWindow(windows []tmux.Window, name string) *tmux.Window {
	return findBy(windows, func(w tmux.Window) string { return w.Name }, name)
}

// findWorktree returns the worktree matching the given branch, or nil.
func findWorktree(worktrees []git.Worktree, branch string) *git.Worktree {
	return findBy(worktrees, func(wt git.Worktree) string { return wt.Branch }, branch)
}

// findNonMainWorktree returns the non-main worktree for branch, or nil.
func findNonMainWorktree(worktrees []git.Worktree, branch string) *git.Worktree {
	if wt := findWorktree(worktrees, branch); wt != nil && !wt.IsMain {
		return wt
	}
	return nil
}

// toSet converts a slice to a set (map[T]struct{}).
func toSet[T comparable](items []T) map[T]struct{} {
	m := make(map[T]struct{}, len(items))
	for _, item := range items {
		m[item] = struct{}{}
	}
	return m
}

// toMap converts a slice to a map using a key function.
func toMap[T any, K comparable](items []T, key func(T) K) map[K]T {
	m := make(map[K]T, len(items))
	for _, item := range items {
		m[key(item)] = item
	}
	return m
}

// shellQuote wraps s in POSIX single quotes, escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// containedIn reports whether child is under parent after cleaning both paths.
func containedIn(parent, child string) bool {
	p := filepath.Clean(parent) + string(filepath.Separator)
	c := filepath.Clean(child) + string(filepath.Separator)
	return strings.HasPrefix(c, p)
}
