package release

import (
	"github.com/spf13/cobra"
)

// ReleaseCmd is the top-level command for managing Gitee releases.
var ReleaseCmd = &cobra.Command{
	Use:   "release <subcommand>",
	Short: "Manage releases",
	Long: `Work with Gitee releases (versioned tags with optional binaries).

A release is a first-class Gitee object that wraps a git tag with a
changelog body and downloadable assets. Use this command to ship
versions of your software without leaving the terminal.`,
}

func init() {
	ReleaseCmd.AddCommand(listCmd)
	ReleaseCmd.AddCommand(viewCmd)
	ReleaseCmd.AddCommand(createCmd)
	ReleaseCmd.AddCommand(editCmd)
	ReleaseCmd.AddCommand(deleteCmd)
	ReleaseCmd.AddCommand(downloadCmd)
}
