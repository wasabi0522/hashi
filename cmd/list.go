package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/wasabi0522/hashi/internal/resource"
	"github.com/wasabi0522/hashi/internal/ui"
)

func (a *App) listCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List worktrees and tmux windows",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runList(cmd, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func (a *App) runList(cmd *cobra.Command, jsonOutput bool) error {
	d, err := a.resolveDeps(false)
	if err != nil {
		return err
	}

	states, err := d.service(a.serviceOpts()...).CollectState(cmd.Context())
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(cmd.OutOrStdout(), states)
	}
	printTable(cmd.OutOrStdout(), states)
	return nil
}

func printJSON(w io.Writer, states []resource.State) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(states)
}

var hashiTableStyle = table.Style{
	Name: "hashi",
	Box: table.BoxStyle{
		PaddingLeft:  "",
		PaddingRight: "  ",
	},
	Options: table.Options{
		DrawBorder:      false,
		SeparateHeader:  false,
		SeparateRows:    false,
		SeparateColumns: false,
	},
}

func printTable(w io.Writer, states []resource.State) {
	tw := table.NewWriter()
	tw.SetOutputMirror(w)

	tw.AppendHeader(table.Row{"", "BRANCH", "WORKTREE", "STATUS"})

	for _, s := range states {
		marker := " "
		if s.Active {
			marker = ui.Green("*")
		}

		worktreeStr := s.Worktree

		var statusMsg string
		if !s.Status.IsHealthy() {
			worktreeStr = ui.Yellow("(" + s.Status.Label() + ")")
			statusMsg = ui.Yellow(fmt.Sprintf("⚠ Run 'hashi %s %s'", s.Status.SuggestedCommand(), s.Branch))
		}

		tw.AppendRow(table.Row{marker, s.Branch, worktreeStr, statusMsg})
	}

	tw.SetStyle(hashiTableStyle)

	tw.Render()
}
