package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wasabi0522/hashi/internal/resource"
)

func (a *App) renameCmd(completeBranches completionFunc) *cobra.Command {
	return &cobra.Command{
		Use:               "rename <old> <new>",
		Aliases:           []string{"mv"},
		Short:             "Rename a branch with its worktree and tmux window",
		Args:              cobra.MatchAll(cobra.ExactArgs(2), validateBranchArgs),
		RunE:              a.runRename,
		ValidArgsFunction: completeBranches,
	}
}

func (a *App) runRename(cmd *cobra.Command, args []string) error {
	return a.withService(func(svc *resource.Service) error {
		_, err := svc.Rename(cmd.Context(), resource.RenameParams{Old: args[0], New: args[1]})
		return err
	})
}
