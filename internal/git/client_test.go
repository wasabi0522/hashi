package git

import (
	"fmt"
	"os"
	osexec "os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/exec"
)

// newExitCodeState returns an *os.ProcessState with the given exit code.
// It does this by running a subprocess that exits with that code.
// Requires "sh" in PATH (standard on all Unix-like systems).
func newExitCodeState(code int) *os.ProcessState {
	cmd := osexec.Command("sh", "-c", fmt.Sprintf("exit %d", code))
	_ = cmd.Run()
	return cmd.ProcessState
}

func mockExec() *exec.ExecutorMock {
	return &exec.ExecutorMock{}
}

func TestNewClient(t *testing.T) {
	e := mockExec()
	c := NewClient(e)
	assert.NotNil(t, c)
}

func TestClientGitCommonDir(t *testing.T) {
	e := mockExec()
	e.OutputFunc = func(name string, args ...string) (string, error) {
		assert.Equal(t, "git", name)
		assert.Contains(t, args, "--git-common-dir")
		return "/repo/.git", nil
	}
	c := NewClient(e)
	out, err := c.GitCommonDir()
	require.NoError(t, err)
	assert.Equal(t, "/repo/.git", out)
}

func TestClientGitCommonDir_Error(t *testing.T) {
	e := mockExec()
	e.OutputFunc = func(name string, args ...string) (string, error) {
		return "", fmt.Errorf("not a git repo")
	}
	c := NewClient(e)
	_, err := c.GitCommonDir()
	assert.Error(t, err)
}

func TestClientSymbolicRef(t *testing.T) {
	e := mockExec()
	e.OutputFunc = func(name string, args ...string) (string, error) {
		assert.Equal(t, "symbolic-ref", args[0])
		return "refs/remotes/origin/main", nil
	}
	c := NewClient(e)
	out, err := c.SymbolicRef("refs/remotes/origin/HEAD")
	require.NoError(t, err)
	assert.Equal(t, "refs/remotes/origin/main", out)
}

func TestClientRemoteGetURL(t *testing.T) {
	e := mockExec()
	e.OutputFunc = func(name string, args ...string) (string, error) {
		assert.Equal(t, []string{"remote", "get-url", "origin"}, args)
		return "git@github.com:org/repo.git", nil
	}
	c := NewClient(e)
	out, err := c.RemoteGetURL("origin")
	require.NoError(t, err)
	assert.Equal(t, "git@github.com:org/repo.git", out)
}

func TestClientListBranches(t *testing.T) {
	t.Run("multiple branches", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "main\nfeature\nfix/bug", nil
		}
		c := NewClient(e)
		branches, err := c.ListBranches()
		require.NoError(t, err)
		assert.Equal(t, []string{"main", "feature", "fix/bug"}, branches)
	})

	t.Run("empty", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "", nil
		}
		c := NewClient(e)
		branches, err := c.ListBranches()
		require.NoError(t, err)
		assert.Nil(t, branches)
	})

	t.Run("error", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "", fmt.Errorf("fail")
		}
		c := NewClient(e)
		_, err := c.ListBranches()
		assert.Error(t, err)
	})
}

func TestClientBranchExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "  main", nil
		}
		c := NewClient(e)
		exists, err := c.BranchExists("main")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "", nil
		}
		c := NewClient(e)
		exists, err := c.BranchExists("nope")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("error", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "", fmt.Errorf("fail")
		}
		c := NewClient(e)
		_, err := c.BranchExists("main")
		assert.Error(t, err)
	})
}

func TestClientRenameBranch(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"branch", "-m", "--", "old", "new"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.RenameBranch("old", "new"))
}

func TestClientDeleteBranch(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"branch", "-D", "--", "feat"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.DeleteBranch("feat"))
}

func TestClientIsMerged(t *testing.T) {
	t.Run("merged", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error { return nil }
		c := NewClient(e)
		merged, err := c.IsMerged("feat", "main")
		require.NoError(t, err)
		assert.True(t, merged)
	})

	t.Run("not merged (exit code 1)", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error {
			return &osexec.ExitError{ProcessState: newExitCodeState(1)}
		}
		c := NewClient(e)
		merged, err := c.IsMerged("feat", "main")
		require.NoError(t, err)
		assert.False(t, merged)
	})

	t.Run("git error propagated", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error {
			return fmt.Errorf("git connection error")
		}
		c := NewClient(e)
		_, err := c.IsMerged("feat", "main")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "git connection error")
	})

	t.Run("args include double dash", func(t *testing.T) {
		e := mockExec()
		e.RunFunc = func(name string, args ...string) error {
			assert.Equal(t, []string{"merge-base", "--is-ancestor", "--", "feat", "main"}, args)
			return nil
		}
		c := NewClient(e)
		_, _ = c.IsMerged("feat", "main")
	})
}

func TestClientHasUncommittedChanges(t *testing.T) {
	t.Run("dirty", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "M file.go", nil
		}
		c := NewClient(e)
		has, err := c.HasUncommittedChanges("/repo")
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("clean", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "", nil
		}
		c := NewClient(e)
		has, err := c.HasUncommittedChanges("/repo")
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("error", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "", fmt.Errorf("fail")
		}
		c := NewClient(e)
		_, err := c.HasUncommittedChanges("/repo")
		assert.Error(t, err)
	})
}

func TestClientListWorktrees(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "worktree /repo\nbranch refs/heads/main\n\nworktree /repo/.wt/feat\nbranch refs/heads/feat\n", nil
		}
		c := NewClient(e)
		wts, err := c.ListWorktrees()
		require.NoError(t, err)
		require.Len(t, wts, 2)
		assert.Equal(t, "main", wts[0].Branch)
		assert.Equal(t, "feat", wts[1].Branch)
	})

	t.Run("error", func(t *testing.T) {
		e := mockExec()
		e.OutputFunc = func(name string, args ...string) (string, error) {
			return "", fmt.Errorf("fail")
		}
		c := NewClient(e)
		_, err := c.ListWorktrees()
		assert.Error(t, err)
	})
}

func TestClientAddWorktree(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"worktree", "add", "--", "/path", "branch"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.AddWorktree("/path", "branch"))
}

func TestClientAddWorktreeNewBranch(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"worktree", "add", "-b", "feat", "--", "/path", "main"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.AddWorktreeNewBranch("/path", "feat", "main"))
}

func TestClientRemoveWorktree(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"worktree", "remove", "--force", "/path"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.RemoveWorktree("/path"))
}

func TestClientRepairWorktrees(t *testing.T) {
	e := mockExec()
	e.RunFunc = func(name string, args ...string) error {
		assert.Equal(t, []string{"worktree", "repair"}, args)
		return nil
	}
	c := NewClient(e)
	require.NoError(t, c.RepairWorktrees())
}

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Worktree
	}{
		{name: "empty", input: "", want: nil},
		{
			name:  "single main worktree",
			input: "worktree /Users/user/repo\nHEAD abc123\nbranch refs/heads/main",
			want:  []Worktree{{Path: "/Users/user/repo", Branch: "main", IsMain: true}},
		},
		{
			name: "main and feature worktree",
			input: "worktree /Users/user/repo\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /Users/user/repo/.worktrees/feature-login\nHEAD def456\nbranch refs/heads/feature-login",
			want: []Worktree{
				{Path: "/Users/user/repo", Branch: "main", IsMain: true},
				{Path: "/Users/user/repo/.worktrees/feature-login", Branch: "feature-login", IsMain: false},
			},
		},
		{
			name: "detached HEAD worktree",
			input: "worktree /Users/user/repo\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /Users/user/repo/.worktrees/detached\nHEAD def456\ndetached",
			want: []Worktree{
				{Path: "/Users/user/repo", Branch: "main", IsMain: true},
				{Path: "/Users/user/repo/.worktrees/detached", Branch: "", IsMain: false, Detached: true},
			},
		},
		{
			name: "slash in branch name",
			input: "worktree /Users/user/repo\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /Users/user/repo/.worktrees/feat/auth\nHEAD def456\nbranch refs/heads/feat/auth",
			want: []Worktree{
				{Path: "/Users/user/repo", Branch: "main", IsMain: true},
				{Path: "/Users/user/repo/.worktrees/feat/auth", Branch: "feat/auth", IsMain: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWorktreeList(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
