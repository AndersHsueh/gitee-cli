package sshkey

import (
	"github.com/spf13/cobra"
)

// SSHKeyCmd is the top-level command for managing the current user's
// Gitee SSH public keys.
var SSHKeyCmd = &cobra.Command{
	Use:   "ssh-key <subcommand>",
	Short: "Manage SSH keys",
	Long: `Manage the SSH public keys associated with your Gitee account.

These keys let you push/pull Gitee repositories over SSH without entering
your password. Use this command to list, add, or remove keys without
opening the web UI.`,
}

func init() {
	SSHKeyCmd.AddCommand(listCmd)
	SSHKeyCmd.AddCommand(addCmd)
	SSHKeyCmd.AddCommand(deleteCmd)
}
