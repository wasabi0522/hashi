package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	hashiexec "github.com/wasabi0522/hashi/internal/exec"
)

var version = "dev"

// BuildRootCmd builds the complete CLI command tree.
func (a *App) BuildRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "hashi",
		Short: "Git worktree + tmux session manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("hashi version %s\n", version))
	rootCmd.PersistentFlags().BoolVarP(&a.verbose, "verbose", "v", false, "Enable verbose logging")

	defaultExec := hashiexec.NewDefaultExecutor()
	completeBranches := func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completeBranchesWithExec(defaultExec)
	}

	// Register subcommands
	rootCmd.AddCommand(a.newCmd(completeBranches))
	rootCmd.AddCommand(a.switchCmd(completeBranches))
	rootCmd.AddCommand(a.renameCmd(completeBranches))
	rootCmd.AddCommand(a.removeCmd(completeBranches))
	rootCmd.AddCommand(a.listCmd())
	rootCmd.AddCommand(a.initCmd())
	rootCmd.AddCommand(completionCmd(rootCmd))

	return rootCmd
}

// Execute creates an App and runs the CLI.
func Execute() {
	app := NewApp()
	cmd := app.BuildRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
