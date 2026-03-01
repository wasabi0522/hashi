package resource

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/tmux"
)

// Logger defines an interface for logging best-effort operation failures.
type Logger interface {
	Warn(msg string, args ...any)
}

// Option configures a Service.
type Option func(*Service)

// WithLogger sets the logger for best-effort operation warnings.
func WithLogger(l Logger) Option {
	return func(s *Service) { s.logger = l }
}

// WithCommonParams sets the common parameters for operations.
func WithCommonParams(cp CommonParams) Option {
	return func(s *Service) { s.cp = cp }
}

// WithShellCommands overrides the set of commands recognized as interactive shells.
func WithShellCommands(m map[string]struct{}) Option {
	return func(s *Service) { s.shellCommands = m }
}

// Service provides resource operations backed by git and tmux clients.
type Service struct {
	git           git.Client
	tmux          tmux.Client
	cp            CommonParams
	shellCommands map[string]struct{}
	logger        Logger
}

// nopLogger discards all log messages.
type nopLogger struct{}

func (nopLogger) Warn(string, ...any) {}

// NewService creates a Service with defaults for shell commands.
func NewService(g git.Client, tm tmux.Client, opts ...Option) *Service {
	s := &Service{
		git:           g,
		tmux:          tm,
		shellCommands: DefaultShellCommands,
		logger:        nopLogger{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// CommonParams holds fields shared by New, Switch, Rename, and Remove operations.
type CommonParams struct {
	RepoRoot      string
	WorktreeDir   string
	DefaultBranch string
	SessionName   string
	Shell         string
	CopyFiles     []string
	PostNewHooks  []string
}

// WorktreePath returns the filesystem path for the given branch's worktree.
func (p CommonParams) WorktreePath(branch string) string {
	return filepath.Join(p.RepoRoot, p.WorktreeDir, branch)
}

// Status represents the health status of a hashi-managed resource.
type Status int

const (
	// StatusOK indicates all resources (branch, worktree, window) are present.
	StatusOK Status = iota
	// StatusWorktreeMissing indicates the tmux window exists but the worktree is missing.
	StatusWorktreeMissing
	// StatusOrphanedWindow indicates the tmux window exists but the branch does not.
	StatusOrphanedWindow
	// StatusOrphanedWorktree indicates the worktree exists but the branch has been deleted.
	StatusOrphanedWorktree
)

// statusMeta holds all metadata for a single Status value.
type statusMeta struct {
	name    string // serialized name (e.g. "ok", "worktree_missing")
	label   string // human-readable label for unhealthy statuses
	suggest string // hashi subcommand to fix an unhealthy status
}

var statusTable = [...]statusMeta{
	StatusOK:               {name: "ok"},
	StatusWorktreeMissing:  {name: "worktree_missing", label: "worktree missing", suggest: "new"},
	StatusOrphanedWindow:   {name: "orphaned_window", label: "orphaned window", suggest: "remove"},
	StatusOrphanedWorktree: {name: "orphaned_worktree", label: "orphaned worktree", suggest: "remove"},
}

func (s Status) meta() statusMeta {
	if int(s) < len(statusTable) {
		return statusTable[s]
	}
	return statusMeta{name: "unknown"}
}

// String returns the string representation of the Status.
func (s Status) String() string { return s.meta().name }

// MarshalJSON returns the JSON encoding of the Status.
func (s Status) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "%q", s.String()), nil
}

// UnmarshalJSON parses a JSON string into a Status.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)
	for i, m := range statusTable {
		if m.name == str {
			*s = Status(i)
			return nil
		}
	}
	return fmt.Errorf("unknown status: %s", str)
}

// IsHealthy reports whether the status indicates all resources are present.
func (s Status) IsHealthy() bool { return s == StatusOK }

// Label returns a human-readable label for unhealthy statuses.
// Returns an empty string for StatusOK or unknown status values.
func (s Status) Label() string { return s.meta().label }

// SuggestedCommand returns the hashi subcommand to fix an unhealthy status.
// Returns an empty string for StatusOK or unknown status values.
func (s Status) SuggestedCommand() string { return s.meta().suggest }

// OperationType represents the kind of resource operation performed.
type OperationType int

const (
	OpNew OperationType = iota
	OpSwitch
	OpRename
)

// String returns the string representation of the OperationType.
func (o OperationType) String() string {
	switch o {
	case OpNew:
		return "new"
	case OpSwitch:
		return "switch"
	case OpRename:
		return "rename"
	default:
		return "unknown"
	}
}

// OperationResult holds the outcome of a New, Switch, or Rename operation.
// Currently used internally; will be surfaced in CLI output (e.g. --json flag).
type OperationResult struct {
	Operation    OperationType
	Branch       string
	WorktreePath string
	Created      bool // true if a new worktree was created
}

// State represents the combined state of a branch across git and tmux.
type State struct {
	Branch    string `json:"branch"`
	Worktree  string `json:"worktree,omitempty"`
	Window    bool   `json:"window"`
	Active    bool   `json:"active"`
	IsDefault bool   `json:"is_default"`
	Status    Status `json:"status"`
}
