package issue

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var IssueCmd = &cobra.Command{
	Use:   "issue <subcommand>",
	Short: "Manage issues",
	Long:  `Work with GitHub-compatible issues on Gitee repositories.`,
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
}

func init() {
	IssueCmd.AddCommand(createCmd)
	IssueCmd.AddCommand(listCmd)
	IssueCmd.AddCommand(viewCmd)
	IssueCmd.AddCommand(editCmd)
	IssueCmd.AddCommand(closeCmd)
	IssueCmd.AddCommand(reopenCmd)
	IssueCmd.AddCommand(commentCmd)
	// NOTE: labelCmd is registered by cmd/issue/label.go's init(), next to
	// the command it owns. Registering it here as well appended the same
	// *cobra.Command to IssueCmd twice, which made `gitee help reference`
	// (and cobra's own help listing) print the whole `issue label` subtree
	// twice. Keep exactly one registration — in label.go.
	IssueCmd.AddCommand(commentsCmd)
}
