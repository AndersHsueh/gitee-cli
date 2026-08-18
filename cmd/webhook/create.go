package webhook

import (
	"gitee-cli/cmd"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var createFlags struct {
	repo           string
	url            string
	title          string
	password       string
	encryptionType int
	pushEvents     bool
	tagPushEvents  bool
	issuesEvents   bool
	noteEvents     bool
	mergeReqEvents bool
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new webhook for a repository",
	Example: `  gitee webhook create --repo owner/repo --url https://example.com/hook
  gitee webhook create -R owner/repo -u https://ci.example.com/hook \
      --title "CI" --issues-events --merge-request-events
  gitee webhook create -R owner/repo -u https://x.example.com/hook \
      --password s3cret --encryption-type 0`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(webhookCreateAgentHelp)
		}
		return nil
	},
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVarP(&createFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&createFlags.url, "url", "u", "", "Delivery URL to POST events to (required)")
	f.StringVarP(&createFlags.title, "title", "t", "", "Webhook title (max 191 chars)")
	f.StringVar(&createFlags.password, "password", "", "Signing secret sent with the request URL to prevent spoofing")
	f.IntVar(&createFlags.encryptionType, "encryption-type", 0, "Encryption type for --password: 0=password (HMAC-like), 1=signing key (reserved)")
	f.BoolVar(&createFlags.pushEvents, "push-events", true, "Subscribe to push events (default true; pass --push-events=false to opt out)")
	f.BoolVar(&createFlags.tagPushEvents, "tag-push-events", false, "Subscribe to tag push events")
	f.BoolVar(&createFlags.issuesEvents, "issues-events", false, "Subscribe to issue open/close events")
	f.BoolVar(&createFlags.noteEvents, "note-events", false, "Subscribe to issue/PR comment events")
	f.BoolVar(&createFlags.mergeReqEvents, "merge-request-events", false, "Subscribe to pull request open/merge events")
	_ = createCmd.MarkFlagRequired("url")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	// Cobra's BoolVar doesn't have a "Changed()" sentinel for "user explicitly
	// passed --flag=false", so we treat the default (push=true) as a suggestion
	// and let the user override with --push-events=false. All other events
	// default to false on the server, so we send them only when the user
	// explicitly opts in.
	req := types.WebhookCreateRequest{
		URL:                createFlags.url,
		Title:              createFlags.title,
		Password:           createFlags.password,
		PushEvents:         utils.BoolPtr(createFlags.pushEvents),
		TagPushEvents:      utils.BoolPtr(createFlags.tagPushEvents),
		IssuesEvents:       utils.BoolPtr(createFlags.issuesEvents),
		NoteEvents:         utils.BoolPtr(createFlags.noteEvents),
		MergeRequestEvents: utils.BoolPtr(createFlags.mergeReqEvents),
	}
	if createFlags.password != "" {
		et := createFlags.encryptionType
		req.EncryptionType = &et
	}
	dc, err := cmd.NewDomainClient()
	if err != nil {
		return err
	}
	return dc.WebhookCreate(createFlags.repo, req)
}

const webhookCreateAgentHelp = `# gitee webhook create — Agent Help

## Purpose
Register a new webhook on a Gitee repository. The delivery URL will
receive HTTP POST requests whenever an event the webhook subscribes to
occurs in the repo. Use this to wire Gitee events into CI/CD, chat
ops, monitoring, or any external system that speaks HTTP.

## Pre-conditions
- Authenticated with write/admin rights on the target repo
- --url is a publicly reachable HTTPS endpoint (HTTP allowed for testing)
- --url is non-empty (Cobra MarkFlagRequired)

## Required flags
  --url / -u   Delivery URL — the endpoint that will receive POSTs

## Optional flags
  --repo / -R                owner/repo (or set global --repo)
  --title / -t               Webhook title (display only, max 191 chars)
  --password                 Signing secret sent with the payload
  --encryption-type          0 = password (default), 1 = signing key
  --push-events              Subscribe to push (default true)
  --tag-push-events          Subscribe to tag push (default false)
  --issues-events            Subscribe to issue open/close (default false)
  --note-events              Subscribe to issue/PR comments (default false)
  --merge-request-events     Subscribe to PR open/merge (default false)

Pass --push-events=false to opt out of the push default. The other
event booleans are sticky: once --issues-events is set, the hook
receives issue events until you DELETE the hook or PATCH the field.

## Idempotency
NOT idempotent. Each call creates a NEW webhook. Re-running with the
same --url creates a duplicate (no de-dup on the server). Use
` + "`" + `gitee webhook list` + "`" + ` to discover existing IDs and ` + "`" + `gitee webhook edit` + "`" + `
to modify an existing one.

## Atomicity
Atomic. Either the webhook is created and an id is returned, or no
change happens.

## Side effects
- The delivery URL will start receiving POSTs on the next subscribed event
  (potentially within seconds — including events on existing PRs/issues
  if you subscribe to those types)
- Webhook count for the repo increases by 1
- No email or web notification is sent to the repo owner

## Post-conditions (on success)
- A new webhook with a new id is visible in ` + "`" + `gitee webhook list` + "`" + `
- Stdout contains the new id, delivery URL, and subscribed event list
- The delivery URL is NOT contacted during create (use ` + "`" + `gitee webhook test` + "`" + ` to verify)

## Output (stdout, table mode)
  ✓ Webhook created
    ID:    1234
    URL:   https://ci.example.com/hook
    Events:push,issues

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 403 Forbidden:    not a repo admin
- 404 Not Found:    repo doesn't exist
- 422 Unprocessable: --url is unreachable, --password too short, or
  --encryption-type is not 0 or 1

## Race condition warning ⚠️
The new webhook can fire BEFORE this command's stdout is parsed. If you
chain off this output (e.g. ` + "`" + `ID=$(gitee webhook create ... | grep ID | awk '{print $2}')` + "`" + `),
the hook is already live and may have fired. If you need to disable it
immediately, do not create one that auto-fires.

## Next steps (orchestration hints)
1. Create the hook: ` + "`" + `gitee webhook create -R owner/repo -u https://x/hook` + "`" + `
2. Test delivery: ` + "`" + `gitee webhook test $ID` + "`" + ` (verifies the URL accepts POSTs)
3. Inspect last delivery: ` + "`" + `gitee webhook view $ID` + "`" + ` (shows result/result_code)
4. To unsubscribe one event type: ` + "`" + `gitee webhook edit $ID --push-events=false` + "`" + `

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour. Each call = 1 request.

## Example (closing the loop)
  # 1. Create the webhook
  ID=$(gitee webhook create -R owner/repo -u https://ci.example.com/hook \
       --issues-events --push-events | awk '/^  ID:/ {print $2}')

  # 2. Send a test ping to verify the URL accepts POSTs
  gitee webhook test "$ID"

  # 3. Inspect the last delivery result
  gitee webhook view "$ID"

## See also
- ` + "`" + `gitee webhook list` + "`" + `   — find existing webhooks
- ` + "`" + `gitee webhook view` + "`" + `   — inspect a single webhook
- ` + "`" + `gitee webhook edit` + "`" + `   — change URL, events, or password
- ` + "`" + `gitee webhook test` + "`" + `   — fire a test ping without waiting for a real event
- ` + "`" + `gitee webhook delete` + "`" + ` — remove the webhook entirely
`
