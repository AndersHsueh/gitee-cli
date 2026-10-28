// Package webhook implements the `gitee webhook` command group for managing
// repository webhooks. All subcommands require a target repository; the
// pattern is identical to `gitee label` (per-repo) — see internal/handler/label.go
// for the closest analog.
//
// Webhooks let you POST event payloads (push, PR, issue, ...) to an external
// URL whenever something happens in the repo. This is Gitee's primary CI/CD
// integration point.
package webhook

import (
	"gitee-cli/cmd"
	"github.com/spf13/cobra"
)

// WebhookCmd is the top-level command for managing repository webhooks.
var WebhookCmd = &cobra.Command{
	Use:   "webhook <subcommand>",
	Short: "Manage repository webhooks",
	Long: `Manage webhooks for a Gitee repository.

Webhooks let you POST event payloads (push, PR, issue, ...) to an external
URL whenever something happens in the repo. Use this command to list,
create, view, edit, delete, or test webhooks without opening the web UI.

All subcommands require a target repository via --repo (or the global
--repo flag). Write operations (create/edit/delete) require repo admin
rights on the target.`,
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
}

func init() {
	WebhookCmd.AddCommand(listCmd)
	WebhookCmd.AddCommand(createCmd)
	WebhookCmd.AddCommand(viewCmd)
	WebhookCmd.AddCommand(editCmd)
	WebhookCmd.AddCommand(deleteCmd)
	WebhookCmd.AddCommand(testCmd)
}
