package auth

import (
	"github.com/spf13/cobra"
)

var AuthCmd = &cobra.Command{
	Use:   "auth <subcommand>",
	Short: "Authenticate with Gitee",
}

func init() {
	AuthCmd.AddCommand(statusCmd)
	AuthCmd.AddCommand(tokenCmd)
	AuthCmd.AddCommand(logoutCmd)
	AuthCmd.AddCommand(loginCmd)
	AuthCmd.AddCommand(refreshCmd)
}
