package pr

import (
	"github.com/spf13/cobra"
)

var PRCmd = &cobra.Command{
	Use:   "pr <subcommand>",
	Short: "Manage pull requests",
	Long:  `Work with pull requests on Gitee repositories.`,
}

func init() {
	PRCmd.AddCommand(listCmd)
	PRCmd.AddCommand(viewCmd)
	PRCmd.AddCommand(createCmd)
	PRCmd.AddCommand(mergeCmd)
	PRCmd.AddCommand(closeCmd)
	PRCmd.AddCommand(reopenCmd)
	PRCmd.AddCommand(reviewCmd)
	PRCmd.AddCommand(diffCmd)
}
