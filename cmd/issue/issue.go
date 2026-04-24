package issue

import (
	"github.com/spf13/cobra"
)

var IssueCmd = &cobra.Command{
	Use:   "issue <subcommand>",
	Short: "Manage issues",
	Long:  `Work with GitHub-compatible issues on Gitee repositories.`,
}

func init() {
	IssueCmd.AddCommand(createCmd)
	IssueCmd.AddCommand(listCmd)
	IssueCmd.AddCommand(viewCmd)
	IssueCmd.AddCommand(editCmd)
	IssueCmd.AddCommand(closeCmd)
	IssueCmd.AddCommand(reopenCmd)
	IssueCmd.AddCommand(commentCmd)
	IssueCmd.AddCommand(labelCmd)
}
