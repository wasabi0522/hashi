package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wasabi0522/hashi/internal/config"
	hashicontext "github.com/wasabi0522/hashi/internal/context"
	hashiexec "github.com/wasabi0522/hashi/internal/exec"
	"github.com/wasabi0522/hashi/internal/git"
	"github.com/wasabi0522/hashi/internal/resource"
	"github.com/wasabi0522/hashi/internal/tmux"
)

// App holds the dependency resolution functions and builds the CLI command tree.
type App struct {
	resolveDeps    func(requireTmux bool) (*deps, error)
	resolveGitDeps func() (*gitDeps, error)
	verbose        bool
}

// NewApp creates an App with default dependency resolvers.
func NewApp() *App {
	return &App{
		resolveDeps:    defaultResolveDeps,
		resolveGitDeps: defaultResolveGitDeps,
	}
}

type deps struct {
	exec hashiexec.Executor
	git  git.Client
	tmux tmux.Client
	ctx  *hashicontext.Context
	cfg  *config.Config
}

// resolveOpts controls how dependencies are resolved.
type resolveOpts struct {
	exec        hashiexec.Executor
	requireTmux bool
}

func defaultResolveDeps(requireTmux bool) (*deps, error) {
	return doResolveDeps(resolveOpts{exec: hashiexec.NewDefaultExecutor(), requireTmux: requireTmux})
}

func resolveDepsWithExec(e hashiexec.Executor) (*deps, error) {
	return doResolveDeps(resolveOpts{exec: e, requireTmux: true})
}

func buildGitContext(e hashiexec.Executor) (git.Client, *hashicontext.Context, error) {
	if err := e.LookPath("git"); err != nil {
		return nil, nil, fmt.Errorf("required command 'git' not found")
	}
	g := git.NewClient(e)
	ctx, err := hashicontext.NewResolver(g).Resolve()
	if err != nil {
		return nil, nil, err
	}
	return g, ctx, nil
}

func doResolveDeps(opts resolveOpts) (*deps, error) {
	g, ctx, err := buildGitContext(opts.exec)
	if err != nil {
		return nil, err
	}
	if opts.requireTmux {
		if err := opts.exec.LookPath("tmux"); err != nil {
			return nil, fmt.Errorf("required command 'tmux' not found")
		}
	}
	tm := tmux.NewPrefixedClient(tmux.NewClient(opts.exec), tmux.DefaultPrefix)
	cfg, err := config.Load(filepath.Join(ctx.RepoRoot, ".hashi.yaml"))
	if err != nil {
		return nil, err
	}
	return &deps{exec: opts.exec, git: g, tmux: tm, ctx: ctx, cfg: cfg}, nil
}

// withService resolves dependencies (requiring tmux) and calls fn with the constructed Service.
func (a *App) withService(fn func(svc *resource.Service) error) error {
	d, err := a.resolveDeps(true)
	if err != nil {
		return err
	}
	return fn(d.service(a.serviceOpts()...))
}

func (a *App) serviceOpts() []resource.Option {
	if a.verbose {
		return []resource.Option{resource.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, nil)))}
	}
	return nil
}

func (d *deps) service(opts ...resource.Option) *resource.Service {
	allOpts := []resource.Option{
		resource.WithCommonParams(resource.CommonParams{
			RepoRoot:      d.ctx.RepoRoot,
			WorktreeDir:   d.cfg.WorktreeDir,
			DefaultBranch: d.ctx.DefaultBranch,
			SessionName:   d.ctx.SessionName,
			CopyFiles:     d.cfg.Hooks.CopyFiles,
			PostNewHooks:  d.cfg.Hooks.PostNew,
		}),
	}
	allOpts = append(allOpts, opts...)
	return resource.NewService(d.exec, d.git, d.tmux, allOpts...)
}

type gitDeps struct {
	git git.Client
	ctx *hashicontext.Context
}

func defaultResolveGitDeps() (*gitDeps, error) {
	return resolveGitDepsWithExec(hashiexec.NewDefaultExecutor())
}

func resolveGitDepsWithExec(e hashiexec.Executor) (*gitDeps, error) {
	g, ctx, err := buildGitContext(e)
	if err != nil {
		return nil, err
	}
	return &gitDeps{git: g, ctx: ctx}, nil
}
