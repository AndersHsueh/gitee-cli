package org

import (
	"github.com/spf13/cobra"
)

var OrgCmd = &cobra.Command{
	Use:   "org <subcommand>",
	Short: "Manage organizations",
	Long:  `Work with Gitee organizations.`,
}

func init() {
	OrgCmd.AddCommand(listCmd)
	OrgCmd.AddCommand(viewCmd)
	OrgCmd.AddCommand(memberCmd)
	OrgCmd.AddCommand(repoCmd)
	OrgCmd.AddCommand(issueCmd)
}
