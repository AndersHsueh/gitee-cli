package gist

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete a gist",
	Example: `  gitee gist delete abc123`,
	Args:    cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gistDeleteAgentHelp)
		}
		return nil
	},
	RunE: runDelete,
}

func runDelete(cobraCmd *cobra.Command, args []string) error {
	gistID := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.GistDelete(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("deleting gist: %w", err)
	}

	fmt.Printf("Gist %s deleted\n", gistID)
	return nil
}

// gistDeleteAgentHelp is the agent-targeted help for `gitee gist delete`.
// Maintainer note: keep this in sync with the actual command behavior.
const gistDeleteAgentHelp = `# gitee gist delete — Agent Help

## Purpose
Permanently delete a gist. Use this when an agent needs to clean up
a snippet the user no longer wants (e.g. removing a repro gist
after the corresponding issue is closed, or pruning old WIPs).

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The gist must exist (verify with: gitee gist list or gitee gist view)
- The token's owner must have write access to the gist (i.e. own it)
- The agent (or the user calling it) must have CONFIRMED the delete —
  this is an IRREVERSIBLE destructive command

## Required arguments
  <id>   Gist id (positional, required)

## Optional flags
  none — delete takes only the positional <id>

## Idempotency
IDEMPOTENT in the "second call is a no-op" sense: deleting an
already-deleted gist returns 404, but the end state is the same
(nothing exists). The first call is the destructive one.

## Atomicity
Atomic. The gist either disappears entirely or the call fails.
There is no partial-success state. The Gitee API does NOT expose
a trash / recycle bin — once the response is 2xx, the data is
gone from the API.

## Side effects (IRREVERSIBLE)
- The gist is REMOVED from the authenticated user's account
- All of the gist's files are deleted (no version history retained)
- Any stargazers' bookmark of this gist becomes a dangling reference
- Webhooks subscribed to "Gist" deletion events fire
- Public gists that were indexed by external search engines may
  still appear in their caches for some time, but the gist URL
  itself returns 404 on Gitee

WARNING: There is NO UNDO. The Gitee API has no restore endpoint,
and the CLI does not maintain a local snapshot. If you might want
the content back, run ` + "`" + `gitee gist clone <id>` + "`" + ` FIRST.

## Post-conditions (on success)
- The gist no longer exists on Gitee
- ` + "`" + `gitee gist view <id>` + "`" + ` returns 404
- ` + "`" + `gitee gist list` + "`" + ` no longer shows it
- stdout includes "Gist <id> deleted"
- exit code 0

## Output (stdout, table mode)
  Gist abc123def deleted

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     token is not the gist owner  →  cannot delete
- 404 Not Found:     gist id is wrong or was already deleted
                     →  safe to ignore if you intended a no-op
- 5xx Server error:  transient                →  retry; but be aware
                     that a retry on a transient failure is safe
                     (idempotent) — the second call just 404s

## Race condition warning ⚠️
Delete is IRREVERSIBLE and the gist is shared mutable state:
- An edit or comment landing between your last ` + "`" + `gitee gist view` + "`" + `
  and this delete is destroyed with the gist. If the content might
  matter, run ` + "`" + `gitee gist clone <id>` + "`" + ` immediately before deleting.
- Concurrent deletes (two agents, or agent + web UI): the first
  succeeds, the rest get 404 — which is the desired end state, so
  retrying a delete is safe.
- Do not delete based on a stale ` + "`" + `gitee gist list` + "`" + `: always act on
  the exact id, and re-run ` + "`" + `gitee gist view <id>` + "`" + ` right before if
  there is any doubt the gist is still the one you mean.

## Next steps (orchestration hints)
After a successful delete, an agent typically wants to:
1. Verify with ` + "`" + `gitee gist list` + "`" + ` (the deleted id should be gone)
2. If a PR or issue referenced this gist, update or close that
   reference — the URL will now 404
3. If this was cleaning up after a fix: ` + "`" + `gitee issue close <N>` + "`" + `
   or ` + "`" + `gitee pr close <N>` + "`" + ` to close the original tracker

## OAuth scope required
projects (write) — or gists (write) on the older token model

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example (agent-friendly, with safety)
  # 1. (Optional but recommended) snapshot first
  gitee gist clone abc123def /tmp/abc123def-backup

  # 2. Confirm with the user before proceeding — this is destructive

  # 3. Delete
  gitee gist delete abc123def

  # 4. Verify
  gitee gist list       # abc123def should be absent

## See also
- ` + "`" + `gitee gist list` + "`" + `    — find ids; confirm the delete took
- ` + "`" + `gitee gist view` + "`" + `    — verify the gist is gone (404)
- ` + "`" + `gitee gist clone` + "`" + `   — snapshot BEFORE you delete
                       (irreversible: no undo, no restore endpoint)
- ` + "`" + `gitee --agent-help` + "`" + ` — overview of all commands with agent help
`
