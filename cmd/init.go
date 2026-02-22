package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed templates/hashi.yaml.tmpl
var configTemplate string

func (a *App) initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Generate .hashi.yaml template",
		Args:  cobra.NoArgs,
		RunE:  a.runInit,
	}
}

func (a *App) runInit(cmd *cobra.Command, args []string) error {
	d, err := a.resolveGitDeps()
	if err != nil {
		return err
	}

	path := filepath.Join(d.ctx.RepoRoot, ".hashi.yaml")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf(".hashi.yaml already exists")
		}
		return err
	}
	defer f.Close() //nolint:errcheck // best-effort close on a just-written file
	if _, err := f.WriteString(configTemplate); err != nil {
		return err
	}

	// best-effort: stdout write failure is non-actionable
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", path)
	return nil
}
