package user

import (
	"github.com/spf13/cobra"
)

// UserCmd is the top-level command for inspecting and managing the
// authenticated user's relationship with other Gitee accounts (viewing
// profiles, following / unfollowing, listing followers / following).
var UserCmd = &cobra.Command{
	Use:   "user <subcommand>",
	Short: "Inspect and manage user accounts (view, follow, followers)",
	Long: `Inspect Gitee user accounts and manage the authenticated user's
follow graph.

Use ` + "`" + `user view` + "`" + ` to read a profile (any user, no auth needed for
public fields), ` + "`" + `user follow` + "`" + ` / ` + "`" + `user unfollow` + "`" + ` to toggle
follow relationships, and ` + "`" + `user followers` + "`" + ` / ` + "`" + `user following` + "`" + ` to
enumerate the social graph of any user.

Run ` + "`" + `gitee user view` + "`" + ` (no <username>) to view the authenticated
user's own profile.`,
}

func init() {
	UserCmd.AddCommand(viewCmd)
	UserCmd.AddCommand(followCmd)
	UserCmd.AddCommand(unfollowCmd)
	UserCmd.AddCommand(followersCmd)
	UserCmd.AddCommand(followingCmd)
}
