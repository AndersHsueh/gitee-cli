package auth

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var AuthCmd = &cobra.Command{
	Use:   "auth <subcommand>",
	Short: "Authenticate with Gitee",
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
}

func init() {
	AuthCmd.AddCommand(statusCmd)
	AuthCmd.AddCommand(tokenCmd)
	AuthCmd.AddCommand(logoutCmd)
	AuthCmd.AddCommand(loginCmd)
	AuthCmd.AddCommand(refreshCmd)
	AuthCmd.AddCommand(switchCmd)
}
