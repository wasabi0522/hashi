package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wasabi0522/hashi/internal/resource"
)

// validateBranchArgs returns a cobra.PositionalArgs that validates all arguments as branch names.
// This duplicates the validation in resource.Service methods intentionally:
// cmd-layer validates user input early (via cobra Args), while the resource layer
// performs defensive checks against programmatic callers.
func validateBranchArgs(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		if err := resource.ValidateBranchName(arg); err != nil {
			return err
		}
	}
	return nil
}
