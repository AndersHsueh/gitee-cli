package webhook

import (
	"gitee-cli/cmd"
	"gitee-cli/internal/handler"
	"gitee-cli/internal/utils"

	"github.com/spf13/cobra"
)

var testFlags struct {
	repo string
	id   int64
}

var testCmd = &cobra.Command{
	Use:   "test <id>",
	Short: "Send a test ping to a webhook (verify delivery URL accepts POSTs)",
	Example: `  gitee webhook test 1234 --repo owner/repo
  gitee webhook test -R owner/repo 1234`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(webhookTestAgentHelp)
		}
		return nil
	},
	RunE: runTest,
}

func init() {
	f := testCmd.Flags()
	f.StringVarP(&testFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.Int64VarP(&testFlags.id, "id", "i", 0, "Webhook id (alternative to positional)")
}

func runTest(cobraCmd *cobra.Command, args []string) error {
	id, err := utils.ParseInt64Arg(testFlags.id, args, "webhook")
	if err != nil {
		return err
	}
	return handler.WebhookTest(testFlags.repo, id)
}

const webhookTestAgentHelp = `# gitee webhook test — Agent Help

## Purpose
Fire a test ping at a webhook's delivery URL. Gitee makes a real HTTP
POST to the URL; the receiving endpoint should respond 2xx. Use this
to verify the URL is reachable and accepts Gitee's payload format,
WITHOUT waiting for a real repo event to occur.

## Pre-conditions
- Authenticated with read access (the call is read-only on the repo)
- <id> is an existing webhook id

## Required arguments
  <id>     Webhook id

## Optional flags
  --repo / -R   owner/repo
  --id / -i     Alternative to positional <id>

## Idempotency
NO. Each call dispatches a new outbound HTTP request. Gitee does not
de-duplicate successive tests. Avoid tight loops (the receiving
endpoint may rate-limit you).

## Atomicity
Atomic on the Gitee API side (one POST, one 204). The receiving
endpoint's processing is independent and may fail asynchronously —
check ` + "`" + `gitee webhook view` + "`" + ` for the result/result_code a few seconds
later.

## Side effects
- A new outbound HTTP request is sent to the delivery URL
- The receiving endpoint sees an ` + "`" + `X-Gitee-Event: ping` + "`" + ` (or similar test) header
- No state change in the repo; the webhook subscription is unchanged
- A 204 from Gitee means Gitee accepted the request — NOT that the
  receiver handled it correctly. Verify via ` + "`" + `gitee webhook view` + "`" + `.

## Post-conditions (on success)
- Stdout confirms the test was dispatched
- The webhook's ` + "`" + `result` + "`" + ` / ` + "`" + `result_code` + "`" + ` (visible in ` + "`" + `gitee webhook view` + "`" + `)
  is updated a few seconds later with the receiver's response

## Output (stdout)
  ✓ Test ping sent to webhook id=1234 (delivery result is not inline —
    inspect with ` + "`" + `gitee webhook view 1234` + "`" + `)

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    no read access
- 404 Not Found:    webhook id doesn't exist
- 422 Unprocessable: webhook is disabled (no event subscriptions) —
  patch it via ` + "`" + `gitee webhook edit` + "`" + `

## Race condition warning ⚠️
The ` + "`" + `result` + "`" + ` field on the webhook is updated asynchronously. If you
chain ` + "`" + `gitee webhook test` + "`" + ` immediately into ` + "`" + `gitee webhook view` + "`" + `,
you may still see the previous delivery's result. Sleep 2-3 seconds
before reading.

## Next steps (orchestration hints)
1. Test the hook: ` + "`" + `gitee webhook test 1234 -R owner/repo` + "`" + `
2. Wait briefly for async delivery: ` + "`" + `sleep 2` + "`" + `
3. Inspect the result: ` + "`" + `gitee webhook view 1234 -R owner/repo` + "`" + `
   (look for "Last: success (HTTP 200)" or a non-2xx code)

## OAuth scope required
projects (read) — even though the call dispatches an outbound request,
Gitee does not require write scope for the test ping itself.

## Rate limit
Gitee API v5: 1000 req/hour. Avoid looping ` + "`" + `test` + "`" + ` more than a few
times per minute; receivers may rate-limit you.

## Example (closing the loop)
  # 1. Fire the test ping
  gitee webhook test 1234 -R owner/repo

  # 2. Wait for the async delivery to land
  sleep 3

  # 3. Check what the receiver answered
  gitee webhook view 1234 -R owner/repo | grep Last

## See also
- ` + "`" + `gitee webhook view` + "`" + `    — inspect delivery results
- ` + "`" + `gitee webhook create` + "`" + `  — register a new hook
- ` + "`" + `gitee webhook edit` + "`" + `    — toggle events on a hook
`
