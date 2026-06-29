package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var reviewFlags struct {
	owner          string
	repo           string
	approve        bool
	requestChanges bool
	comment        bool
	body           string
}

var reviewCmd = &cobra.Command{
	Use:   "review <number>",
	Short: "Review a pull request",
	Example: `  gitee pr review 42 --repo owner/repo --approve
  gitee pr review 42 --repo owner/repo --request-changes --body "Needs tests"`,
	// PreRunE fires BEFORE the cobra arg check, so an agent can request
	// --agent-help even when <number> is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prReviewAgentHelp)
		}
		return nil
	},
	RunE: runReview,
}

func init() {
	f := reviewCmd.Flags()
	f.StringVar(&reviewFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&reviewFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.BoolVar(&reviewFlags.approve, "approve", false, "Approve the PR")
	f.BoolVar(&reviewFlags.requestChanges, "request-changes", false, "Request changes on the PR")
	f.BoolVar(&reviewFlags.comment, "comment", false, "Comment on the PR")
	f.StringVarP(&reviewFlags.body, "body", "b", "", "Review comment")
}

func runReview(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(reviewFlags.owner, reviewFlags.repo)
	if err != nil {
		return err
	}

	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: must be an integer", args[0])
	}

	// Determine review event
	event := "comment"
	if reviewFlags.approve {
		event = "approve"
	} else if reviewFlags.requestChanges {
		event = "request_changes"
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.PRReviewRequest{
		Body:  reviewFlags.body,
		Event: event,
	}

	path := api.PRsReview(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Post(ctx, path, req, nil); err != nil {
		return fmt.Errorf("reviewing PR: %w", err)
	}

	fmt.Printf("PR #%d reviewed: %s\n", prNum, event)
	return nil
}

// prReviewAgentHelp is the agent-targeted help for `gitee pr review`.
// Posts an approve / request-changes / comment event on a PR. Distinct
// from ` + "`" + `gitee pr comment` + "`" + ` (which posts a plain general comment
// without a review state) and ` + "`" + `gitee pr comments` + "`" + ` (LIST).
const prReviewAgentHelp = `# gitee pr review — Agent Help

## Purpose
Submit a formal review on a pull request: APPROVE, REQUEST_CHANGES,
or COMMENT. The event is recorded in the PR's review history and
governs whether the PR is mergeable (e.g. repos that require
approvals). Use this after reading the diff and existing comments
to record an automated or human review decision.

## Pre-conditions
- Authenticated (token in config, env, or --token)
- --repo (or --owner) points to an existing, accessible repository
- <number> is a valid PR id
- Token has write access to the repo (any reviewer role works)
- Exactly one of --approve, --request-changes, --comment is given
  (default is COMMENT if none is passed)
- --body is strongly recommended for --request-changes; the Gitee
  web UI marks such comments as actionable

## Required arguments
  <number>   PR number

## Required flags
  None strictly required, but you almost always want to pass exactly
  ONE of --approve / --request-changes / --comment. The default event
  is "comment" if none is specified.

## Optional flags
  --owner              Repository owner
  --repo / -R          Repository name or owner/repo
  --approve            Submit an APPROVE review
  --request-changes    Submit a REQUEST_CHANGES review (blocks merge in
                       repos that enforce it)
  --comment            Submit a plain COMMENT review (no approve/changes)
  --body / -b          Review comment text (required for --request-changes
                       in most workflows; optional for the other two)

## Idempotency
NO. Each call adds a new review event. Re-running with the same flags
posts a duplicate review (the PR's review thread will show two
identical entries). Use ` + "`" + `gitee pr comments <N>` + "`" + ` to read
existing reviews first if you need to avoid duplicates.

## Atomicity
Atomic at the API level. The review is either recorded or the call
fails — there is no partial state.

## Side effects
- A new review event appears in the PR's review history
- The PR's ` + "`" + `updated_at` + "`" + ` timestamp advances
- Subscribers (watchers, assignees, PR author) get a notification
  (in-app; possibly email depending on user settings)
- The PR's mergeability state may change:
    - With --approve: required-approvals count goes up
    - With --request-changes: the PR is blocked from merging
      in repos that enforce "no changes requested"
- Webhooks subscribed to "PR review" events fire
- No commit, no branch head, no merge state is mutated

## Post-conditions (on success)
- Stdout: ` + "`" + `PR #N reviewed: approve | request_changes | comment` + "`" + `
- The new review is visible in the PR's "Conversation" tab on the web UI
- Exit status 0

## Output (stdout, text mode)
  PR #42 reviewed: approve
  PR #42 reviewed: request_changes
  PR #42 reviewed: comment

## Common errors
- 401 Unauthorized:    token missing or expired
- 403 Forbidden:       no write access (or you can't review your own PR
                       in repos that disallow self-review)
- 404 Not Found:       PR number doesn't exist
- 422 Unprocessable:   invalid event value, empty body on request_changes,
                       or PR is closed/merged and no longer accepts reviews
- Network timeouts:    bump ` + "`" + `--timeout` + "`" + ` and retry

## Next steps (orchestration hints)
After submitting a review, an agent typically wants to:
1. Re-list PR comments to see the review appear:
   ` + "`" + `gitee pr comments <N>` + "`" + `
2. If approving: wait for CI to pass then ` + "`" + `gitee pr merge <N>` + "`" + `
3. If requesting changes: stop; wait for the author to push fixes
4. If commenting: continue with whatever triggered the review

## OAuth scope required
projects (write) — reviews are write operations.

## Rate limit
Gitee API v5: 1000 requests/hour. A single review is 1 call.

## Example (agent-friendly)
  # 1. List existing reviews / comments to avoid duplicates
  gitee pr comments 42 --repo owner/repo --output json | jq -r '.[].body'

  # 2. Approve with a short message
  gitee pr review 42 --repo owner/repo --approve --body "LGTM, merging."

  # Or, request changes with an explanation
  gitee pr review 42 --repo owner/repo --request-changes \
    --body "Please add a test for the empty-input case."

  # 3. After approval + green CI, merge
  gitee pr checks 42 --repo owner/repo
  gitee pr merge 42 --repo owner/repo

## See also
- ` + "`" + `gitee pr view <N>` + "`" + `         — read PR metadata
- ` + "`" + `gitee pr diff <N>` + "`" + `        — read the file-level diff first
- ` + "`" + `gitee pr comments <N>` + "`" + `   — LIST existing comments & reviews
- ` + "`" + `gitee pr comment <N>` + "`" + `    — plain comment (no review event)
- ` + "`" + `gitee pr checks <N>` + "`" + `     — verify CI before approving
- ` + "`" + `gitee pr merge <N>` + "`" + `       — merge after approve
- ` + "`" + `gitee --agent-help` + "`" + `      — overview of all commands with agent help
`
