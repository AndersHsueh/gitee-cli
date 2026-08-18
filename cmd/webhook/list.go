package webhook

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var listFlags struct {
	repo  string
	limit int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhooks in a repository",
	Example: `  gitee webhook list --repo owner/repo
  gitee webhook list -R owner/repo --limit 50
  gitee webhook list -R owner/repo --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(webhookListAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		dc, err := cmd.NewDomainClient()
		if err != nil {
			return err
		}
		return dc.WebhookList(listFlags.repo, listFlags.limit)
	},
}

func init() {
	f := listCmd.Flags()
	f.StringVarP(&listFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of webhooks to return")
}

const webhookListAgentHelp = `# gitee webhook list — Agent Help

## Purpose
Enumerate every webhook configured on a Gitee repository. Use this to
discover webhook IDs for testing/editing/deleting, audit which external
endpoints receive repo events, or build an inventory for compliance.

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- Token has ` + "`" + `projects` + "`" + ` scope (read is sufficient)
- --repo resolves to an existing repo (use ` + "`" + `gitee repo view` + "`" + ` to verify)

## Required flags
  --repo / -R   owner/repo

## Optional flags
  --limit / -L   Maximum number of webhooks (default 30, max 100)
  --output       "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET.

## Post-conditions (on success)
- Stdout contains a table of webhooks (ID, URL, EVENTS, CREATED)
- Repo state is unchanged

## Output (stdout, table mode)
  ID    URL                          EVENTS                CREATED
  1234  https://ci.example.com/hook  push,issues           2026-01-15
  1235  https://hooks.example.com/x  push,tag_push,merge_request  2026-02-04

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 403 Forbidden:    no read access to the repo
- 404 Not Found:    repo doesn't exist or you can't see it

## Race condition warning ⚠️
The webhook set is shared, mutable server state. This output is a
point-in-time snapshot: another agent (or a repo admin in the web UI)
can create or delete a hook the moment after you list, so an id from
this output may 404 on a follow-up ` + "`" + `view` + "`" + ` / ` + "`" + `edit` + "`" + ` / ` + "`" + `delete` + "`" + `,
and a hook created seconds ago may be missing here. Act on ids
promptly, treat a 404 on the follow-up call as "the set changed —
re-list", and do not cache the id list across long-running
orchestrations.

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  gitee webhook list -R owner/repo --output json | jq '.[] | {id, url, events}'

## See also
- ` + "`" + `gitee webhook view` + "`" + `   — full details for a single webhook
- ` + "`" + `gitee webhook create` + "`" + ` — register a new webhook
- ` + "`" + `gitee webhook test` + "`" + `   — send a test ping to a webhook
`
