package pr

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var commentFlags struct {
	owner string
	repo  string
	body  string
	id    int64
}

var commentCmd = &cobra.Command{
	Use:   "comment <number>",
	Short: "Add a comment to a pull request",
	Example: `  gitee pr comment 42 --repo owner/repo --body "LGTM, merging"
  gitee pr comment 42 -R owner/repo -b "Thanks for the fix!"`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prCommentAgentHelp)
		}
		return nil
	},
	RunE: runComment,
}

func init() {
	f := commentCmd.Flags()
	f.StringVar(&commentFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&commentFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.Int64VarP(&commentFlags.id, "id", "i", 0, "PR number (alternative to positional)")
	f.StringVarP(&commentFlags.body, "body", "b", "", "Comment body (required)")
	_ = commentCmd.MarkFlagRequired("body")
}

func runComment(cobraCmd *cobra.Command, args []string) error {
	prNum, err := utils.ParseInt64Arg(commentFlags.id, args, "PR")
	if err != nil {
		return err
	}
	if prNum <= 0 {
		return fmt.Errorf("PR number is required (positional or --id)")
	}

	owner, repoName, err := resolvePRRepo(commentFlags.owner, commentFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.PRCommentRequest{Body: commentFlags.body}
	path := api.PRsComments(owner, repoName, int(prNum))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Post(ctx, path, req, nil); err != nil {
		return fmt.Errorf("commenting on PR: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n%s Comment added to PR #%d\n", green("✓"), prNum)
	fmt.Println()
	return nil
}

const prCommentAgentHelp = `# gitee pr comment — Agent Help

## Purpose
Post a new comment on a pull request. Distinct from ` + "`" + `gitee pr comments` + "`" + `
(the plural LIST command). Use this to leave a general review note,
ask the author a question, or trigger a notification on the PR.

For diff (inline) comments tied to a specific file/line, Gitee v5 also
accepts optional ` + "`" + `--commit-id` + "`" + `, ` + "`" + `--path` + "`" + `, and ` + "`" + `--position` + "`" + ` form
parameters, but this CLI's ` + "`" + `comment` + "`" + ` verb is general-purpose
only. Inline diff comments require ` + "`" + `gitee pr review` + "`" + ` (the existing
review subcommand) which has a structured ` + "`" + `--event` + "`" + ` field.

## Pre-conditions
- Authenticated with read access (Gitee allows any authenticated user
  to comment on a PR they can see)
- <number> is an existing PR id
- --body is non-empty (Cobra MarkFlagRequired)

## Required arguments
  <number>   PR number

## Required flags
  --body / -b   Comment text (required; empty string rejected by Cobra)

## Optional flags
  --repo / -R    owner/repo
  --id / -i      Alternative to positional

## Idempotency
NO. Each call adds a new comment. Re-running duplicates the comment
(unlike e.g. ` + "`" + `gitee pr edit` + "`" + ` which is idempotent). Avoid re-running
in scripts without a check.

## Atomicity
Atomic on the API side. The comment either appears in the PR's
conversation or the call fails — no partial state.

## Side effects
- A new comment is added to the PR's conversation thread
- Subscribers (watchers, assignees, mentioned users) receive a
  notification (in-app and possibly email, depending on user
  preferences)
- The PR's ` + "`" + `updated_at` + "`" + ` timestamp advances
- No webhooks fire on plain comments (Gitee only fires webhooks on
  subscribed events like push / PR open / PR merge)

## Post-conditions (on success)
- The comment is visible at the bottom of the PR conversation
- Stdout confirms the PR number

## Output (stdout)
  ✓ Comment added to PR #42

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    no access to the repo
- 404 Not Found:    PR number doesn't exist
- 422 Unprocessable: --body is empty (caught by Cobra before call) or
  exceeds the per-comment length limit (Gitee: ~65k chars)

## OAuth scope required
projects (write) — comments are write operations even though they
don't change code.

## Rate limit
Gitee API v5: 1000 req/hour. Comments are cheap but be polite in
high-traffic repos.

## Example (closing the loop)
  # 1. Read the latest existing comment
  LAST=$(gitee pr comments 42 -R owner/repo --limit 1 --output json | jq -r '.[0].body')

  # 2. Reply with a status update (avoid duplicate)
  if [ "$LAST" != "Deploying to staging now." ]; then
      gitee pr comment 42 -R owner/repo -b "Deploying to staging now."
  fi

## See also
- ` + "`" + `gitee pr comments` + "`" + `  — LIST existing comments
- ` + "`" + `gitee pr review` + "`" + `   — APPROVE / REQUEST_CHANGES / COMMENT with inline diff comments
- ` + "`" + `gitee issue comment` + "`" + ` — same for issues
`
