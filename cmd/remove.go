package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wasabi0522/hashi/internal/resource"
	"github.com/wasabi0522/hashi/internal/ui"
)

func (a *App) removeCmd(completeBranches completionFunc) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "remove [-f] <branch...>",
		Aliases: []string{"rm"},
		Short:   "Remove branches with their worktrees and tmux windows",
		Args:    cobra.MatchAll(cobra.MinimumNArgs(1), validateBranchArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runRemove(cmd, args, force)
		},
		ValidArgsFunction: completeBranches,
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompts")
	return cmd
}

func (a *App) runRemove(cmd *cobra.Command, args []string, force bool) error {

	d, err := a.resolveDeps(true)
	if err != nil {
		return err
	}

	svc := d.service(a.serviceOpts()...)

	for _, branch := range args {
		check, err := svc.PrepareRemove(cmd.Context(), branch)
		if err != nil {
			return err
		}

		if !force {
			prompt := buildRemovePrompt(check)
			if !confirmPrompt(cmd, prompt) {
				continue
			}
		}

		if _, err := svc.ExecuteRemove(cmd.Context(), check); err != nil {
			return err
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", ui.Green(fmt.Sprintf("Removed '%s'", branch)))
	}

	return nil
}

func removeWarnings(check resource.RemoveCheck) []string {
	var w []string
	if check.HasUncommitted {
		w = append(w, "has uncommitted changes")
	}
	if check.IsUnmerged {
		w = append(w, "has unmerged commits")
	}
	return w
}

func buildRemovePrompt(check resource.RemoveCheck) string {
	if !check.HasBranch {
		return fmt.Sprintf("Branch '%s' not found. Clean up orphaned resources?", check.Branch)
	}

	warnings := removeWarnings(check)
	if len(warnings) == 0 {
		return fmt.Sprintf("Remove '%s'?", check.Branch)
	}
	return fmt.Sprintf("Remove '%s'? (%s)", check.Branch, strings.Join(warnings, ", "))
}

func confirmPrompt(cmd *cobra.Command, message string) bool {
	_, _ = fmt.Fprintf(cmd.OutOrStderr(), "%s [y/N] ", message)
	scanner := bufio.NewScanner(cmd.InOrStdin())
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}
