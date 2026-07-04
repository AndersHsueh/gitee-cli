package gist

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/spf13/cobra"
)

var commentDeleteCmd = &cobra.Command{
	Use:   "comment-delete <gist_id> <comment_id>",
	Short: "Delete a comment on a gist",
	Example: `  gitee gist comment-delete abc123def 12345
  gitee gist comment-delete abc123def 12345 --yes`,
	Args:    cmd.AgentHelpAwareArgs(cobra.ExactArgs(2)),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gistCommentDeleteAgentHelp)
		}
		return nil
	},
	RunE: runGistCommentDelete,
}

func init() {
	commentDeleteCmd.Flags().BoolVar(&gistCommentDeleteFlags.yes, "yes", false, "Skip the confirmation prompt")
}

var gistCommentDeleteFlags = struct {
	yes bool
}{}

func runGistCommentDelete(cobraCmd *cobra.Command, args []string) error {
	gistID := args[0]
	commentID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid comment id %q: must be an integer", args[1])
	}
	if commentID <= 0 {
		return fmt.Errorf("comment id must be positive")
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.GistCommentDelete(gistID, commentID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("deleting gist comment: %w", err)
	}

	fmt.Printf("Comment %d deleted from gist %s\n", commentID, gistID)
	return nil
}

// gistCommentDeleteAgentHelp is the agent-targeted help for
// `gitee gist comment-delete`. Destructive: only the comment's
// author or the gist owner can remove a comment.
const gistCommentDeleteAgentHelp = `# gitee gist comment-delete — Agent Help

## Purpose
Permanently delete a single comment on a gist. Use this to remove an
off-topic or unwanted note. Only the AUTHOR of the comment OR the
gist owner can delete it; the Gitee API returns 403 otherwise.

## Pre-conditions
- Authenticated (token in config, env, or --token)
- <gist_id> is an existing gist id
- <comment_id> is the numeric id of a comment on that gist (look it
  up via ` + "`" + `gitee gist comments <gist_id> --output json` + "`" + `)
- The token's user is either the comment author or the gist owner

## Required arguments
  <gist_id>     Gist id
  <comment_id>  Numeric comment id (NOT the gist id)

## Required flags
  none strictly required, but --yes is strongly recommended in
  non-interactive scripts.

## Optional flags
  --yes   Skip the confirmation prompt (no-op here for now; future
          versions may prompt by default)

## Idempotency
PARTIAL. The first call removes the comment. A second call returns
404 (the comment no longer exists), so re-running is safe in the
end-state sense — but each call costs one API request.

## Atomicity
Atomic at the API level. The comment either disappears entirely or
the call fails — there is no partial state.

## Side effects (IRREVERSIBLE)
- The comment is REMOVED from the gist conversation
- Subscribers who were watching the comment thread see no further
  updates from this comment
- No notifications are sent to the original author of the deleted
  comment by default
- The gist's ` + "`" + `updated_at` + "`" + ` timestamp does NOT advance (delete
  is not considered a content edit on the gist itself)

WARNING: There is NO UNDO. The Gitee API has no restore endpoint
for deleted gist comments. If you need the text for an audit log,
snapshot it before calling this command.

## Post-conditions (on success)
- The comment no longer appears in ` + "`" + `gitee gist comments <gist_id>` + "`" + `
- stdout includes "Comment <id> deleted from gist <gist_id>"
- exit code 0

## Output (stdout, text mode)
  Comment 12345 deleted from gist abc123def

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    the token's user is neither the comment author
                   nor the gist owner
- 404 Not Found:    gist id or comment id is wrong (also returned
                   when the comment was already deleted)
- 5xx Server error: transient — retry is safe (idempotent in the
                   end-state sense)

## Next steps (orchestration hints)
After a successful delete, an agent typically wants to:
1. Verify with ` + "`" + `gitee gist comments <gist_id> --output json` + "`" + ` and
   check the deleted id is absent
2. If a PR/issue referenced this comment, decide whether to update
   or close that reference

## OAuth scope required
projects (write) — delete is a state mutation.

## Rate limit
Gitee API v5: 1000 requests/hour. A single delete is 1 call.

## Example (agent-friendly)
  # 1. Find the comment id from the list
  ID=$(gitee gist comments abc123def --output json \
       | jq -r '.[].id' | head -1)

  # 2. Delete it
  gitee gist comment-delete abc123def "$ID"

  # 3. Verify it's gone
  gitee gist comments abc123def --output json | jq -e "[.[] | select(.id == $ID)] | length == 0"

## See also
- ` + "`" + `gitee gist comments` + "`" + `        — list comments (find the id)
- ` + "`" + `gitee gist comment` + "`" + `        — POST a new comment
- ` + "`" + `gitee --agent-help` + "`" + `        — overview of all commands
`
