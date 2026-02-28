package resource_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hashiexec "github.com/wasabi0522/hashi/internal/exec"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/resource"
	"github.com/wasabi0522/hashi/internal/tmux"
	"github.com/wasabi0522/hashi/testutil"
)

func skipIfNoTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not found, skipping integration test")
	}
}

func tmuxKillSession(t *testing.T, session string) {
	t.Helper()
	_ = exec.Command("tmux", "kill-session", "-t", session).Run()
}

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s: %v", args, out, err)
	}
}

func newTestService(t *testing.T, cp resource.CommonParams) (*resource.Service, git.Client) {
	t.Helper()
	e := hashiexec.NewDefaultExecutor()
	g := git.NewClient(e)
	return resource.NewService(e, g, tmux.NewClient(e), resource.WithCommonParams(cp)), g
}

// logNonConnectError logs an error from New/Switch/Rename if it is not the
// expected AttachSession failure in non-interactive test environments.
func logNonConnectError(t *testing.T, op string, err error) {
	t.Helper()
	if err != nil {
		t.Logf("%s returned error (expected in non-interactive env): %v", op, err)
	}
}

// testCommonParams returns a CommonParams with sensible defaults for integration testing.
func testCommonParams(repoRoot, session string) resource.CommonParams {
	return resource.CommonParams{
		RepoRoot:      repoRoot,
		WorktreeDir:   ".worktrees",
		DefaultBranch: "main",
		SessionName:   session,
	}
}

// setupTmuxTest prepares a tmux integration test by checking for tmux, clearing TMUX env,
// generating a unique session name with the given suffix, and registering cleanup.
func setupTmuxTest(t *testing.T, suffix string) string {
	t.Helper()
	skipIfNoTmux(t)
	t.Setenv("TMUX", "")
	session := "test-integration-" + suffix + "-" + t.Name()
	t.Cleanup(func() { tmuxKillSession(t, session) })
	return session
}

func TestIntegration_NewAndList(t *testing.T) {
	session := setupTmuxTest(t, "new")

	repoRoot := testutil.GitRepo(t)
	t.Chdir(repoRoot)

	cp := testCommonParams(repoRoot, session)
	svc, g := newTestService(t, cp)

	_, err := svc.New(context.Background(), resource.NewParams{Branch: "feature-test"})
	logNonConnectError(t, "New", err)

	// Verify worktree was created
	wtPath := filepath.Join(repoRoot, ".worktrees", "feature-test")
	_, err = os.Stat(wtPath)
	require.NoError(t, err, "worktree directory should exist")

	// Verify branch exists
	exists, err := g.BranchExists("feature-test")
	require.NoError(t, err)
	assert.True(t, exists, "branch should exist")

	// Collect state and verify
	states, err := svc.CollectState(context.Background())
	require.NoError(t, err)

	var found bool
	for _, s := range states {
		if s.Branch == "feature-test" {
			found = true
			assert.Equal(t, resource.StatusOK, s.Status)
			assert.True(t, s.Window)
			break
		}
	}
	assert.True(t, found, "feature-test should appear in state list")
}

func TestIntegration_SwitchExistingBranch(t *testing.T) {
	session := setupTmuxTest(t, "sw")

	repoRoot := testutil.GitRepoWithBranch(t, "existing-branch")
	t.Chdir(repoRoot)

	cp := testCommonParams(repoRoot, session)
	svc, _ := newTestService(t, cp)

	_, err := svc.Switch(context.Background(), resource.SwitchParams{Branch: "existing-branch"})
	logNonConnectError(t, "Switch", err)

	// Verify worktree was created
	wtPath := filepath.Join(repoRoot, ".worktrees", "existing-branch")
	_, err = os.Stat(wtPath)
	require.NoError(t, err)
}

func TestIntegration_RenameAndRemove(t *testing.T) {
	session := setupTmuxTest(t, "rr")

	repoRoot := testutil.GitRepo(t)
	t.Chdir(repoRoot)

	cp := testCommonParams(repoRoot, session)
	svc, g := newTestService(t, cp)

	_, err := svc.New(context.Background(), resource.NewParams{Branch: "to-rename"})
	logNonConnectError(t, "New", err)

	// Verify branch was created before proceeding
	exists, err := g.BranchExists("to-rename")
	require.NoError(t, err)
	require.True(t, exists)

	// Rename
	_, err = svc.Rename(context.Background(), resource.RenameParams{
		Old: "to-rename",
		New: "renamed",
	})
	require.NoError(t, err)

	// Verify old branch/worktree gone, new ones exist
	exists, _ = g.BranchExists("to-rename")
	assert.False(t, exists)
	exists, _ = g.BranchExists("renamed")
	assert.True(t, exists)

	newWtPath := filepath.Join(repoRoot, ".worktrees", "renamed")
	_, err = os.Stat(newWtPath)
	require.NoError(t, err)

	// Remove
	check, err := svc.PrepareRemove(context.Background(), "renamed")
	require.NoError(t, err)

	_, err = svc.ExecuteRemove(context.Background(), check)
	require.NoError(t, err)

	exists, _ = g.BranchExists("renamed")
	assert.False(t, exists)
}

func TestIntegration_RemoveFromInsideWorktree(t *testing.T) {
	session := setupTmuxTest(t, "rmcwd")

	repoRoot := testutil.GitRepo(t)
	t.Chdir(repoRoot)

	cp := testCommonParams(repoRoot, session)
	svc, g := newTestService(t, cp)

	_, err := svc.New(context.Background(), resource.NewParams{Branch: "cwd-test"})
	logNonConnectError(t, "New", err)

	exists, err := g.BranchExists("cwd-test")
	require.NoError(t, err)
	require.True(t, exists)

	// Move CWD into the worktree that will be deleted
	wtPath := filepath.Join(repoRoot, ".worktrees", "cwd-test")
	t.Chdir(wtPath)

	// Remove should succeed even though CWD is inside the deleted worktree
	check, err := svc.PrepareRemove(context.Background(), "cwd-test")
	require.NoError(t, err)

	result, err := svc.ExecuteRemove(context.Background(), check)
	require.NoError(t, err)
	assert.True(t, result.WorktreeRemoved)
	assert.True(t, result.BranchDeleted)

	exists, _ = g.BranchExists("cwd-test")
	assert.False(t, exists)
}

func TestIntegration_ListBranches(t *testing.T) {
	repoRoot := testutil.GitRepo(t)

	t.Chdir(repoRoot)

	cp := testCommonParams(repoRoot, "dummy")
	_, g := newTestService(t, cp)

	// Create some branches
	gitCmd(t, repoRoot, "branch", "feature-a")
	gitCmd(t, repoRoot, "branch", "feature-b")

	branches, err := g.ListBranches()
	require.NoError(t, err)

	branchStr := strings.Join(branches, ",")
	assert.Contains(t, branchStr, "main")
	assert.Contains(t, branchStr, "feature-a")
	assert.Contains(t, branchStr, "feature-b")
}

// --- hashi new ---

func TestIntegration_NewWithBase(t *testing.T) {
	session := setupTmuxTest(t, "newbase")

	repoRoot := testutil.GitRepo(t)

	// Add a file to main so we can verify base inheritance
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "base_file.txt"), []byte("from base"), 0644))
	gitCmd(t, repoRoot, "add", "base_file.txt")
	gitCmd(t, repoRoot, "commit", "-m", "add base file")

	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, session)
	svc, _ := newTestService(t, cp)

	_, err := svc.New(context.Background(), resource.NewParams{Branch: "from-base", Base: "main"})
	logNonConnectError(t, "New", err)

	wtPath := filepath.Join(repoRoot, ".worktrees", "from-base")
	_, err = os.Stat(wtPath)
	require.NoError(t, err, "worktree directory should exist")

	// Verify file inherited from base
	_, err = os.Stat(filepath.Join(wtPath, "base_file.txt"))
	assert.NoError(t, err, "file from base branch should exist in worktree")
}

func TestIntegration_NewExistingBranch(t *testing.T) {
	session := setupTmuxTest(t, "newexist")

	repoRoot := testutil.GitRepoWithBranch(t, "existing-branch")
	t.Chdir(repoRoot)

	// Worktree should not exist yet
	wtPath := filepath.Join(repoRoot, ".worktrees", "existing-branch")
	_, err := os.Stat(wtPath)
	require.True(t, os.IsNotExist(err), "worktree should not exist yet")

	cp := testCommonParams(repoRoot, session)
	svc, _ := newTestService(t, cp)

	_, err = svc.New(context.Background(), resource.NewParams{Branch: "existing-branch"})
	logNonConnectError(t, "New", err)

	_, err = os.Stat(wtPath)
	assert.NoError(t, err, "worktree should be auto-created for existing branch")
}

func TestIntegration_NewSlashBranch(t *testing.T) {
	session := setupTmuxTest(t, "newslash")

	repoRoot := testutil.GitRepo(t)
	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, session)
	svc, _ := newTestService(t, cp)

	_, err := svc.New(context.Background(), resource.NewParams{Branch: "feature/login"})
	logNonConnectError(t, "New", err)

	// Verify nested worktree directory
	wtPath := filepath.Join(repoRoot, ".worktrees", "feature", "login")
	_, err = os.Stat(wtPath)
	assert.NoError(t, err, "slash branch should create nested worktree directory")
}

func TestIntegration_NewPostNewHooks(t *testing.T) {
	session := setupTmuxTest(t, "newhook")

	repoRoot := testutil.GitRepo(t)
	t.Chdir(repoRoot)

	cp := testCommonParams(repoRoot, session)
	cp.PostNewHooks = []string{"echo hook_ran"}
	svc, _ := newTestService(t, cp)

	// Hooks are passed as tmux init commands (sh -c + for loop).
	// Actual initCmd behavior is tested in unit tests (TestBuildInitCmd).
	_, err := svc.New(context.Background(), resource.NewParams{Branch: "hook-test"})
	logNonConnectError(t, "New", err)
}

func TestIntegration_NewErrors(t *testing.T) {
	repoRoot := testutil.GitRepoWithBranch(t, "existing-branch")
	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	t.Run("existing branch with base", func(t *testing.T) {
		_, err := svc.New(context.Background(), resource.NewParams{Branch: "existing-branch", Base: "main"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify base branch for existing branch")
	})

	t.Run("non-existent base", func(t *testing.T) {
		_, err := svc.New(context.Background(), resource.NewParams{Branch: "new-branch", Base: "no-such-base"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

// --- hashi switch ---

func TestIntegration_SwitchNonExistent(t *testing.T) {
	repoRoot := testutil.GitRepo(t)
	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	_, err := svc.Switch(context.Background(), resource.SwitchParams{Branch: "no-such-branch"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestIntegration_SwitchWithExistingWorktree(t *testing.T) {
	session := setupTmuxTest(t, "swwt")

	repoRoot := testutil.GitRepoWithWorktree(t, "existing-wt")
	t.Chdir(repoRoot)

	// Add marker file to existing worktree
	wtPath := filepath.Join(repoRoot, ".worktrees", "existing-wt")
	markerPath := filepath.Join(wtPath, "wt_marker.txt")
	require.NoError(t, os.WriteFile(markerPath, []byte("marker"), 0644))

	cp := testCommonParams(repoRoot, session)
	svc, _ := newTestService(t, cp)

	_, err := svc.Switch(context.Background(), resource.SwitchParams{Branch: "existing-wt"})
	logNonConnectError(t, "Switch", err)

	// Marker file should still exist (worktree was not recreated)
	_, err = os.Stat(markerPath)
	assert.NoError(t, err, "marker file should survive, proving worktree was reused")
}

// --- hashi rename ---

func TestIntegration_RenameMovesWorktree(t *testing.T) {
	repoRoot := testutil.GitRepoWithWorktree(t, "old-name")
	t.Chdir(repoRoot)

	// Add marker file to verify physical move
	oldWtPath := filepath.Join(repoRoot, ".worktrees", "old-name")
	require.NoError(t, os.WriteFile(filepath.Join(oldWtPath, "marker.txt"), []byte("moved"), 0644))

	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	result, err := svc.Rename(context.Background(), resource.RenameParams{Old: "old-name", New: "new-name"})
	require.NoError(t, err)
	assert.False(t, result.Created, "existing worktree should be moved, not recreated")

	// Old path should be gone
	_, err = os.Stat(oldWtPath)
	assert.True(t, os.IsNotExist(err), "old worktree directory should not exist")

	// New path should have marker file
	newWtPath := filepath.Join(repoRoot, ".worktrees", "new-name")
	_, err = os.Stat(filepath.Join(newWtPath, "marker.txt"))
	assert.NoError(t, err, "marker file should exist in moved worktree")
}

func TestIntegration_RenameWithoutWorktreeCreatesOne(t *testing.T) {
	repoRoot := testutil.GitRepoWithBranch(t, "no-wt")
	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	result, err := svc.Rename(context.Background(), resource.RenameParams{Old: "no-wt", New: "with-wt"})
	require.NoError(t, err)
	assert.True(t, result.Created, "worktree should be newly created")

	wtPath := filepath.Join(repoRoot, ".worktrees", "with-wt")
	_, err = os.Stat(wtPath)
	assert.NoError(t, err, "worktree directory should exist after rename")
}

func TestIntegration_RenameErrors(t *testing.T) {
	repoRoot := testutil.GitRepoWithBranch(t, "existing")
	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	t.Run("default branch", func(t *testing.T) {
		_, err := svc.Rename(context.Background(), resource.RenameParams{Old: "main", New: "not-main"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot rename default branch")
	})

	t.Run("non-existent branch", func(t *testing.T) {
		_, err := svc.Rename(context.Background(), resource.RenameParams{Old: "no-such-branch", New: "anything"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("target already exists", func(t *testing.T) {
		_, err := svc.Rename(context.Background(), resource.RenameParams{Old: "existing", New: "main"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

// --- hashi remove ---

func TestIntegration_RemoveErrors(t *testing.T) {
	repoRoot := testutil.GitRepo(t)
	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	t.Run("default branch", func(t *testing.T) {
		_, err := svc.PrepareRemove(context.Background(), "main")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot remove default branch")
	})

	t.Run("non-existent branch", func(t *testing.T) {
		_, err := svc.PrepareRemove(context.Background(), "no-such-branch")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestIntegration_RemoveWorktreeCleanup(t *testing.T) {
	repoRoot := testutil.GitRepoWithWorktree(t, "to-delete")
	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, "dummy")
	svc, g := newTestService(t, cp)

	check, err := svc.PrepareRemove(context.Background(), "to-delete")
	require.NoError(t, err)
	assert.True(t, check.HasBranch)
	assert.True(t, check.HasWorktree)

	result, err := svc.ExecuteRemove(context.Background(), check)
	require.NoError(t, err)
	assert.True(t, result.BranchDeleted)
	assert.True(t, result.WorktreeRemoved)

	exists, _ := g.BranchExists("to-delete")
	assert.False(t, exists, "branch should be deleted")

	wtPath := filepath.Join(repoRoot, ".worktrees", "to-delete")
	_, err = os.Stat(wtPath)
	assert.True(t, os.IsNotExist(err), "worktree directory should be deleted")
}

func TestIntegration_RemoveMultipleBranches(t *testing.T) {
	repoRoot := testutil.GitRepo(t)

	branches := []string{"multi-a", "multi-b", "multi-c"}
	for _, branch := range branches {
		gitCmd(t, repoRoot, "branch", branch)
		wtPath := filepath.Join(repoRoot, ".worktrees", branch)
		gitCmd(t, repoRoot, "worktree", "add", wtPath, branch)
	}

	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, "dummy")
	svc, g := newTestService(t, cp)

	for _, branch := range branches {
		check, err := svc.PrepareRemove(context.Background(), branch)
		require.NoError(t, err)
		_, err = svc.ExecuteRemove(context.Background(), check)
		require.NoError(t, err)
	}

	for _, branch := range branches {
		exists, _ := g.BranchExists(branch)
		assert.False(t, exists, "branch %s should be deleted", branch)
	}
}

func TestIntegration_RemoveSlashBranchCleansParent(t *testing.T) {
	repoRoot := testutil.GitRepo(t)

	gitCmd(t, repoRoot, "branch", "feature/login")
	wtPath := filepath.Join(repoRoot, ".worktrees", "feature", "login")
	require.NoError(t, os.MkdirAll(filepath.Dir(wtPath), 0755))
	gitCmd(t, repoRoot, "worktree", "add", wtPath, "feature/login")

	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	check, err := svc.PrepareRemove(context.Background(), "feature/login")
	require.NoError(t, err)

	_, err = svc.ExecuteRemove(context.Background(), check)
	require.NoError(t, err)

	// Parent directory (feature/) should also be removed if empty
	featureDir := filepath.Join(repoRoot, ".worktrees", "feature")
	_, err = os.Stat(featureDir)
	assert.True(t, os.IsNotExist(err), "empty parent directory 'feature/' should be cleaned up")
}

func TestIntegration_PrepareRemoveDetectsState(t *testing.T) {
	t.Run("unmerged", func(t *testing.T) {
		repoRoot := testutil.GitRepoWithWorktree(t, "unmerged-branch")

		t.Chdir(repoRoot)

		// Add a commit to the branch that is not in main
		wtPath := filepath.Join(repoRoot, ".worktrees", "unmerged-branch")
		require.NoError(t, os.WriteFile(filepath.Join(wtPath, "new_file.txt"), []byte("change"), 0644))
		gitCmd(t, wtPath, "add", "new_file.txt")
		gitCmd(t, wtPath, "commit", "-m", "unmerged commit")

		cp := testCommonParams(repoRoot, "dummy")
		svc, _ := newTestService(t, cp)

		check, err := svc.PrepareRemove(context.Background(), "unmerged-branch")
		require.NoError(t, err)
		assert.True(t, check.IsUnmerged, "branch with extra commit should be unmerged")
		assert.False(t, check.HasUncommitted, "no uncommitted changes")
	})

	t.Run("uncommitted", func(t *testing.T) {
		repoRoot := testutil.GitRepoWithWorktree(t, "dirty-branch")

		t.Chdir(repoRoot)

		// Add uncommitted file in worktree
		wtPath := filepath.Join(repoRoot, ".worktrees", "dirty-branch")
		require.NoError(t, os.WriteFile(filepath.Join(wtPath, "dirty_file.txt"), []byte("dirty"), 0644))

		cp := testCommonParams(repoRoot, "dummy")
		svc, _ := newTestService(t, cp)

		check, err := svc.PrepareRemove(context.Background(), "dirty-branch")
		require.NoError(t, err)
		assert.True(t, check.HasUncommitted, "worktree with untracked file should have uncommitted changes")
	})

	t.Run("clean", func(t *testing.T) {
		repoRoot := testutil.GitRepoWithWorktree(t, "clean-branch")

		t.Chdir(repoRoot)

		cp := testCommonParams(repoRoot, "dummy")
		svc, _ := newTestService(t, cp)

		check, err := svc.PrepareRemove(context.Background(), "clean-branch")
		require.NoError(t, err)
		assert.False(t, check.IsUnmerged, "branch with no extra commits should be merged")
		assert.False(t, check.HasUncommitted, "clean worktree should have no uncommitted changes")
	})
}

// --- CollectState ---

func TestIntegration_CollectStateMultipleBranches(t *testing.T) {
	repoRoot := testutil.GitRepo(t)

	for _, branch := range []string{"branch-a", "branch-b"} {
		gitCmd(t, repoRoot, "branch", branch)
		wtPath := filepath.Join(repoRoot, ".worktrees", branch)
		gitCmd(t, repoRoot, "worktree", "add", wtPath, branch)
	}

	t.Chdir(repoRoot)
	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	states, err := svc.CollectState(context.Background())
	require.NoError(t, err)

	branchSet := make(map[string]bool)
	for _, s := range states {
		branchSet[s.Branch] = true
	}

	assert.True(t, branchSet["main"], "main should always be present")
	assert.True(t, branchSet["branch-a"], "branch-a should be present")
	assert.True(t, branchSet["branch-b"], "branch-b should be present")
}

func TestIntegration_CollectStateOrphanedWorktree(t *testing.T) {
	repoRoot := testutil.GitRepoWithWorktree(t, "orphan-branch")

	t.Chdir(repoRoot)

	// Delete branch ref directly to create orphaned worktree
	// (git branch -D refuses to delete a branch checked out in a worktree)
	gitCmd(t, repoRoot, "update-ref", "-d", "refs/heads/orphan-branch")

	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	states, err := svc.CollectState(context.Background())
	require.NoError(t, err)

	var found bool
	for _, s := range states {
		if s.Branch == "orphan-branch" {
			found = true
			assert.Equal(t, resource.StatusOrphanedWorktree, s.Status)
			break
		}
	}
	assert.True(t, found, "orphaned worktree should appear in state list")
}

// --- hashi switch (additional) ---

func TestIntegration_SwitchToDefaultBranch(t *testing.T) {
	skipIfNoTmux(t)
	t.Setenv("TMUX", "")

	repoRoot := testutil.GitRepo(t)
	session := "test-integration-swmain-" + filepath.Base(repoRoot)
	t.Cleanup(func() { tmuxKillSession(t, session) })

	t.Chdir(repoRoot)

	cp := resource.CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: session}
	svc, _ := newTestService(t, cp)

	_, err := svc.Switch(context.Background(), resource.SwitchParams{Branch: "main"})
	logNonConnectError(t, "Switch", err)

	// No .worktrees/main should be created — default branch uses repo root
	wtPath := filepath.Join(repoRoot, ".worktrees", "main")
	_, err = os.Stat(wtPath)
	assert.True(t, os.IsNotExist(err), "default branch should not create a worktree under .worktrees/")
}

func TestIntegration_SwitchWithPostNewHooks(t *testing.T) {
	skipIfNoTmux(t)
	t.Setenv("TMUX", "")

	repoRoot := testutil.GitRepoWithBranch(t, "hook-branch")
	session := "test-integration-swhook-" + filepath.Base(repoRoot)
	t.Cleanup(func() { tmuxKillSession(t, session) })

	t.Chdir(repoRoot)

	cp := resource.CommonParams{
		RepoRoot: repoRoot, WorktreeDir: ".worktrees",
		DefaultBranch: "main", SessionName: session,
		PostNewHooks: []string{"touch hook_ran.txt"},
	}
	svc, _ := newTestService(t, cp)

	// Actual initCmd behavior is tested in unit tests (TestBuildInitCmd).
	_, err := svc.Switch(context.Background(), resource.SwitchParams{Branch: "hook-branch"})
	logNonConnectError(t, "Switch", err)
}

// --- hashi new (additional) ---

func TestIntegration_NewDefaultBranch(t *testing.T) {
	skipIfNoTmux(t)
	t.Setenv("TMUX", "")

	repoRoot := testutil.GitRepo(t)
	session := "test-integration-newmain-" + filepath.Base(repoRoot)
	t.Cleanup(func() { tmuxKillSession(t, session) })

	t.Chdir(repoRoot)

	cp := resource.CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: session}
	svc, _ := newTestService(t, cp)

	_, err := svc.New(context.Background(), resource.NewParams{Branch: "main"})
	logNonConnectError(t, "New", err)

	// Should not create .worktrees/main
	wtPath := filepath.Join(repoRoot, ".worktrees", "main")
	_, err = os.Stat(wtPath)
	assert.True(t, os.IsNotExist(err), "default branch should not create a worktree under .worktrees/")
}

// --- hashi rename (additional) ---

func TestIntegration_RenameSlashBranch(t *testing.T) {
	repoRoot := testutil.GitRepo(t)

	// Create branch with slash and its worktree
	gitCmd(t, repoRoot, "branch", "feature/old")
	oldWtPath := filepath.Join(repoRoot, ".worktrees", "feature", "old")
	require.NoError(t, os.MkdirAll(filepath.Dir(oldWtPath), 0755))
	gitCmd(t, repoRoot, "worktree", "add", oldWtPath, "feature/old")

	// Add marker to verify physical move
	require.NoError(t, os.WriteFile(filepath.Join(oldWtPath, "marker.txt"), []byte("moved"), 0644))

	t.Chdir(repoRoot)

	cp := resource.CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "dummy"}
	svc, _ := newTestService(t, cp)

	result, err := svc.Rename(context.Background(), resource.RenameParams{Old: "feature/old", New: "feature/new"})
	require.NoError(t, err)
	assert.False(t, result.Created, "worktree should be moved, not recreated")

	// Old path should be gone
	_, err = os.Stat(oldWtPath)
	assert.True(t, os.IsNotExist(err))

	// New path should have marker
	newWtPath := filepath.Join(repoRoot, ".worktrees", "feature", "new")
	_, err = os.Stat(filepath.Join(newWtPath, "marker.txt"))
	assert.NoError(t, err, "marker should exist in moved worktree")

	// Old parent "old" dir should be cleaned up, but "feature" should remain
	_, err = os.Stat(filepath.Join(repoRoot, ".worktrees", "feature"))
	assert.NoError(t, err, "feature/ parent should still exist")
}

// --- hashi remove (additional) ---

func TestIntegration_RemoveBranchWithoutWorktree(t *testing.T) {
	repoRoot := testutil.GitRepoWithBranch(t, "no-wt-branch")

	t.Chdir(repoRoot)

	cp := resource.CommonParams{RepoRoot: repoRoot, WorktreeDir: ".worktrees", DefaultBranch: "main", SessionName: "dummy"}
	svc, g := newTestService(t, cp)

	check, err := svc.PrepareRemove(context.Background(), "no-wt-branch")
	require.NoError(t, err)
	assert.True(t, check.HasBranch)
	assert.False(t, check.HasWorktree, "branch without worktree")

	result, err := svc.ExecuteRemove(context.Background(), check)
	require.NoError(t, err)
	assert.True(t, result.BranchDeleted)
	assert.False(t, result.WorktreeRemoved, "no worktree to remove")

	exists, _ := g.BranchExists("no-wt-branch")
	assert.False(t, exists)
}

func TestIntegration_CollectStateMainAlwaysPresent(t *testing.T) {
	repoRoot := testutil.GitRepo(t)

	t.Chdir(repoRoot)

	cp := testCommonParams(repoRoot, "dummy")
	svc, _ := newTestService(t, cp)

	states, err := svc.CollectState(context.Background())
	require.NoError(t, err)

	require.Len(t, states, 1, "only main should be present")
	assert.Equal(t, "main", states[0].Branch)
	assert.Equal(t, resource.StatusOK, states[0].Status)
}
