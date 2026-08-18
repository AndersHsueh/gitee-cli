package gist

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var commentFlags struct {
	body string
}

var commentCmd = &cobra.Command{
	Use:   "comment <gist_id>",
	Short: "Add a comment to a gist",
	Example: `  gitee gist comment abc123def --body "Nice trick with the slice here."
  gitee gist comment abc123def -b "Thanks for the fix!"`,
	Args:    cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(gistCommentAgentHelp)
		}
		return nil
	},
	RunE: runGistComment,
}

func init() {
	f := commentCmd.Flags()
	f.StringVarP(&commentFlags.body, "body", "b", "", "Comment body (required)")
	_ = commentCmd.MarkFlagRequired("body")
}

func runGistComment(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	gistID := args[0]

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	req := types.GistCommentRequest{Body: commentFlags.body}
	path := api.GistCommentCreate(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var created types.GistComment
	if err := client.Post(ctx, path, req, &created); err != nil {
		return fmt.Errorf("commenting on gist: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Comment added to gist %s\n", green("✓"), gistID)
	if created.ID != 0 {
		fmt.Printf("  Comment id: %d\n", created.ID)
	}
	return nil
}

// gistCommentAgentHelp is the agent-targeted help for
// `gitee gist comment` (singular; POST a new comment). Distinct from
// `gitee gist comments` (plural LIST) and `gitee gist comment-delete`
// (DELETE one).
const gistCommentAgentHelp = `# gitee gist comment — Agent Help

## Purpose
Post a new comment on a gist. Distinct from ` + "`" + `gitee gist comments` + "`" + `
(the plural LIST command) and ` + "`" + `gitee gist comment-delete` + "`" + `
(DELETE one). Use this to leave a general note, ask the author a
question, or trigger a notification on the gist.

## Pre-conditions
- Authenticated with read access to the gist
- <gist_id> is an existing gist id
- --body is non-empty (Cobra MarkFlagRequired)

## Required arguments
  <gist_id>   Gist id (enforced by cobra.ExactArgs(1))

## Required flags
  --body / -b   Comment text (required; empty string rejected by Cobra)

## Optional flags
  none — only the positional <gist_id> and the required --body are
  consumed by this command.

## Idempotency
NO. Each call adds a new comment. Re-running duplicates the comment
(unlike e.g. ` + "`" + `gitee gist edit` + "`" + ` which is idempotent). Avoid
re-running in scripts without a check.

## Atomicity
Atomic on the API side. The comment either appears in the gist's
conversation or the call fails — no partial state.

## Side effects
- A new comment is added to the gist's conversation thread
- Subscribers (watchers of the gist) receive a notification (in-app
  and possibly email, depending on user preferences)
- The gist's ` + "`" + `updated_at` + "`" + ` timestamp advances
- No webhooks fire on plain comments

## Post-conditions (on success)
- The comment is visible at the bottom of the gist conversation
- Stdout confirms the gist id and (when the API returns it) the new
  comment id

## Output (stdout)
  ✓ Comment added to gist abc123def
    Comment id: 12345

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    no read access to the gist
- 404 Not Found:    gist id doesn't exist
- 422 Unprocessable: --body is empty (caught by Cobra before call) or
  exceeds the per-comment length limit (Gitee: ~65k chars)

## Race condition warning ⚠️
The gist's comment thread is shared mutable state. Specific risks:
- NOT idempotent: retrying after a timeout or 5xx (where the server
  may actually have succeeded) posts the SAME comment twice. Before
  a blind retry, check ` + "`" + `gitee gist comments <gist_id> --output json` + "`" + `
  for your body text.
- If you only comment under a condition (e.g. "reply only if nobody
  answered yet"), re-run ` + "`" + `gitee gist comments` + "`" + ` immediately before
  this call — another agent may have posted in the gap.
- The new comment may not appear in ` + "`" + `gitee gist comments` + "`" + ` for
  1-2 seconds (indexing lag). Use the "Comment id:" line from THIS
  output; do not re-list to find it.

## Next steps (orchestration hints)
After posting, an agent typically wants to:
1. Verify with ` + "`" + `gitee gist comments <gist_id> --output json` + "`" + ` and
   check the new id appears at the tail
2. Decide whether to delete an off-topic earlier comment:
   ` + "`" + `gitee gist comment-delete <gist_id> <id>` + "`" + `

## OAuth scope required
projects (write) — comments are write operations even though they
don't change gist content.

## Rate limit
Gitee API v5: 1000 req/hour. Comments are cheap but be polite on
high-traffic gists.

## Example (closing the loop)
  # 1. Read the latest existing comment
  LAST=$(gitee gist comments abc123def --limit 1 --output json | jq -r '.[0].body')

  # 2. Reply with a status update (avoid duplicate)
  if [ "$LAST" != "Deploying to staging now." ]; then
      gitee gist comment abc123def --body "Deploying to staging now."
  fi

## See also
- ` + "`" + `gitee gist comments` + "`" + `        — LIST existing comments
- ` + "`" + `gitee gist comment-delete` + "`" + ` — DELETE one by id
- ` + "`" + `gitee gist view` + "`" + `           — read the gist body
- ` + "`" + `gitee --agent-help` + "`" + `        — overview of all commands
`
