package milestone

import (
	"github.com/spf13/cobra"
)

// MilestoneCmd is the top-level command for managing Gitee repository milestones.
//
// Mirrors the structure of `gitee label` (one package per command group,
// a single top-level Cobra command, with one verb per file). All subcommands
// resolve their target repo via the standard --repo flag with the global
// --repo fallback, matching the convention used by `gitee label` and
// `gitee webhook`.
var MilestoneCmd = &cobra.Command{
	Use:   "milestone <subcommand>",
	Short: "Manage repository milestones",
	Long: `Manage milestones for a Gitee repository.

Milestones group issues and pull requests under a release target or a
delivery deadline. You can create, list, view, edit, and delete
milestones scoped to a single repository.`,
}

func init() {
	MilestoneCmd.AddCommand(listCmd)
	MilestoneCmd.AddCommand(viewCmd)
	MilestoneCmd.AddCommand(createCmd)
	MilestoneCmd.AddCommand(editCmd)
	MilestoneCmd.AddCommand(deleteCmd)
}