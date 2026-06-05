package webhook

import (
	"fmt"
	"os"

	"gitee-cli/cmd"
	"gitee-cli/internal/handler"
	"gitee-cli/internal/utils"
	"github.com/spf13/cobra"
)

var viewFlags struct {
	repo string
	id   int64
}

var viewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a single webhook's details",
	Example: `  gitee webhook view 1234 --repo owner/repo
  gitee webhook view -R owner/repo 1234 --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(webhookViewAgentHelp)
			os.Exit(0)
		}
		return nil
	},
	RunE: runView,
}

func init() {
	f := viewCmd.Flags()
	f.StringVarP(&viewFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.Int64VarP(&viewFlags.id, "id", "i", 0, "Webhook id (alternative to positional)")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	id, err := utils.ParseInt64Arg(viewFlags.id, args, "webhook")
	if err != nil {
		return err
	}
	return handler.WebhookView(viewFlags.repo, id)
}

const webhookViewAgentHelp = `# gitee webhook view — Agent Help

## Purpose
Display full details for a single webhook: delivery URL, subscribed
events, creation time, and (if the last delivery completed) the result
and HTTP status code. Use this to audit a specific hook before editing
or to diagnose why deliveries are failing.

## Pre-conditions
- Authenticated
- Token has ` + "`" + `projects` + "`" + ` scope (read is sufficient)
- --repo resolves to an existing repo
- <id> (or --id) is the integer id of an existing webhook (use
  ` + "`" + `gitee webhook list` + "`" + ` to find it)

## Required arguments
  <id>     Webhook id (find via ` + "`" + `gitee webhook list` + "`" + `)

## Optional flags
  --repo / -R   owner/repo
  --id / -i     Alternative to positional <id>
  --output      "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET.

## Post-conditions (on success)
- Stdout contains the webhook's URL, events, created_at, and (if
  available) the last delivery result
- Repo state is unchanged

## Output (stdout, table mode)
  ─ Webhook Details (id=1234)
    URL:      https://ci.example.com/hook
    Title:    CI
    Events:   push,issues
    Created:  2026-01-15T08:23:11+08:00
    Last:     success (HTTP 200)

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    no read access
- 404 Not Found:    webhook id doesn't exist (was it deleted?)
- 422 Unprocessable: --id is not an integer

## Parsing hint (json output)
  gitee webhook view -R owner/repo 1234 --output json | jq '{id, url, events}'

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 req/hour.

## See also
- ` + "`" + `gitee webhook list` + "`" + `    — enumerate webhooks
- ` + "`" + `gitee webhook edit` + "`" + `    — change URL, events, or password
- ` + "`" + `gitee webhook test` + "`" + `    — fire a test ping
`
