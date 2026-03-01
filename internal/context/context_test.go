package context

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wasabi0522/hashi/internal/git"
)

func newMock() *git.ClientMock {
	return &git.ClientMock{}
}

func TestResolve(t *testing.T) {
	t.Run("full resolution with remote URL", func(t *testing.T) {
		mock := newMock()
		mock.GitCommonDirFunc = func() (string, error) {
			return "/Users/user/repo/.git", nil
		}
		mock.SymbolicRefFunc = func(ref string) (string, error) {
			return "refs/remotes/origin/main", nil
		}
		mock.RemoteGetURLFunc = func(remote string) (string, error) {
			return "https://github.com/wasabi0522/hashi.git", nil
		}

		r := NewResolver(mock)
		ctx, err := r.Resolve()
		require.NoError(t, err)
		assert.Equal(t, "/Users/user/repo", ctx.RepoRoot)
		assert.Equal(t, "main", ctx.DefaultBranch)
		assert.Equal(t, "wasabi0522/hashi", ctx.SessionName)
	})

	t.Run("not a git repository", func(t *testing.T) {
		mock := newMock()
		mock.GitCommonDirFunc = func() (string, error) {
			return "", errors.New("fatal: not a git repository")
		}

		r := NewResolver(mock)
		_, err := r.Resolve()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})
}

func TestResolveDefaultBranch(t *testing.T) {
	t.Run("from symbolic ref", func(t *testing.T) {
		mock := newMock()
		mock.SymbolicRefFunc = func(ref string) (string, error) {
			return "refs/remotes/origin/develop", nil
		}

		r := &Resolver{git: mock}
		branch, err := r.resolveDefaultBranch()
		require.NoError(t, err)
		assert.Equal(t, "develop", branch)
	})

	t.Run("fallback to main", func(t *testing.T) {
		mock := newMock()
		mock.SymbolicRefFunc = func(ref string) (string, error) {
			return "", errors.New("not set")
		}
		mock.BranchExistsFunc = func(name string) (bool, error) {
			return name == "main", nil
		}

		r := &Resolver{git: mock}
		branch, err := r.resolveDefaultBranch()
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("fallback to master", func(t *testing.T) {
		mock := newMock()
		mock.SymbolicRefFunc = func(ref string) (string, error) {
			return "", errors.New("not set")
		}
		mock.BranchExistsFunc = func(name string) (bool, error) {
			return name == "master", nil
		}

		r := &Resolver{git: mock}
		branch, err := r.resolveDefaultBranch()
		require.NoError(t, err)
		assert.Equal(t, "master", branch)
	})

	t.Run("no default branch found", func(t *testing.T) {
		mock := newMock()
		mock.SymbolicRefFunc = func(ref string) (string, error) {
			return "", errors.New("not set")
		}
		mock.BranchExistsFunc = func(name string) (bool, error) {
			return false, nil
		}

		r := &Resolver{git: mock}
		_, err := r.resolveDefaultBranch()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not determine default branch")
	})
}

func TestResolveSessionName(t *testing.T) {
	t.Run("from SSH remote URL", func(t *testing.T) {
		mock := newMock()
		mock.RemoteGetURLFunc = func(remote string) (string, error) {
			return "git@github.com:wasabi0522/hashi.git", nil
		}

		r := &Resolver{git: mock}
		name := r.resolveSessionName("/Users/user/repo")
		assert.Equal(t, "wasabi0522/hashi", name)
	})

	t.Run("fallback to directory name", func(t *testing.T) {
		mock := newMock()
		mock.RemoteGetURLFunc = func(remote string) (string, error) {
			return "", errors.New("no remote")
		}

		r := &Resolver{git: mock}
		name := r.resolveSessionName("/Users/user/my-project")
		assert.Equal(t, "my-project", name)
	})
}

func TestSanitizeSessionName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "my-project", "my-project"},
		{"with colon", "host:path", "host-path"},
		{"with space", "my project", "my-project"},
		{"with tab", "my\tproject", "my-project"},
		{"leading dots", "..hidden", "hidden"},
		{"only dots", "...", "hashi"},
		{"empty", "", "hashi"},
		{"org/repo", "wasabi0522/hashi", "wasabi0522/hashi"},
		{"unicode CJK", "日本語プロジェクト", "日本語プロジェクト"},
		{"emoji", "🚀rocket", "🚀rocket"},
		{"multiple leading dots", "...config", "config"},
		{"single dot", ".", "hashi"},
		{"control char NUL", "my\x00project", "my-project"},
		{"control char BEL", "my\x07project", "my-project"},
		{"control char DEL", "my\x7fproject", "my-project"},
		{"newline", "my\nproject", "my-project"},
		{"long name", strings.Repeat("a", 200), strings.Repeat("a", 200)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSessionName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveDefaultBranchError(t *testing.T) {
	t.Run("BranchExists error", func(t *testing.T) {
		mock := newMock()
		mock.SymbolicRefFunc = func(ref string) (string, error) {
			return "", errors.New("not set")
		}
		mock.BranchExistsFunc = func(name string) (bool, error) {
			return false, errors.New("git error")
		}

		r := &Resolver{git: mock}
		_, err := r.resolveDefaultBranch()
		require.Error(t, err)
	})
}

func TestResolveWithDefaultBranchError(t *testing.T) {
	mock := newMock()
	mock.GitCommonDirFunc = func() (string, error) {
		return "/repo/.git", nil
	}
	mock.SymbolicRefFunc = func(ref string) (string, error) {
		return "", errors.New("not set")
	}
	mock.BranchExistsFunc = func(name string) (bool, error) {
		return false, nil
	}

	r := NewResolver(mock)
	_, err := r.Resolve()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not determine default branch")
}

func TestParseOrgRepo(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"HTTPS with .git", "https://github.com/wasabi0522/hashi.git", "wasabi0522/hashi"},
		{"HTTPS without .git", "https://github.com/wasabi0522/hashi", "wasabi0522/hashi"},
		{"SSH", "git@github.com:wasabi0522/hashi.git", "wasabi0522/hashi"},
		{"SSH without .git", "git@github.com:wasabi0522/hashi", "wasabi0522/hashi"},
		{"SSH protocol", "ssh://git@github.com/wasabi0522/hashi.git", "wasabi0522/hashi"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOrgRepo(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}
