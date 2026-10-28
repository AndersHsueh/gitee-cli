package pr

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

var createFlags struct {
	owner  string
	repo   string
	title  string
	body   string
	source string
	target string
	draft  bool
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request",
	Example: `  gitee pr create --repo owner/repo --title "Add feature" --source feature --target main
  gitee pr create --repo owner/repo --title "Fix bug" --source fix --target dev --draft`,
	// PreRunE fires BEFORE ValidateRequiredFlags, so an agent can request
	// --agent-help even when required flags (e.g. --title, --source, --target)
	// are missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(prCreateAgentHelp)
		}
		return nil
	},
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVar(&createFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&createFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVarP(&createFlags.title, "title", "t", "", "PR title (required)")
	f.StringVarP(&createFlags.body, "body", "b", "", "PR description")
	f.StringVar(&createFlags.source, "source", "", "Source branch name (required)")
	f.StringVar(&createFlags.target, "target", "", "Target branch name (required)")
	f.BoolVar(&createFlags.draft, "draft", false, "Create as draft PR")
	_ = createCmd.MarkFlagRequired("title")
	_ = createCmd.MarkFlagRequired("source")
	_ = createCmd.MarkFlagRequired("target")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	owner, repoName, err := resolvePRRepo(createFlags.owner, createFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	req := types.PRCreateRequest{
		Title: createFlags.title,
		Body:  createFlags.body,
		Head:  createFlags.source,
		Base:  createFlags.target,
		Draft: createFlags.draft,
	}

	path := api.PRsCreate(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Post(ctx, path, req, &pr); err != nil {
		return fmt.Errorf("creating PR: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s PR #%d created: %s\n", green("✓"), pr.Number, bold(pr.Title))
	fmt.Printf("  URL: %s\n", color.CyanString(pr.HTMLURL))
	return nil
}

// prCreateAgentHelp is the agent-targeted help for `gitee pr create`.
// It is intentionally long, structured, and includes machine-relevant
// details (idempotency, side effects, orchestration hints) that the
// default --help omits because they're noise for humans.
//
// Maintainer note: keep this in sync with the actual command behavior.
// When flags change, update the relevant section here.
const prCreateAgentHelp = `# gitee pr create — Agent Help

## Purpose
Create a new pull request in a Gitee repository. Use this when an agent has
finished implementing a fix or feature on a branch and needs to surface it
for human review.

## Pre-conditions
- Authenticated: gitee auth status returns a valid token
- Repository exists: --repo (or --owner + --repo) points to an accessible repo
- Source branch exists on the remote: --source must be pushed (git push origin <branch>)
- Target branch exists in the repo: --target must be a valid branch name
- --title is non-empty (enforced)
- --source and --target are non-empty (enforced)
- Write access: token must have write permission on the target repo

## Required flags
  --title / -t       PR title (required, non-empty)
  --source           Source branch name (required)
  --target           Target branch name (required)

## Optional flags
  --body / -b        PR description (Markdown supported)
  --draft            Create as draft PR (NOT mergeable until promoted)
  --owner            Repository owner
  --repo / -R        Repository name or owner/repo

## Idempotency
NOT idempotent. Each invocation creates a NEW PR with a fresh number.
If a PR for the same source→target already exists, Gitee returns 409 Conflict.
To "update" an existing PR's content, use ` + "`" + `gitee pr edit` + "`" + ` (NOT this command).

## Atomicity
Atomic. Either the PR is created and a number is returned, or no
change happens. There is no partial-success state.

## Side effects
- Email notification sent to all repo watchers AND target branch reviewers
- Webhooks subscribed to "Pull Request" events fire
- The PR is in "open" state by default, or "draft" if --draft is set
- The PR appears in the repo's PR list immediately
- If --draft is set: the PR is NOT mergeable until promoted to "ready for review"

## Post-conditions (on success)
- A PR with a new unique number exists
- Stdout includes the PR number (#N) and URL
- The PR is in "open" or "draft" state
- The PR links source branch → target branch

## Output (stdout, table mode)
  ✓ PR #18 created: fix: handle null pointer
    URL: https://gitee.com/owner/repo/pulls/18

## Parsing the PR number
  PR_OUT=$(gitee pr create --repo owner/repo --title "..." --source fix/x --target main)
  PR_NUM=$(echo "$PR_OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

## Common errors
- 401 Unauthorized:   token expired                  →  gitee auth login
- 403 Forbidden:      no write access                →  check token scopes
- 404 Not Found:      repo or branch missing         →  verify spelling, ensure branch is pushed
- 409 Conflict:       PR already exists              →  check existing PRs, or use it
- 422 Unprocessable:  validation failed              →  check source/target branch names

## Race condition warning ⚠️
The shared mutable state is the repo's (source→target) PR mapping.
This command is NOT idempotent: a retry after a network timeout either
fails with 409 Conflict (the first attempt landed) or creates a
second, duplicate-intent PR. On 409, do NOT retry — find the existing
PR with ` + "`" + `gitee pr list --head <you:source>` + "`" + ` and use its number.
Conversely, DO NOT call ` + "`" + `gitee pr list` + "`" + ` immediately after a
successful create to verify: Gitee's list indexing can lag 1-2 seconds
and the new PR may briefly be missing. Use the PR number from THIS
command's stdout directly. Two agents racing to open a PR for the same
branch pair: one wins, the other gets 409.

## Next steps (orchestration hints) — CRITICAL

After a successful PR create, an agent MUST close the loop with the
originating issue. Skipping this leaves the issue in an orphan state:
the PR exists, the issue is still "open", and a human reading the issue
has no way to know a fix is already available.

Standard closing-the-loop sequence:
1. If the PR fixes an issue, comment on that issue with the PR reference:
     gitee issue comment {ISSUE_NUM} --body "Fixed in PR #{PR_NUM}"
2. The PR will NOT auto-link the issue — explicit commenting is required
   (even if the PR body contains "Closes #N", that only takes effect on
    merge, and is invisible until then).
3. After human review and merge, the issue can be closed:
     gitee issue close {ISSUE_NUM}

⚠️  Why this matters: in the test-agent → coder-agent workflow, the issue
    is filed by one agent and fixed by another. The human reads issues
    FIRST when they wake up, not PRs. If the issue doesn't mention the PR,
    the human re-files a duplicate task. This is the #1 reason agents
    leave orphan issues in repos.

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example (agent-friendly — closing the loop end-to-end)
  # 1. Create the PR
  PR_OUT=$(gitee pr create --repo owner/repo \
    --title "fix: handle null pointer" \
    --source fix/null-pointer --target main \
    --body "Closes #42")
  PR_NUM=$(echo "$PR_OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

  # 2. Link the PR back to the originating issue (CRITICAL — see above)
  gitee issue comment 42 --body "Fixed in PR #${PR_NUM}"

  # 3. (After human merges the PR) Close the issue
  gitee issue close 42

## See also
- ` + "`" + `gitee pr list` + "`" + `              — find existing PRs
- ` + "`" + `gitee pr view N` + "`" + `            — read a PR
- ` + "`" + `gitee pr merge N` + "`" + `           — merge a PR (human action)
- ` + "`" + `gitee pr review N` + "`" + `          — submit a review
- ` + "`" + `gitee issue comment N` + "`" + `      — link a PR to an issue (essential!)
- ` + "`" + `gitee issue create` + "`" + `         — open a new issue
- ` + "`" + `gitee --agent-help` + "`" + `         — overview of all commands with agent help
`
