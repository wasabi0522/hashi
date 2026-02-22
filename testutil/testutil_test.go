package testutil

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitRepo(t *testing.T) {
	dir := GitRepo(t)
	assert.DirExists(t, filepath.Join(dir, ".git"))
}

func TestGitRepoWithRemote(t *testing.T) {
	dir := GitRepoWithRemote(t, "https://github.com/test/repo.git")
	assert.DirExists(t, filepath.Join(dir, ".git"))
}

func TestGitRepoWithBranch(t *testing.T) {
	dir := GitRepoWithBranch(t, "feature")
	assert.DirExists(t, filepath.Join(dir, ".git"))
}

func TestGitRepoWithWorktree(t *testing.T) {
	dir := GitRepoWithWorktree(t, "feature")
	assert.DirExists(t, filepath.Join(dir, ".git"))
	assert.DirExists(t, filepath.Join(dir, ".worktrees", "feature"))
}

func TestRepoBuilder(t *testing.T) {
	dir := NewRepo(t).
		WithRemote("https://github.com/test/repo.git").
		WithBranch("feat-a").
		WithBranch("feat-a"). // duplicate branch is deduplicated
		WithWorktree("feat-b").
		Build()
	assert.DirExists(t, filepath.Join(dir, ".git"))
	assert.DirExists(t, filepath.Join(dir, ".worktrees", "feat-b"))
}
