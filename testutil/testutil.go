package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// RepoBuilder constructs temporary git repositories for testing.
type RepoBuilder struct {
	t         *testing.T
	remote    string
	branches  []string
	worktrees []string
}

// NewRepo creates a RepoBuilder for the given test.
func NewRepo(t *testing.T) *RepoBuilder {
	t.Helper()
	return &RepoBuilder{t: t}
}

// WithRemote sets the origin remote URL.
func (b *RepoBuilder) WithRemote(url string) *RepoBuilder {
	b.remote = url
	return b
}

// WithBranch adds a branch to be created.
func (b *RepoBuilder) WithBranch(name string) *RepoBuilder {
	b.branches = append(b.branches, name)
	return b
}

// WithWorktree adds a branch and a worktree for it.
func (b *RepoBuilder) WithWorktree(branch string) *RepoBuilder {
	b.branches = append(b.branches, branch)
	b.worktrees = append(b.worktrees, branch)
	return b
}

// Build creates the repository and returns the root directory path.
func (b *RepoBuilder) Build() string {
	b.t.Helper()

	dir := b.t.TempDir()

	run(b.t, dir, "git", "init", "-b", "main")
	run(b.t, dir, "git", "config", "user.email", "test@example.com")
	run(b.t, dir, "git", "config", "user.name", "Test")

	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		b.t.Fatal(err)
	}
	run(b.t, dir, "git", "add", ".")
	run(b.t, dir, "git", "commit", "-m", "initial commit")

	if b.remote != "" {
		run(b.t, dir, "git", "remote", "add", "origin", b.remote)
	}

	created := make(map[string]bool)
	for _, branch := range b.branches {
		if !created[branch] {
			run(b.t, dir, "git", "branch", branch)
			created[branch] = true
		}
	}

	for _, branch := range b.worktrees {
		wtDir := filepath.Join(dir, ".worktrees", branch)
		run(b.t, dir, "git", "worktree", "add", wtDir, branch)
	}

	return dir
}

// GitRepo creates a temporary git repository with an initial commit.
// The directory is cleaned up when the test finishes.
func GitRepo(t *testing.T) string {
	t.Helper()
	return NewRepo(t).Build()
}

// GitRepoWithRemote creates a temporary git repository with a configured remote URL.
func GitRepoWithRemote(t *testing.T, remoteURL string) string {
	t.Helper()
	return NewRepo(t).WithRemote(remoteURL).Build()
}

// GitRepoWithBranch creates a temporary git repository with an additional branch.
func GitRepoWithBranch(t *testing.T, branch string) string {
	t.Helper()
	return NewRepo(t).WithBranch(branch).Build()
}

// GitRepoWithWorktree creates a temporary git repository with a worktree for the given branch.
func GitRepoWithWorktree(t *testing.T, branch string) string {
	t.Helper()
	return NewRepo(t).WithWorktree(branch).Build()
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %s: %v", name, args, out, err)
	}
}
