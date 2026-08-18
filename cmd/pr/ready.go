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

var readyFlags struct {
	owner string
	repo  string
	id    int64
}

var readyCmd = &cobra.Command{
	Use:   "ready <number>",
	Short: "Mark a draft pull request as ready for review",
	Example: `  gitee pr ready 42 --repo owner/repo
  gitee pr ready 42 -R owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(prReadyAgentHelp)
		}
		return nil
	},
	RunE: runReady,
}

func init() {
	f := readyCmd.Flags()
	f.StringVar(&readyFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&readyFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.Int64VarP(&readyFlags.id, "id", "i", 0, "PR number (alternative to positional)")
}

func runReady(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	prNum, err := utils.ParseInt64Arg(readyFlags.id, args, "PR")
	if err != nil {
		return err
	}
	if prNum <= 0 {
		return fmt.Errorf("PR number is required (positional or --id)")
	}

	owner, repoName, err := resolvePRRepo(readyFlags.owner, readyFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	// Gitee v5 PATCH /pulls/{n} with {"draft": false} marks the PR ready
	// for review. utils.BoolPtr(false) ensures we send the explicit false
	// rather than letting omitempty drop the field.
	req := types.PREditRequest{Draft: utils.BoolPtr(false)}

	path := api.PRsEdit(owner, repoName, int(prNum))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Patch(ctx, path, req, &pr); err != nil {
		return fmt.Errorf("marking PR ready: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("\n%s PR #%s marked ready for review\n", green("✓"), bold(fmt.Sprintf("%d", prNum)))
	if pr.HTMLURL != "" {
		fmt.Printf("  URL: %s\n", pr.HTMLURL)
	}
	fmt.Println()
	return nil
}

const prReadyAgentHelp = `# gitee pr ready — Agent Help

## Purpose
Mark a draft pull request as ready for review. Gitee v5 stores a
"draft" boolean on PRs (analogous to GitHub draft PRs); this command
flips it to false so reviewers can be requested and merge becomes
available.

## Pre-conditions
- Authenticated with write access to the target repo
- <number> is an existing PR id
- The PR is currently in draft state (running on a non-draft PR is a
  no-op on the server but still costs 1 API call)

## Required arguments
  <number>   PR number (the #N displayed in ` + "`" + `gitee pr list` + "`" + `)

## Optional flags
  --repo / -R   owner/repo
  --id / -i     Alternative to positional

## Idempotency
YES. Re-running on an already-ready PR is a no-op (server keeps
` + "`" + `draft: false` + "`" + `). Cost: 1 API call per run.

## Atomicity
Atomic. Either the flag flips, or no change.

## Side effects
- Reviewers can now be requested on the PR (Gitee auto-disables review
  requests on draft PRs)
- The "draft" badge is removed from the PR list / detail view
- Subscribers (watchers, assignees) may receive a notification that
  the PR is ready for review
- No email blast by default

## Post-conditions (on success)
- The PR's ` + "`" + `draft` + "`" + ` field is ` + "`" + `false` + "`" + ` (visible in ` + "`" + `gitee pr view` + "`" + `)
- The PR can be merged via ` + "`" + `gitee pr merge` + "`" + ` if all checks pass
- Stdout shows the PR number and HTML URL

## Output (stdout)
  ✓ PR #42 marked ready for review
    URL: https://gitee.com/owner/repo/pulls/42

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    not a repo collaborator
- 404 Not Found:    PR number doesn't exist
- 422 Unprocessable: Gitee may reject the PATCH if the PR is in a
  state that prevents the flag flip (e.g. already merged). Check
  ` + "`" + `gitee pr view` + "`" + `.

## Race condition warning ⚠️
The shared mutable state is the PR's draft flag (and, transitively,
its mergeability). This command itself is idempotent — re-running is a
no-op — but the flag can be flipped back: a concurrent
` + "`" + `gitee pr edit <N> --draft` + "`" + ` between your ready and a follow-up
` + "`" + `gitee pr merge <N>` + "`" + ` re-drafts the PR, and the merge then fails with
422. Also note that ready does NOT make a PR mergeable by itself: CI
and review requirements still apply and can change at any time. Before
merging, confirm with a fresh ` + "`" + `gitee pr view <N> --output json` + "`" + `
that .draft is still false and required checks are green.

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour.

## Example (closing the loop)
  # 1. Find a draft PR
  gitee pr list -R owner/repo --output json | jq '.[] | select(.draft) | .number'

  # 2. Mark it ready
  gitee pr ready 42 -R owner/repo

  # 3. Verify the flag flipped
  gitee pr view 42 -R owner/repo | grep Draft

## See also
- ` + "`" + `gitee pr list` + "`" + `   — find draft PRs (jq select on .draft)
- ` + "`" + `gitee pr view` + "`" + `   — inspect current state
- ` + "`" + `gitee pr edit` + "`" + `   — edit title/body/state/draft (this command is a shortcut for ` + "`" + `--no-draft` + "`" + `)
- ` + "`" + `gitee pr merge` + "`" + ` — merge once ready
`
