package context

import (
	"errors"
	"fmt"
	"net/url"
	osexec "os/exec"
	"path/filepath"
	"strings"

	"github.com/wasabi0522/hashi/internal/git"
)

// Context holds resolved repository information.
type Context struct {
	RepoRoot      string
	DefaultBranch string
	SessionName   string
}

// Resolver resolves repository context from git metadata.
type Resolver struct {
	git git.Client
}

// NewResolver creates a Resolver backed by the given git client.
func NewResolver(git git.Client) *Resolver {
	return &Resolver{git: git}
}

// Resolve resolves the full repository context.
func (r *Resolver) Resolve() (*Context, error) {
	repoRoot, err := r.resolveRepoRoot()
	if err != nil {
		return nil, err
	}

	defaultBranch, err := r.resolveDefaultBranch()
	if err != nil {
		return nil, err
	}

	sessionName := r.resolveSessionName(repoRoot)

	return &Context{
		RepoRoot:      repoRoot,
		DefaultBranch: defaultBranch,
		SessionName:   sessionName,
	}, nil
}

func (r *Resolver) resolveRepoRoot() (string, error) {
	gitDir, err := r.git.GitCommonDir()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return filepath.Dir(gitDir), nil
}

func (r *Resolver) resolveDefaultBranch() (string, error) {
	ref, err := r.git.SymbolicRef("refs/remotes/origin/HEAD")
	if err == nil {
		return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
	}

	// SymbolicRef exits with code 1 when the ref is missing — fall through.
	// Other exit codes (e.g. 128 for fatal git errors) indicate a real
	// problem, so propagate them instead of silently using a fallback.
	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() != 1 {
		return "", fmt.Errorf("resolving default branch: %w", err)
	}

	// Fallback: check main, then master
	for _, name := range []string{"main", "master"} {
		exists, err := r.git.BranchExists(name)
		if err != nil {
			return "", err
		}
		if exists {
			return name, nil
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}

func (r *Resolver) resolveSessionName(repoRoot string) string {
	rawURL, err := r.git.RemoteGetURL("origin")
	if err == nil {
		if orgRepo := parseOrgRepo(rawURL); orgRepo != "" {
			return sanitizeSessionName(orgRepo)
		}
	}

	// Fallback: directory name
	return sanitizeSessionName(filepath.Base(repoRoot))
}

// sanitizeSessionName makes a string safe for use as a tmux session name.
// tmux treats ':' and '.' specially; whitespace is replaced for usability.
func sanitizeSessionName(s string) string {
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' {
			return '-'
		}
		return r
	}, s)
	s = strings.TrimLeft(s, ".")
	if s == "" {
		return "hashi"
	}
	return s
}

// parseOrgRepo extracts "org/repo" from a git remote URL.
func parseOrgRepo(rawURL string) string {
	// SSH format: git@host:org/repo.git
	if idx := strings.Index(rawURL, "@"); idx >= 0 && !strings.Contains(rawURL, "://") {
		colonIdx := strings.Index(rawURL, ":")
		if colonIdx > idx {
			return cleanRepoPath(rawURL[colonIdx+1:])
		}
	}

	// URL format: https://host/org/repo.git or ssh://git@host/org/repo.git
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return cleanRepoPath(u.Path)
}

// cleanRepoPath normalizes a repository path by removing leading slashes and .git suffix.
func cleanRepoPath(path string) string {
	path = strings.TrimPrefix(path, "/")
	return strings.TrimSuffix(path, ".git")
}
