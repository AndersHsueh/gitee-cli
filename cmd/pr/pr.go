package pr

import (
	"github.com/spf13/cobra"
)

var PRCmd = &cobra.Command{
	Use:   "pr <subcommand>",
	Short: "Manage pull requests",
	Long:  `Work with pull requests on Gitee repositories.`,
}

// CoCmd is a top-level alias of `gitee pr checkout`.
// Registered separately so users can run `gitee co <number>` from anywhere
// (matches `gh co <number>`).
var CoCmd = &cobra.Command{
	Use:   "co <number>",
	Short: "Alias for 'gitee pr checkout'",
	Long:  `Alias for 'gitee pr checkout'. Fetches and checks out a PR's head ref.`,
	// Disable flag inheritance: we want a clean UX where `gitee co` behaves
	// exactly like `gitee pr checkout`. Flags are still inherited from root
	// (--repo, --token, etc.) because root defines them as PersistentFlags.
	RunE: func(c *cobra.Command, args []string) error {
		// Delegate to checkoutCmd's RunE. Reuse the same flag values.
		return checkoutCmd.RunE(c, args)
	},
	// Don't show up as a separate entry in `gitee --help` listings; alias
	// is registered manually in main.go.
	Hidden: true,
}

func init() {
	PRCmd.AddCommand(listCmd)
	PRCmd.AddCommand(viewCmd)
	PRCmd.AddCommand(createCmd)
	PRCmd.AddCommand(mergeCmd)
	PRCmd.AddCommand(closeCmd)
	PRCmd.AddCommand(reopenCmd)
	PRCmd.AddCommand(reviewCmd)
	PRCmd.AddCommand(diffCmd)
	PRCmd.AddCommand(commentsCmd)
	// checkoutCmd is registered in cmd/pr/checkout.go's init() (source of
	// truth) to avoid double-registration that would cause "command
	// already registered" warnings at startup.
	// P1c additions
	PRCmd.AddCommand(readyCmd)
	PRCmd.AddCommand(commentCmd)
	PRCmd.AddCommand(editCmd)
	PRCmd.AddCommand(checksCmd)
}
