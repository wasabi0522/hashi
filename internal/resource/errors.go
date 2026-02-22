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
