package issue

import (
	"github.com/spf13/cobra"

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
			RunE:    src.RunE,
			// Inherit PreRunE from the source so --agent-help still works
			// on the legacy path.
			PreRunE: src.PreRunE,
		}
		alias.Flags().AddFlagSet(src.Flags())
		labelCmd.AddCommand(alias)
	}
}
