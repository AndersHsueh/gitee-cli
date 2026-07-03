package repo

import (
	"github.com/spf13/cobra"
)

var RepoCmd = &cobra.Command{
	Use:   "repo <subcommand>",
	Short: "Manage repositories",
	Long:  `Work with Gitee repositories.`,
}

func init() {
	RepoCmd.AddCommand(listCmd)
	RepoCmd.AddCommand(viewCmd)
	RepoCmd.AddCommand(createCmd)
	RepoCmd.AddCommand(forkCmd)
	RepoCmd.AddCommand(editCmd)
	RepoCmd.AddCommand(deleteCmd)
	RepoCmd.AddCommand(cloneCmd)
	RepoCmd.AddCommand(browseCmd)
	RepoCmd.AddCommand(branchesCmd)
	RepoCmd.AddCommand(archiveCmd)
	RepoCmd.AddCommand(syncCmd)
	RepoCmd.AddCommand(renameCmd)
	RepoCmd.AddCommand(tagCmd)
}
