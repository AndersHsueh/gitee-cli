package notification

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var listFlags struct {
	unreadOnly bool
	readOnly   bool
	limit      int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List notification threads",
	Example: `  gitee notification list
  gitee notification list --unread
  gitee notification list --unread --output json
  gitee notification list --limit 50`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(notificationListAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		filter := ""
		if listFlags.unreadOnly {
			filter = "true"
		} else if listFlags.readOnly {
			filter = "false"
		}
		dc, err := cmd.NewDomainClientFrom(rt)
		if err != nil {
			return err
		}
		return dc.NotificationList(filter, listFlags.limit)
	},
}

func init() {
	f := listCmd.Flags()
	f.BoolVar(&listFlags.unreadOnly, "unread", false, "Show only unread threads")
	f.BoolVar(&listFlags.readOnly, "read", false, "Show only already-read threads (mutually exclusive with --unread)")
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of threads to fetch")
}

const notificationListAgentHelp = `# gitee notification list — Agent Help

## Purpose
List the notification threads addressed to the authenticated user. Use
this to triage the inbox, find which threads need a response, or feed
the inbox into another tool (e.g. summarize unread mentions).

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- Token has ` + "`" + `notifications` + "`" + ` scope (default for OAuth tokens; personal
  access tokens need it explicitly)

## Optional flags
  --unread         Show only UNREAD threads (server-side filter)
  --read           Show only already-read threads (mutually exclusive with --unread)
  --limit / -L     Page size sent as per_page (default 30). The CLI does not
                   clamp it and does not request further pages. Gitee's
                   maximum per_page is 100.
  --output         ` + "`" + `table` + "`" + ` (default) or ` + "`" + `json` + "`" + `

If neither ` + "`" + `--unread` + "`" + ` nor ` + "`" + `--read` + "`" + ` is given, both are returned.

## Idempotency
YES. Read-only.

## Atomicity
One GET. Either that single page of threads is returned, or the call fails.
A thread count above --limit is not included.

## Side effects
None.

## Post-conditions (on success)
- Stdout contains a table of threads (ID, REPO, TYPE, SUBJECT, WHEN)
- An asterisk ` + "`" + `*` + "`" + ` in the leftmost column marks threads that are still
  unread; use ` + "`" + `notification read <id>` + "`" + ` to clear them
- Account state is unchanged

## Output (stdout, table mode)
  ID      REPO                    TYPE    SUBJECT                                       WHEN
  *1234   myorg/billing-svc       Issue   API: invoice webhooks time out after 30s     2026-07-04 09:12
  1233    myorg/billing-svc       PR      feat: add idempotency-key to /charges         2026-07-03 18:40
  *1232   andershsueh/gitee-cli   Issue   feat(notification): add notification cmd      2026-07-03 14:05

## Output (stdout, json mode)
  [
    {
      "id": "1234",
      "unread": true,
      "reason": "subscribed",
      "updated_at": "...",
      "subject": {"title": "...", "type": "Issue", "url": "..."},
      "repository": {"full_name": "myorg/billing-svc", ...}
    },
    ...
  ]

## Common errors
- 401 Unauthorized: token missing or expired → gitee auth login
- 403 Forbidden: token lacks ` + "`" + `notifications` + "`" + ` scope

## Race condition warning ⚠️
The inbox is shared, mutable server state: threads flip between read
and unread as other sessions (web UI, other agents) mark them, and
new threads arrive at any time. This output is a snapshot — by the
time you feed an id into ` + "`" + `notification read` + "`" + `, the thread may
already be read (harmless; re-marking is a server-side no-op) or,
rarely, gone (404 — skip it and continue the loop). Feed ids from
THIS output directly into ` + "`" + `notification read` + "`" + `; do not re-list
between iterations to "confirm", and expect the ` + "`" + `*` + "`" + ` markers to
shift between runs.

## OAuth scope required
notifications

## Rate limit
1000 req/hour authenticated.

## Example
  # What's new since I last checked?
  gitee notification list --unread --output json \\
    | jq '.[] | {id, repo: .repository.full_name, subject: .subject.title}'

  # Quick triage loop
  for id in $(gitee notification list --unread -o json | jq -r '.[].id'); do
      gitee notification read "$id" --force
  done

## See also
- ` + "`" + `gitee notification count` + "`" + ` — just the unread count (cheap to call)
- ` + "`" + `gitee notification read` + "`" + `  — mark one or all as read
- ` + "`" + `gitee issue list` + "`" + `         — follow up on Issue-type notifications
- ` + "`" + `gitee pr list` + "`" + `            — follow up on PR-type notifications
`
