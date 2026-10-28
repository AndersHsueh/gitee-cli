package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee-cli/cmd"
	"gitee-cli/cmd/label"
)

// Legacy label command retained as an alias of `gitee label`.
// The actual behavior lives in cmd/label (top-level) and internal/handler.
// This file keeps the existing `gitee issue label` path working for users
// who haven't migrated to the top-level form yet.

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Manage issue labels (legacy path; prefer 'gitee label')",
	Long: `Manage issue labels for a repository.

This is the LEGACY path: 'gitee issue label'. The top-level
'gitee label' is the canonical form going forward. Both commands
call the same underlying handler and behave identically.`,
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
}

func init() {
	// Register the legacy labelCmd as a parent.
	IssueCmd.AddCommand(labelCmd)

	// Add alias subcommands that delegate to the top-level cmd/label
	// implementations. The flag definitions are copied so callers can use
	// the same --repo / --name / etc. flags on both paths.
	for _, src := range label.Subcommands() {
		alias := &cobra.Command{
			Use:     src.Use,
			Short:   src.Short,
			Long:    src.Long,
			Example: src.Example,
			// Inherit the canonical verb's positional-args contract
			// (label verbs take no positional args — all input is flags).
			Args: src.Args,
			RunE: src.RunE,
		}
		// Agent-help: the doc body is shared with the top-level label verb,
		// but the heading must name the path the user actually invoked
		// (`gitee issue label list`, not `gitee label list`). Retitle it.
		md := label.AgentHelpMarkdown(src.Name())
		alias.PreRunE = func(c *cobra.Command, args []string) error {
			if rt := cmd.RuntimeFromCmd(c); rt != nil && rt.AgentHelp {
				if md == "" {
					return fmt.Errorf("no agent help available for %q", c.CommandPath())
				}
				cmd.PrintAgentHelpForCommand(c, md)
			}
			return nil
		}
		alias.Flags().AddFlagSet(src.Flags())
		labelCmd.AddCommand(alias)
	}
}
