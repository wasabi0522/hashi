package cmd

import (
	"github.com/spf13/cobra"

	"github.com/wasabi0522/hashi/internal/resource"
)

func (a *App) switchCmd(completeBranches completionFunc) *cobra.Command {
	return &cobra.Command{
		Use:               "switch <branch>",
		Aliases:           []string{"sw"},
		Short:             "Switch to an existing branch",
		Args:              cobra.MatchAll(cobra.ExactArgs(1), validateBranchArgs),
		RunE:              a.runSwitch,
		ValidArgsFunction: completeBranches,
	}
}

func (a *App) runSwitch(cmd *cobra.Command, args []string) error {
	return a.withService(func(svc *resource.Service) error {
		_, err := svc.Switch(cmd.Context(), resource.SwitchParams{Branch: args[0]})
		return err
	})
}
