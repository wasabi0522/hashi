package resource

import "fmt"

// BranchNotFoundError indicates the specified branch does not exist.
type BranchNotFoundError struct {
	Branch string
}

func (e *BranchNotFoundError) Error() string {
	return fmt.Sprintf("branch '%s' does not exist", e.Branch)
}

// BranchExistsError indicates the specified branch already exists.
type BranchExistsError struct {
	Branch string
}

func (e *BranchExistsError) Error() string {
	return fmt.Sprintf("branch '%s' already exists", e.Branch)
}

// DefaultBranchError indicates an operation cannot be performed on the default branch.
type DefaultBranchError struct {
	Action string
}

func (e *DefaultBranchError) Error() string {
	return fmt.Sprintf("cannot %s default branch", e.Action)
}

// RepoRootBranchMismatchError indicates the repo root has a different branch checked out
// than the default branch, and cannot be automatically corrected.
type RepoRootBranchMismatchError struct {
	Expected string
	Actual   string
}

func (e *RepoRootBranchMismatchError) Error() string {
	return fmt.Sprintf("repository root has '%s' checked out instead of '%s'; commit or stash changes and run: git -C <repo-root> switch %s", e.Actual, e.Expected, e.Expected)
}
