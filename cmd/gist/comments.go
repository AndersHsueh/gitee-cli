package gist

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var commentsFlags struct {
	limit int
	page  int
}

var commentsCmd = &cobra.Command{
	Use:   "comments <gist_id>",
	Short: "List comments on a gist",
	Example: `  gitee gist comments abc123def
  gitee gist comments abc123def --output json`,
	Args:    cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gistCommentsAgentHelp)
		}
		return nil
	},
	RunE: runGistComments,
}

func init() {
	f := commentsCmd.Flags()
	f.IntVarP(&commentsFlags.limit, "limit", "L", 30, "Maximum number of comments to return")
	f.IntVar(&commentsFlags.page, "page", 1, "Page number (1-based)")
}

func runGistComments(cobraCmd *cobra.Command, args []string) error {
	gistID := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.GistCommentsList(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var comments []types.GistComment
	if err := client.Get(ctx, path, &comments); err != nil {
		return fmt.Errorf("listing gist comments: %w", err)
	}

	// Apply --limit (page-level, naive truncation; matches how other
	// Gitee list commands handle the same flag without --paginate).
	if commentsFlags.limit > 0 && len(comments) > commentsFlags.limit {
		comments = comments[:commentsFlags.limit]
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(comments, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Comments on gist %s:\n", gistID)
	if len(comments) == 0 {
		fmt.Println("  (no comments)")
		return nil
	}
	for _, c := range comments {
		fmt.Printf("  [%s] %s: %s\n", c.CreatedAt, c.User.Login, c.Body)
	}
	return nil
}

// gistCommentsAgentHelp is the agent-targeted help for
// `gitee gist comments`. Distinct from `gitee gist comment` (singular,
// POST a new comment) and `gitee gist comment-delete` (DELETE one).
const gistCommentsAgentHelp = `# gitee gist comments — Agent Help

## Purpose
List all conversation comments on a single gist, in chronological
order. Use this to read what readers of the gist have already
discussed before posting a follow-up or removing off-topic notes.
Distinct from ` + "`" + `gitee gist comment` + "`" + ` (singular, POST a new
comment) and ` + "`" + `gitee gist comment-delete` + "`" + ` (DELETE one).

## Pre-conditions
- Authenticated (token in config, env, or --token)
- <gist_id> is a valid gist id (public gists are readable by any
  authenticated user; private gists require owner / collaborator)
- Read access to the gist

## Required arguments
  <gist_id>   Gist id (enforced by cobra.ExactArgs(1))

## Optional flags
  --limit / -L   Maximum number of comments to return (default 30)
  --page         Page number for single-page mode (default 1)
  --output json  Emit a JSON array of comment objects instead of a list

## Idempotency
YES. Read-only. Calling it twice yields the same list (modulo
concurrent comment activity). Safe to retry on transient errors.

## Atomicity
A single GET. The call either succeeds and returns the current
snapshot or fails — no partial-success state.

## Side effects
- None on the gist itself. No notifications, no webhooks fire.
- The token's read access is recorded in Gitee's access log.
- Rate-limit counter advances by 1.

## Post-conditions (on success)
- Stdout contains a human-readable list (default) or a JSON array
  (with --output json) of gist comment objects
- Exit status 0
- No file or network state mutated

## Output (stdout, table mode)
  Comments on gist abc123def:
    [2026-06-25T10:00:00Z] alice: Nice trick with the slice here.
    [2026-06-26T08:30:00Z] bob:   Thanks, glad it helps.
    [2026-06-28T14:15:00Z] alice: Closing in on a release.

## Output (stdout, json mode)
  [
    {
      "id": 12345,
      "body": "Nice trick with the slice here.",
      "user": { "login": "alice", "...": "..." },
      "created_at": "2026-06-25T10:00:00Z",
      "updated_at": "2026-06-25T10:00:00Z"
    }
  ]

  Each object has: id, body, user.login, created_at, updated_at.

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    no read access to the gist (private, not yours)
- 404 Not Found:    gist id doesn't exist
- Network timeouts: bump ` + "`" + `--timeout` + "`" + ` and retry

## Next steps (orchestration hints)
After listing comments, an agent typically wants to:
1. Decide whether to post a follow-up: ` + "`" + `gitee gist comment <gist_id> --body "..."` + "`" + `
2. Decide whether to remove an off-topic comment: ` + "`" + `gitee gist comment-delete <gist_id> <comment_id>` + "`" + `
   (only the author of the comment OR the gist owner can delete)
3. Skip the loop if the latest comment already covers the same ground
   (use jq to check ` + "`" + `.[-1].body` + "`" + `)

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour. Calling this once costs 1 request.

## Example (agent-friendly)
  # 1. List all comments as JSON
  gitee gist comments abc123def --output json > comments.json

  # 2. Show the latest comment body
  jq -r '.[-1].body' comments.json

  # 3. Count comments per author
  jq -r '.[].user.login' comments.json | sort | uniq -c | sort -rn

  # 4. If we haven't acknowledged the latest, post a reply
  LATEST=$(jq -r '.[-1].body' comments.json)
  if [ "$LATEST" != "Pushed the fix, thanks." ]; then
      gitee gist comment abc123def --body "Pushed the fix, thanks."
  fi

## See also
- ` + "`" + `gitee gist comment <gist_id>` + "`" + `      — POST a new comment
- ` + "`" + `gitee gist comment-delete <gist_id> <id>` + "`" + ` — DELETE one
- ` + "`" + `gitee gist view <gist_id>` + "`" + `       — read the gist body
- ` + "`" + `gitee --agent-help` + "`" + `              — overview of all commands
`
