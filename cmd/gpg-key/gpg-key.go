package gpgkey

import (
	"github.com/spf13/cobra"
)

// GPGKeyCmd is the top-level command for managing the current user's
// Gitee GPG public keys.
var GPGKeyCmd = &cobra.Command{
	Use:   "gpg-key <subcommand>",
	Short: "Manage GPG keys",
	Long: `Manage the GPG public keys associated with your Gitee account.

These keys let you sign git commits and tags on Gitee with a verified
identity. Use this command to list, add, or remove keys without
opening the web UI.`,
}

func init() {
	GPGKeyCmd.AddCommand(listCmd)
	GPGKeyCmd.AddCommand(addCmd)
	GPGKeyCmd.AddCommand(deleteCmd)
}
