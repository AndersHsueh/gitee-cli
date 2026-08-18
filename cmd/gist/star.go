package gist

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/spf13/cobra"
)

var starCmd = &cobra.Command{
	Use:     "star <id>",
	Short:   "Star a gist",
	Example: `  gitee gist star abc123`,
	Args:    cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gistStarAgentHelp)
		}
		return nil
	},
	RunE: runStar,
}

func runStar(cobraCmd *cobra.Command, args []string) error {
	gistID := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.GistStar(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Put(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("starring gist: %w", err)
	}

	fmt.Printf("Gist %s starred\n", gistID)
	return nil
}

// gistStarAgentHelp is the agent-targeted help for `gitee gist star`.
// Maintainer note: keep this in sync with the actual command behavior.
const gistStarAgentHelp = `# gitee gist star — Agent Help

## Purpose
Star a gist on behalf of the authenticated user. Adds the gist to
the user's starred-gists list (bookmark for later). Use this when
an agent needs to mark a snippet as worth remembering.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The gist must exist (verify with: gitee gist list or gitee gist view)
- The gist must be world-readable (public) to star — private gists
  can only be starred by their owner

## Required arguments
  <id>   Gist id (positional, required)

## Optional flags
  none — star takes only the positional <id>

## Idempotency
IDEMPOTENT. Starring an already-starred gist is a no-op server-side
(the star record is already there). Re-running the same call does
not create duplicate stars.

## Atomicity
Atomic. The star either gets recorded or the call fails. There is
no partial-success state.

## Side effects
- The gist is added to the authenticated user's starred list
- Webhooks subscribed to "Gist" star events may fire
- The gist's owner may receive a notification (Gitee-side behavior;
  not under the agent's control)
- The gist's stargazer count increments by 1 on subsequent reads
- REVERSIBLE: use ` + "`" + `gitee gist unstar <id>` + "`" + ` to undo

## Post-conditions (on success)
- The gist appears in the user's starred list
- stdout includes "Gist <id> starred"
- exit code 0

## Output (stdout, table mode)
  Gist abc123def starred

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     token lacks gist scope   →  check token permissions
- 404 Not Found:     gist id is wrong or was deleted  →  gitee gist list
- 5xx Server error:  transient                →  retry; idempotent so safe

## Race condition warning ⚠️
The star relationship is shared mutable state, but this command is
idempotent, which defuses most races:
- If the gist is deleted between your ` + "`" + `gitee gist view` + "`" + ` and this
  call, star returns 404 — nothing to retry; the target is gone.
- A concurrent ` + "`" + `gitee gist unstar` + "`" + ` (another session or agent) can
  flip the state right after your success. If the star must stick,
  verify afterwards rather than assuming.
- Retrying after a timeout/5xx is SAFE: starring an already-starred
  gist is a server-side no-op (see Idempotency).

## Next steps (orchestration hints)
After starring, an agent typically wants to:
1. ` + "`" + `gitee gist view <id>` + "`" + ` — confirm content (if not already read)
2. ` + "`" + `gitee gist clone <id>` + "`" + ` — bring it local if needed
3. ` + "`" + `gitee gist unstar <id>` + "`" + ` — to remove from the starred list
   later (this is the natural inverse — and it is idempotent too)

## OAuth scope required
projects (write) — or gists (write) on the older token model

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example (closing the loop)
  # 1. Star a gist you want to remember
  gitee gist star abc123def

  # 2. Pull a local copy while you're at it
  gitee gist clone abc123def

  # 3. Later, if you don't need the bookmark:
  gitee gist unstar abc123def

## See also
- ` + "`" + `gitee gist unstar` + "`" + ` — the inverse (idempotent)
- ` + "`" + `gitee gist view` + "`" + `   — verify the gist before starring
- ` + "`" + `gitee gist list` + "`" + `   — find ids
- ` + "`" + `gitee --agent-help` + "`" + ` — overview of all commands with agent help
`
