package notification

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var countCmd = &cobra.Command{
	Use:   "count",
	Args:  cobra.NoArgs,
	Short: "Show the number of unread notifications",
	Example: `  gitee notification count
  gitee notification count --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(notificationCountAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		dc, err := cmd.NewDomainClientFrom(rt)
		if err != nil {
			return err
		}
		return dc.NotificationCount()
	},
}

const notificationCountAgentHelp = `# gitee notification count — Agent Help

## Purpose
Print just the number of unread notification threads for the
authenticated user. Cheaper than ` + "`" + `notification list` + "`" + ` and intended
for polling: shell prompts, status bars, CI gates that block on
"is the inbox clean?".

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- Token has ` + "`" + `notifications` + "`" + ` scope

## Idempotency
YES. Read-only.

## Atomicity
Atomic. Single API call.

## Side effects
None.

## Post-conditions (on success)
- Stdout prints the count (table mode) or a JSON object (json mode)
- Account state is unchanged

## Output (stdout, table mode)
  Unread notifications: 7

## Output (stdout, json mode)
  {"count": 7}

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden: token lacks ` + "`" + `notifications` + "`" + ` scope

## Race condition warning ⚠️
The unread counter is shared, mutable server state: it changes when
new notifications arrive AND when any other session (web UI, another
agent) marks threads read. The number you print is stale the moment
it appears — a count of 0 does not guarantee the inbox stays empty,
and a non-zero count may already have been cleared elsewhere. For
gates like "block until inbox is clean", re-poll instead of trusting
one sample; if you need to know WHICH threads are unread, follow up
with ` + "`" + `gitee notification list --unread` + "`" + ` rather than reasoning from
the count alone.

## OAuth scope required
notifications

## Rate limit
1000 req/hour authenticated. The endpoint is intentionally lightweight
so it is safe to poll on a short interval (e.g. every minute in a
shell status line).

## Example
  # Shell prompt: show the unread count in $PS1
  export PS1='$(gitee notification count --output json | jq -r .count) > '

  # Wait until inbox is clean (useful for end-of-day scripts)
  while [ "$(gitee notification count --output json | jq -r .count)" -ne 0 ]; do
      sleep 60
  done

## See also
- ` + "`" + `gitee notification list` + "`" + ` — see WHICH threads are unread
- ` + "`" + `gitee notification read` + "`" + `  — clear them
`
