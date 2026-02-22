package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	hashiexec "github.com/wasabi0522/hashi/internal/exec"
	"github.com/wasabi0522/hashi/internal/git"
)

// completionFunc is the type for cobra shell completion functions.
type completionFunc = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

func completionCmd(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion <bash|zsh|fish>",
		Short:     "Generate shell completion scripts",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
}

// completeBranchesWithExec creates a completion function that lists git branch names.
func completeBranchesWithExec(e hashiexec.Executor) ([]string, cobra.ShellCompDirective) {
	if err := e.LookPath("git"); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	g := git.NewClient(e)
	branches, err := g.ListBranches()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return branches, cobra.ShellCompDirectiveNoFileComp
}
