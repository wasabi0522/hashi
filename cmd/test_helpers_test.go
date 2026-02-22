package cmd

import (
	"bytes"
	"testing"
)

// appWithDeps creates an App that resolves to the given deps.
func appWithDeps(d *deps) *App {
	return &App{
		resolveDeps: func(requireTmux bool) (*deps, error) { return d, nil },
		resolveGitDeps: func() (*gitDeps, error) {
			return &gitDeps{git: d.git, ctx: d.ctx}, nil
		},
	}
}

// appWithDepsError creates an App whose resolveDeps returns an error.
func appWithDepsError(err error) *App {
	return &App{
		resolveDeps:    func(requireTmux bool) (*deps, error) { return nil, err },
		resolveGitDeps: func() (*gitDeps, error) { return nil, err },
	}
}

// executeCommand runs the CLI command tree with the given args and returns the output.
func executeCommand(t *testing.T, app *App, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := app.BuildRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}
