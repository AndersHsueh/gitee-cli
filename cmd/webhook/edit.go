package webhook

import (
	"fmt"
	"os"

	"gitee-cli/cmd"
	"gitee-cli/internal/handler"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"
	"github.com/spf13/cobra"
)

var editFlags struct {
	repo           string
	id             int64
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

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit an existing webhook (URL, events, or password)",
	Example: `  # Mute push events
  gitee webhook edit 1234 --repo owner/repo --no-push-events

  # Rotate the signing secret
  gitee webhook edit 1234 -R owner/repo --password "new s3cret"

  # Change the delivery URL
  gitee webhook edit 1234 -R owner/repo --url https://new.example.com/hook

  # Subscribe to a new event
  gitee webhook edit 1234 -R owner/repo --merge-request-events`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(webhookEditAgentHelp)
			os.Exit(0)
		}
		return nil
	},
	RunE: runEdit,
}

func init() {
	f := editCmd.Flags()
	f.StringVarP(&editFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&editFlags.url, "url", "u", "", "New delivery URL (required by Gitee PATCH; pass the current URL if not changing it)")
	f.StringVarP(&editFlags.title, "title", "t", "", "New title (empty = leave unchanged)")
	f.StringVar(&editFlags.password, "password", "", "New signing secret (empty = leave unchanged)")
	f.IntVar(&editFlags.encryptionType, "encryption-type", 0, "Encryption type for --password: 0=password (default), 1=signing key")

	// 5 event subscriptions. Default false so we can detect "user did not
	// pass the flag" via cobra Flags().Changed(name) and skip the field in
	// the PATCH body. Use --X to opt in, --no-X to opt out, omit to leave
	// unchanged.
	f.BoolVar(&editFlags.pushEvents, "push-events", false, "Subscribe to push events (use --no-push-events to unsubscribe; omit to leave unchanged)")
	f.BoolVar(&editFlags.tagPushEvents, "tag-push-events", false, "Subscribe to tag push events")
	f.BoolVar(&editFlags.issuesEvents, "issues-events", false, "Subscribe to issue open/close events")
	f.BoolVar(&editFlags.noteEvents, "note-events", false, "Subscribe to issue/PR comment events")
	f.BoolVar(&editFlags.mergeReqEvents, "merge-request-events", false, "Subscribe to pull request open/merge events")
}

func runEdit(cobraCmd *cobra.Command, args []string) error {
	id, err := utils.ParseInt64Arg(editFlags.id, args, "webhook")
	if err != nil {
		return err
	}
	if id <= 0 {
		return fmt.Errorf("webhook id is required (positional or --id)")
	}
	if editFlags.url == "" {
		return fmt.Errorf("--url is required by Gitee's PATCH contract (pass the current URL if not changing it; fetch via 'gitee webhook view')")
	}

	req := types.WebhookEditRequest{
		URL:                editFlags.url,
		Title:              editFlags.title,
		Password:           editFlags.password,
		PushEvents:         utils.BoolPtrIfChanged(cobraCmd, "push-events", editFlags.pushEvents),
		TagPushEvents:      utils.BoolPtrIfChanged(cobraCmd, "tag-push-events", editFlags.tagPushEvents),
		IssuesEvents:       utils.BoolPtrIfChanged(cobraCmd, "issues-events", editFlags.issuesEvents),
		NoteEvents:         utils.BoolPtrIfChanged(cobraCmd, "note-events", editFlags.noteEvents),
		MergeRequestEvents: utils.BoolPtrIfChanged(cobraCmd, "merge-request-events", editFlags.mergeReqEvents),
	}
	if editFlags.password != "" {
		et := editFlags.encryptionType
		req.EncryptionType = &et
	}
	return handler.WebhookEdit(editFlags.repo, id, req)
}

// boolPtrIfChanged moved to internal/utils/utils.go (utils.BoolPtrIfChanged)
// to share with cmd/pr/edit.go. Both call sites use the same cobra-idiomatic
// Changed() check; see utils for the full doc.

const webhookEditAgentHelp = `# gitee webhook edit — Agent Help

## Purpose
Modify an existing webhook: change its delivery URL, add/remove event
subscriptions, rotate the signing secret, or rename the title. Use this
instead of delete+create when you want to preserve the webhook id and
its delivery history (visible in ` + "`" + `gitee webhook view` + "`" + `'s ` + "`" + `result` + "`" + ` field).

## Pre-conditions
- Authenticated with write/admin rights on the target repo
- <id> is an existing webhook id
- --url is the (possibly new) delivery URL — Gitee's PATCH contract
  requires it; pass the current URL when not changing it

## Required arguments
  <id>     Webhook id

## Required flags
  --url / -u   The (possibly new) delivery URL. Gitee's PATCH payload
               requires the url field. If you are NOT changing the URL,
               first run ` + "`" + `gitee webhook view` + "`" + ` to discover the
               current URL and pass it back unchanged.

## Optional flags
  --repo / -R                owner/repo
  --title / -t               New title (empty = leave unchanged)
  --password                 New signing secret (empty = leave unchanged)
  --encryption-type          0 or 1 (only meaningful with --password)
  --push-events              Subscribe to push events (--no-push-events to opt out)
  --tag-push-events          Subscribe to tag push events
  --issues-events            Subscribe to issue open/close events
  --note-events              Subscribe to issue/PR comment events
  --merge-request-events     Subscribe to PR open/merge events

For each event flag, three choices: pass ` + "`" + `--X` + "`" + ` to opt in, pass
` + "`" + `--no-X` + "`" + ` to opt out, or omit both to leave the subscription unchanged.
This is the cobra-idiomatic way to disambiguate "unset" from "set to
false" on a BoolVar flag (Flags().Changed(name) reports whether the
flag was explicitly set).

## Idempotency
YES (kind of). Re-running with the same flag values is a no-op on the
server — the hook's event subscriptions and URL stay the same. It is
NOT, however, a "create or update" — the hook must already exist.

## Atomicity
Atomic. The PATCH is a single API call; either all changed fields are
updated or none are.

## Side effects
- The new URL starts receiving new event deliveries within seconds
  (if --url changed)
- Subscribed event types take effect on the next repo event
- Rotated --password invalidates the previous secret on the receiver
  side — coordinate with the receiving system BEFORE rotating
- No email or web notification is sent to the repo owner

## Post-conditions (on success)
- The hook now has the new field values
- Existing deliveries in flight are NOT replayed
- Stdout shows the updated id, URL, and event summary

## Output (stdout, table mode)
  ✓ Webhook updated (id=1234)
    URL:   https://new.example.com/hook
    Events:push,issues,merge_request

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    not a repo admin
- 404 Not Found:    webhook id doesn't exist
- 422 Unprocessable: --url is missing or unreachable, --password is
  too short, or --encryption-type is not 0/1

## Next steps (orchestration hints)
1. Fetch the current state: ` + "`" + `gitee webhook view 1234 -R owner/repo` + "`" + `
2. Apply changes: ` + "`" + `gitee webhook edit 1234 -R owner/repo --url ... --merge-request-events` + "`" + `
3. Verify: ` + "`" + `gitee webhook view 1234 -R owner/repo` + "`" + ` (check Events line)
4. If you rotated --password, fire a test ping: ` + "`" + `gitee webhook test 1234` + "`" + `

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour.

## Example (closing the loop)
  # 1. Mute push events on webhook 1234 (URL unchanged)
  URL=$(gitee webhook view 1234 -R owner/repo --output json | jq -r .url)
  gitee webhook edit 1234 -R owner/repo --url "$URL" --no-push-events

  # 2. Verify push is now off
  gitee webhook view 1234 -R owner/repo | grep Events

## See also
- ` + "`" + `gitee webhook view` + "`" + `    — inspect current state
- ` + "`" + `gitee webhook create` + "`" + `  — make a new hook
- ` + "`" + `gitee webhook test` + "`" + `    — verify delivery after changes
`
