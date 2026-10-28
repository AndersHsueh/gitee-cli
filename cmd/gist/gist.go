package gist

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var GistCmd = &cobra.Command{
	Use:   "gist <subcommand>",
	Short: "Manage gists",
	Long:  `Work with Gitee code snippets (gists).`,
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
}

func init() {
	GistCmd.AddCommand(listCmd)
	GistCmd.AddCommand(viewCmd)
	GistCmd.AddCommand(createCmd)
	GistCmd.AddCommand(editCmd)
	GistCmd.AddCommand(deleteCmd)
	GistCmd.AddCommand(starCmd)
	GistCmd.AddCommand(unstarCmd)
	GistCmd.AddCommand(cloneCmd)
	GistCmd.AddCommand(commentsCmd)
	GistCmd.AddCommand(commentCmd)
	GistCmd.AddCommand(commentDeleteCmd)
}
