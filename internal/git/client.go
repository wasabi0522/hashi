package git

import (
	"strings"

	"github.com/wasabi0522/hashi/internal/exec"
)

var _ Client = (*client)(nil)

type client struct {
	exec exec.Executor
}

// NewClient creates a git Client backed by the given Executor.
func NewClient(exec exec.Executor) Client {
	return &client{exec: exec}
}

func (c *client) GitCommonDir() (string, error) {
	return c.exec.Output("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
}

func (c *client) SymbolicRef(ref string) (string, error) {
	return c.exec.Output("git", "symbolic-ref", ref)
}

func (c *client) RemoteGetURL(remote string) (string, error) {
	return c.exec.Output("git", "remote", "get-url", remote)
}

func (c *client) ListBranches() ([]string, error) {
	out, err := c.exec.Output("git", "branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	var branches []string
	for line := range strings.SplitSeq(out, "\n") {
		branches = append(branches, line)
	}
	return branches, nil
}

func (c *client) CurrentBranch(dir string) (string, error) {
	return c.exec.Output("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
}

func (c *client) SwitchBranch(dir, branch string) error {
	return c.exec.Run("git", "-C", dir, "switch", branch)
}

func (c *client) BranchExists(name string) (bool, error) {
	out, err := c.exec.Output("git", "branch", "--list", "--", name)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (c *client) RenameBranch(old, new string) error {
	return c.exec.Run("git", "branch", "-m", "--", old, new)
}

func (c *client) DeleteBranch(name string) error {
	return c.exec.Run("git", "branch", "-D", "--", name)
}

func (c *client) DeleteBranchFrom(dir, name string) error {
	return c.exec.Run("git", "-C", dir, "branch", "-D", "--", name)
}

func (c *client) IsMerged(branch, base string) (bool, error) {
	err := c.exec.Run("git", "merge-base", "--is-ancestor", "--", branch, base)
	if err == nil {
		return true, nil
	}
	if exec.IsExitCode(err, 1) {
		return false, nil
	}
	return false, err
}

func (c *client) HasUncommittedChanges(worktreePath string) (bool, error) {
	out, err := c.exec.Output("git", "-C", worktreePath, "status", "--porcelain", "--")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

func (c *client) ListWorktrees() ([]Worktree, error) {
	out, err := c.exec.Output("git", "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(out), nil
}

func (c *client) AddWorktree(path, branch string) error {
	return c.exec.Run("git", "worktree", "add", "--", path, branch)
}

func (c *client) AddWorktreeNewBranch(path, branch, base string) error {
	return c.exec.Run("git", "worktree", "add", "-b", branch, "--", path, base)
}

func (c *client) RemoveWorktree(path string) error {
	return c.exec.Run("git", "worktree", "remove", "--force", path)
}

func (c *client) RepairWorktrees() error {
	return c.exec.Run("git", "worktree", "repair")
}

// parseWorktreeList parses the porcelain output of `git worktree list --porcelain`.
func parseWorktreeList(output string) []Worktree {
	if output == "" {
		return nil
	}

	var worktrees []Worktree
	blocks := strings.Split(output, "\n\n")

	for i, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		var wt Worktree
		wt.IsMain = (i == 0)

		for line := range strings.SplitSeq(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				wt.Path = strings.TrimPrefix(line, "worktree ")
			case strings.HasPrefix(line, "branch "):
				ref := strings.TrimPrefix(line, "branch ")
				wt.Branch = strings.TrimPrefix(ref, "refs/heads/")
			case line == "detached":
				wt.Detached = true
			}
		}

		if wt.Path != "" {
			worktrees = append(worktrees, wt)
		}
	}

	return worktrees
}
