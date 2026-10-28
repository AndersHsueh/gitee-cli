package notification

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

// NotificationCmd is the top-level command for the Gitee notification
// center. The API is user-scoped (no {owner}/{repo} in the path) so
// this group has no --repo flag — see cmd/ssh-key for the closest
// analog (also user-scoped).
var NotificationCmd = &cobra.Command{
	Use:   "notification <subcommand>",
	Short: "Manage your Gitee notification inbox",
	Long: `Work with Gitee's notification center from the terminal.

The notification center aggregates mentions, review requests, subscribed
thread updates, and other activity addressed to the authenticated user.
Use these verbs to triage the inbox without opening the web UI.

  gitee notification list    — list threads (optionally only unread)
  gitee notification count    — show the unread count
  gitee notification read     — mark one thread or all as read

This is a top-level entry point (like ` + "`" + `gitee ssh-key` + "`" + `). The verbs are
deliberately minimal: mark-as-unread is supported via ` + "`" + `--unread` + "`" + ` on read.`,
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
}

func init() {
	NotificationCmd.AddCommand(listCmd)
	NotificationCmd.AddCommand(countCmd)
	NotificationCmd.AddCommand(readCmd)
}
