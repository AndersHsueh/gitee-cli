package org

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var OrgCmd = &cobra.Command{
	Use:   "org <subcommand>",
	Short: "Manage organizations",
	Long:  `Work with Gitee organizations.`,
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
}

func init() {
	OrgCmd.AddCommand(listCmd)
	OrgCmd.AddCommand(viewCmd)
	OrgCmd.AddCommand(memberCmd)
	OrgCmd.AddCommand(repoCmd)
	OrgCmd.AddCommand(issueCmd)
}
