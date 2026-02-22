package git

//go:generate moq -out git_mock.go . Client

// Querier abstracts read-only git operations needed for context resolution.
type Querier interface {
	GitCommonDir() (string, error)
	SymbolicRef(ref string) (string, error)
	RemoteGetURL(remote string) (string, error)
}

// BranchReader abstracts read-only branch operations.
type BranchReader interface {
	BranchExists(name string) (bool, error)
	ListBranches() ([]string, error)
	IsMerged(branch, base string) (bool, error)
	HasUncommittedChanges(worktreePath string) (bool, error)
}

// BranchWriter abstracts write branch operations.
type BranchWriter interface {
	RenameBranch(old, new string) error
	DeleteBranch(name string) error
	DeleteBranchFrom(dir, name string) error
}

// WorktreeManager abstracts worktree operations.
type WorktreeManager interface {
	ListWorktrees() ([]Worktree, error)
	AddWorktree(path, branch string) error
	AddWorktreeNewBranch(path, branch, base string) error
	RemoveWorktree(path string) error
	RepairWorktrees() error
}

// Client abstracts git operations for testing.
type Client interface {
	Querier
	BranchReader
	BranchWriter
	WorktreeManager
}

// Worktree represents a git worktree entry.
type Worktree struct {
	Path   string
	Branch string
	// IsMain is true for the first worktree entry in `git worktree list` output,
	// which corresponds to the main working tree (the bare checkout directory).
	IsMain bool
	// Detached is true when the worktree has a detached HEAD (no branch).
	Detached bool
}
