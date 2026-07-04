package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var commentsFlags struct {
	owner string
	repo  string
}

var commentsCmd = &cobra.Command{
	Use:     "comments <number>",
	Short:   "List comments on a pull request",
	Example: `  gitee pr comments 42 --repo owner/repo`,
	Args:    cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	// PreRunE fires BEFORE the cobra arg check, so an agent can request
	// --agent-help even when <number> is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prCommentsAgentHelp)
		}
		return nil
	},
	RunE: runComments,
}

func init() {
	f := commentsCmd.Flags()
	f.StringVar(&commentsFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&commentsFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runComments(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolvePRRepo(commentsFlags.owner, commentsFlags.repo)
	if err != nil {
		return err
	}

	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: must be an integer", args[0])
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.PRsComments(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var comments []types.PRComment
	if err := client.Get(ctx, path, &comments); err != nil {
		return fmt.Errorf("listing PR comments: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(comments, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Comments on PR #%d:\n", prNum)
	for _, c := range comments {
		fmt.Printf("  [%s] %s: %s\n", c.CreatedAt, c.User.Login, c.Body)
	}
	return nil
}

// prCommentsAgentHelp is the agent-targeted help for `gitee pr comments`.
// LIST existing conversation comments on a PR. Distinct from
// ` + "`" + `gitee pr comment` + "`" + ` (singular, POST a new comment) and
// ` + "`" + `gitee pr review` + "`" + ` (POST a review event).
const prCommentsAgentHelp = `# gitee pr comments — Agent Help

## Purpose
List all conversation comments on a single pull request, in
chronological order. Use this to read what reviewers and the
author have already discussed before posting a new comment,
approving, or merging. Distinct from ` + "`" + `gitee pr comment` + "`" + `
(POST a new comment) and ` + "`" + `gitee pr review` + "`" + ` (POST a
review event with APPROVE / REQUEST_CHANGES / COMMENT state).

## Pre-conditions
- Authenticated (token in config, env, or --token)
- --repo (or --owner) points to an existing, accessible repository
- <number> is a valid PR id
- Read access to the repo

## Required arguments
  <number>   PR number (enforced by cobra.ExactArgs(1))

## Required flags
  None strictly required, but --repo / -R (or global --repo) is
  needed to disambiguate the target repo.

## Optional flags
  --owner            Repository owner
  --repo / -R        Repository name or owner/repo
  --paginate         Walk all pages of comments
  --limit N          Total cap across pages when --paginate is set
  --page N           Page number for single-page mode (default 1)
  --per-page N       Items per page (default 30, max 100)
  --output json      Emit a JSON array of comment objects instead of a list

## Idempotency
YES. Read-only. Calling it twice yields the same list (modulo
concurrent comment activity). Safe to retry on transient errors.

## Atomicity
A single GET (or paged sequence of GETs when --paginate is set).
Each call either succeeds and returns the current snapshot or
fails — no partial-success state.

## Side effects
- None on the Gitee repo. No notifications, no webhooks fire.
- The token's read access is recorded in Gitee's access log.
- Rate-limit counter advances by 1 (or N pages when --paginate is set).

## Post-conditions (on success)
- Stdout contains a human-readable list (default) or a JSON array
  (with --output json) of PR comment objects
- Exit status 0
- No file or network state mutated

## Output (stdout, table mode)
  Comments on PR #42:
    [2026-06-25T10:00:00Z] alice: Looks good, just one nit.
    [2026-06-26T08:30:00Z] bob:   Pushed a fix for the nit.
    [2026-06-28T14:15:00Z] alice: Approved.

## Output (stdout, json mode)
  [
    {
      "id": 12345,
      "body": "Looks good, just one nit.",
      "user": { "login": "alice", "...": "..." },
      "created_at": "2026-06-25T10:00:00Z",
      "updated_at": "2026-06-25T10:00:00Z",
      "...": "..."
    }
  ]

  Each object has: id, body, user.login, created_at, updated_at,
  and (for inline diff comments) path, position, commit_id.

## Common errors
- 401 Unauthorized:  token missing or expired
- 403 Forbidden:     no read access to the repo
- 404 Not Found:     PR number doesn't exist
- Network timeouts:  bump ` + "`" + `--timeout` + "`" + ` and retry

## Next steps (orchestration hints)
After listing comments, an agent typically wants to:
1. Decide whether to post a follow-up:
   ` + "`" + `gitee pr comment <N> --body "..."` + "`" + ` (general) or
   ` + "`" + `gitee pr review <N> --approve` + "`" + ` (formal review)
2. Skip the loop if the latest comment already covers the same
   ground (use jq to check ` + "`" + `.[-1].body` + "`" + `)
3. View the diff for context: ` + "`" + `gitee pr diff <N>` + "`" + `
4. After approval: ` + "`" + `gitee pr merge <N>` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour. With --per-page 100, a PR with
500 comments costs 5 calls.

## Example (agent-friendly)
  # 1. List all comments as JSON
  gitee pr comments 42 --repo owner/repo --output json > comments.json

  # 2. Show the latest comment body
  jq -r '.[-1].body' comments.json

  # 3. Count comments per author
  jq -r '.[].user.login' comments.json | sort | uniq -c | sort -rn

  # 4. If we haven't acknowledged the latest, post a reply
  LATEST=$(jq -r '.[-1].body' comments.json)
  if [ "$LATEST" != "Pushed the fix, thanks." ]; then
      gitee pr comment 42 --repo owner/repo \
        --body "Pushed the fix, thanks."
  fi

## See also
- ` + "`" + `gitee pr comment <N>` + "`" + `   — POST a new comment
- ` + "`" + `gitee pr review <N>` + "`" + `    — POST a formal review event
- ` + "`" + `gitee pr view <N>` + "`" + `      — read PR metadata
- ` + "`" + `gitee issue comments <N>` + "`" + ` — equivalent for issues
- ` + "`" + `gitee --agent-help` + "`" + `    — overview of all commands with agent help
`
