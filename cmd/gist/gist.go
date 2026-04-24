package gist

import (
	"github.com/spf13/cobra"
)

var GistCmd = &cobra.Command{
	Use:   "gist <subcommand>",
	Short: "Manage gists",
	Long:  `Work with Gitee code snippets (gists).`,
}

func init() {
	GistCmd.AddCommand(listCmd)
	GistCmd.AddCommand(viewCmd)
	GistCmd.AddCommand(createCmd)
	GistCmd.AddCommand(editCmd)
	GistCmd.AddCommand(deleteCmd)
	GistCmd.AddCommand(starCmd)
}
