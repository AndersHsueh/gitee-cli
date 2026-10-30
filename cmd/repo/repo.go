package repo

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var RepoCmd = &cobra.Command{
	Use:   "repo <subcommand>",
	Short: "Manage repositories",
	Long:  `Work with Gitee repositories.`,
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
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
	RepoCmd.AddCommand(collaboratorCmd)
	RepoCmd.AddCommand(starCmd)
	RepoCmd.AddCommand(repoUnstarCmd)
	RepoCmd.AddCommand(watchCmd)
	RepoCmd.AddCommand(repoUnwatchCmd)
	RepoCmd.AddCommand(fileCmd)
	RepoCmd.AddCommand(commitsCmd)
	RepoCmd.AddCommand(compareCmd)
	RepoCmd.AddCommand(protectCmd)
	RepoCmd.AddCommand(repoUnprotectCmd)
}
