package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wasabi0522/hashi/internal/resource"
)

func (a *App) newCmd(completeBranches completionFunc) *cobra.Command {
	return &cobra.Command{
		Use:               "new <branch> [base]",
		Aliases:           []string{"n"},
		Short:             "Create a new branch with worktree and tmux window",
		Args:              cobra.MatchAll(cobra.RangeArgs(1, 2), validateBranchArgs),
		RunE:              a.runNew,
		ValidArgsFunction: completeBranches,
	}
}

func (a *App) runNew(cmd *cobra.Command, args []string) error {
	branch := args[0]
	var base string
	if len(args) >= 2 {
		base = args[1]
	}

	return a.withService(func(svc *resource.Service) error {
		_, err := svc.New(cmd.Context(), resource.NewParams{Branch: branch, Base: base})
		return err
	})
}
