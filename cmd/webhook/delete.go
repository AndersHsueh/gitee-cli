package webhook

import (
	"fmt"

	"gitee-cli/cmd"
	"gitee-cli/internal/handler"
	"gitee-cli/internal/utils"

	"github.com/spf13/cobra"
)

var deleteFlags struct {
	repo  string
	id    int64
	force bool
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a webhook by id",
	Example: `  gitee webhook delete 1234 --repo owner/repo
  gitee webhook delete -R owner/repo 1234 --force`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(webhookDeleteAgentHelp)
		}
		return nil
	},
	RunE: runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.StringVarP(&deleteFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.Int64VarP(&deleteFlags.id, "id", "i", 0, "Webhook id (alternative to positional)")
	f.BoolVarP(&deleteFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

func runDelete(cobraCmd *cobra.Command, args []string) error {
	id, err := utils.ParseInt64Arg(deleteFlags.id, args, "webhook")
	if err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("webhook id is required (positional or --id)")
	}

	confirm := ""
	if deleteFlags.force {
		confirm = "y"
	}
	return handler.WebhookDelete(deleteFlags.repo, id, confirm)
}

const webhookDeleteAgentHelp = `# gitee webhook delete — Agent Help

## Purpose
Remove a webhook from a Gitee repository, stopping all event deliveries
to its URL. Use this when a CI target is decommissioned, a webhook URL
is compromised, or you are cleaning up after a migration.

## Pre-conditions
- Authenticated with write/admin rights on the target repo
- <id> corresponds to an existing webhook (returns 404 otherwise)

## Required arguments
  <id>     Webhook id (find via ` + "`" + `gitee webhook list` + "`" + `)

## Optional flags
  --repo / -R    owner/repo
  --id / -i      Alternative to positional <id>
  --force / -f   Skip the y/N confirmation prompt

## Idempotency
NOT idempotent. Re-running on a deleted webhook returns 404. There is
no "create or update" path on delete; use ` + "`" + `gitee webhook create` + "`" + ` to
register a replacement.

## Atomicity
Atomic. Either the webhook is gone, or no change.

## Side effects
- The delivery URL stops receiving new event POSTs IMMEDIATELY
  (in-flight requests complete, new ones do not fire)
- The webhook id is permanently freed (you cannot "undelete"; the same
  id may eventually be reused if you create a new webhook)
- No email or web notification is sent to the repo owner
- Webhooks subscribed to "repository" events may fire once (for the
  delete itself, depending on your Gitee plan)

## Post-conditions (on success)
- The webhook no longer appears in ` + "`" + `gitee webhook list` + "`" + `
- The delivery URL receives no further Gitee traffic
- The git history and repo contents are unchanged

## Output (stdout)
  ✓ Webhook id=1234 deleted

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    not a repo admin
- 404 Not Found:    webhook id doesn't exist (was it already deleted?)

## Race condition warning ⚠️
The webhook set is shared, mutable server state: another agent or a
repo admin can delete (or recreate) hooks at any time. In a
` + "`" + `list` + "`" + ` → pick an id → ` + "`" + `delete` + "`" + ` flow the hook may already be gone when
this call lands and you get a 404 — treat that 404 as "desired end
state reached", not necessarily as a failure of YOUR call. Ids can
eventually be recycled, so deleting right after a fresh
` + "`" + `gitee webhook list` + "`" + ` protects you from killing a newly-created hook
that reused an old id. With multiple agents cleaning up the same
repo, serialize deletes and act on ids from a list run that is
seconds old at most. Retrying a failed delete is safe: worst case is
a second 404.

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour.

## Example (closing the loop)
  # 1. Find the webhook id
  gitee webhook list -R owner/repo

  # 2. Delete by id (with confirmation)
  gitee webhook delete 1234 -R owner/repo

  # 3. Verify it's gone
  gitee webhook list -R owner/repo | grep 1234  # should print nothing

## See also
- ` + "`" + `gitee webhook list` + "`" + `   — find the id
- ` + "`" + `gitee webhook view` + "`" + `   — confirm details before deleting
- ` + "`" + `gitee webhook create` + "`" + ` — register a replacement
`
