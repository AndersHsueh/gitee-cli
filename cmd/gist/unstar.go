package gist

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/spf13/cobra"
)

var unstarCmd = &cobra.Command{
	Use:     "unstar <id>",
	Short:   "Unstar a gist",
	Example: `  gitee gist unstar <gistID>`,
	Args:    cobra.ExactArgs(1),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gistUnstarAgentHelp)
		}
		return nil
	},
	RunE: runUnstar,
}

func runUnstar(c *cobra.Command, args []string) error {
	gistID := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.GistUnstar(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("unstarring gist: %w", err)
	}

	fmt.Printf("Gist %s unstarred\n", gistID)
	return nil
}

// gistUnstarAgentHelp is the agent-targeted help for `gitee gist unstar`.
// Maintainer note: keep this in sync with the actual command behavior.
const gistUnstarAgentHelp = `# gitee gist unstar — Agent Help

## Purpose
Remove a gist from the authenticated user's starred list. This is
the inverse of ` + "`" + `gitee gist star` + "`" + `. Use this when an agent needs to clean
up the starred list (e.g. pruning bookmarks the user no longer
references, or undoing an accidental star).

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The gist must currently be starred by the authenticated user
  (unstarring a non-starred gist is a no-op — see Idempotency)
- The gist itself does NOT need to still exist on Gitee; the star
  record can be cleared independently

## Required arguments
  <gistID>   Gist id (positional, required; enforced by ExactArgs(1))

## Optional flags
  none — unstar takes only the positional <gistID>

## Idempotency
IDEMPOTENT. Unstarring a gist that is not currently starred is a
no-op (the star record wasn't there in the first place). Re-running
the same call does not fail or duplicate anything.

## Atomicity
Atomic. The star record is either removed or the call fails. There
is no partial-success state.

## Side effects
- The gist is removed from the authenticated user's starred list
- The gist's stargazer count decrements by 1 on subsequent reads
- The gist ITSELF is not affected — only the star relationship is
  cleared (this is NOT a delete; use ` + "`" + `gitee gist delete` + "`" + ` for that)
- Webhooks subscribed to "Gist" star events may fire (unstar variant)
- REVERSIBLE: use ` + "`" + `gitee gist star <id>` + "`" + ` to undo

## Post-conditions (on success)
- The gist no longer appears in the user's starred list
- The gist itself still exists on Gitee (only the star was removed)
- stdout includes "Gist <id> unstarred"
- exit code 0

## Output (stdout, table mode)
  Gist abc123def unstarred

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     token lacks gist scope   →  check token permissions
- 404 Not Found:     star record doesn't exist (already unstarred, or
                     the gist was deleted)  →  safe to ignore; the
                     end state you want is already in place
- 5xx Server error:  transient                →  retry; idempotent so safe

## Next steps (orchestration hints)
After unstarring, an agent typically wants to:
1. Verify by re-listing the starred gists (no CLI command yet for
   "list starred gists" — fall back to the web UI or ` + "`" + `gitee gist list` + "`" + `
   which lists OWNED gists, not starred)
2. If the intent was actually to remove the gist entirely, run
   ` + "`" + `gitee gist delete <id>` + "`" + ` — but only if the user owns the gist

## OAuth scope required
projects (write) — or gists (write) on the older token model

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example (closing the loop)
  # 1. Unstar a gist (the natural inverse of ` + "`" + `gitee gist star` + "`" + `)
  gitee gist unstar abc123def

  # 2. To fully remove the gist (different command — this only
  #    removes the star relationship):
  gitee gist delete abc123def

## See also
- ` + "`" + `gitee gist star` + "`" + `     — the inverse (idempotent)
- ` + "`" + `gitee gist delete` + "`" + `  — remove the gist itself (irreversible;
                       only works for gists the token owns)
- ` + "`" + `gitee gist list` + "`" + `   — find owned ids
- ` + "`" + `gitee --agent-help` + "`" + ` — overview of all commands with agent help
`
