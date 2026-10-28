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

var closeFlags struct {
	owner string
	repo  string
}

var closeCmd = &cobra.Command{
	Use:     "close <number>",
	Short:   "Close a pull request",
	Example: `  gitee pr close 42 --repo owner/repo`,
	// PreRunE fires BEFORE the cobra arg check, so an agent can request
	// --agent-help even when <number> is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(prCloseAgentHelp)
		}
		return nil
	},
	RunE: runClose,
}

func init() {
	f := closeCmd.Flags()
	f.StringVar(&closeFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&closeFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runClose(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(closeFlags.owner, closeFlags.repo)
	if err != nil {
		return err
	}

	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: must be an integer", args[0])
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.PRsGet(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	req := map[string]string{"state": "closed"}
	var pr types.PullRequest
	if err := client.Patch(ctx, path, req, &pr); err != nil {
		return fmt.Errorf("closing PR: %w", err)
	}

	fmt.Printf("PR #%d closed\n", prNum)
	return nil
}

// prCloseAgentHelp is the agent-targeted help for `gitee pr close`.
// Closes a PR WITHOUT merging it. Reversible via `gitee pr reopen`.
const prCloseAgentHelp = `# gitee pr close — Agent Help

## Purpose
Close a pull request WITHOUT merging it. The PR's head branch and
commits are not deleted, and the PR can be reopened with
` + "`" + `gitee pr reopen` + "`" + `. Use this when a PR is no longer wanted
(stale, superseded, abandoned) but its commits should be preserved
for reference.

## Pre-conditions
- Authenticated (token in config, env, or --token)
- --repo (or --owner) points to an existing, accessible repository
- <number> is a valid PR id
- Token has write access to the repo (any user with triage+ can close)
- The PR is currently in "open" state (closing a "closed" PR is a
  no-op error; closing a "merged" PR is rejected)

## Required arguments
  <number>   PR number

## Required flags
  None strictly required, but the command is useless without a
  positional <number> and a --repo / -R (or global --repo).

## Optional flags
  --owner       Repository owner
  --repo / -R   Repository name or owner/repo

## Idempotency
PARTIAL. Closing an already-closed PR returns a 422 from the Gitee
API, so re-running after success is a hard error (not a silent
no-op). The action is, however, REVERSIBLE: call ` + "`" + `gitee pr reopen <N>` + "`" + `
to put the PR back in "open" state.

## Atomicity
Atomic at the API level. The PR's state either advances to "closed"
or the call fails — there is no partial state.

## Side effects
- The PR's state advances to "closed" (no longer mergeable)
- The head branch is NOT deleted
- The base branch is NOT modified (no merge happens)
- Subscribers (watchers, assignees, PR author) get a notification
- Webhooks subscribed to "PR closed" events fire
- Linked issues are NOT auto-closed (closing a PR is not the same
  as fixing an issue)
- ACTION: reversible via ` + "`" + `gitee pr reopen <N>` + "`" + `.
  No data is lost: title, body, comments, reviews, and the head
  branch's commits are all preserved.

## Post-conditions (on success)
- Stdout: ` + "`" + `PR #N closed` + "`" + `
- The PR is in "closed" state on the Gitee web UI
- Exit status 0

## Output (stdout, text mode)
  PR #42 closed

## Common errors
- 401 Unauthorized:  token missing or expired
- 403 Forbidden:     no write access to the repo
- 404 Not Found:     PR number doesn't exist
- 422 Unprocessable: PR is already closed or already merged
- Network timeouts:  bump ` + "`" + `--timeout` + "`" + ` and retry

## Race condition warning ⚠️
The shared mutable state is the PR's state field. Between your
` + "`" + `gitee pr view <N>` + "`" + ` check and this call, a human can merge the PR —
your close then fails with 422 (merged PRs cannot be closed), which is
the API protecting you; do NOT blind-retry a 422. Two agents closing
the same PR concurrently: one succeeds, the other gets 422 — treat
that 422 as "already closed", not as a retryable failure, and confirm
with a fresh ` + "`" + `gitee pr view <N>` + "`" + `. Retry after an ambiguous failure
(network timeout) is safe only if you have not seen the "PR #N closed"
success line; verify current state with ` + "`" + `gitee pr view <N>` + "`" + ` first.

## Next steps (orchestration hints)
After closing a PR, an agent typically wants to:
1. Optionally clean up the head branch:
   ` + "`" + `git push origin --delete <head>` + "`" + ` (manual, NOT done by close)
2. Comment with a reason so the author knows why:
   ` + "`" + `gitee pr comment <N> --body "Closing: superseded by #M"` + "`" + `
3. If you closed by mistake: ` + "`" + `gitee pr reopen <N>` + "`" + ` to undo
4. Optionally delete the head branch ref on the remote

## OAuth scope required
projects (write) — close is a state mutation.

## Rate limit
Gitee API v5: 1000 requests/hour. A single close is 1 call.

## Example (agent-friendly)
  # 1. Confirm the PR is open before closing
  STATE=$(gitee pr view 42 --repo owner/repo --output json | jq -r '.state')
  if [ "$STATE" = "open" ]; then
      gitee pr close 42 --repo owner/repo
  fi

  # 2. Leave a comment explaining why
  gitee pr comment 42 --repo owner/repo \
    --body "Closing as superseded by #99."

  # 3. (optional) Clean up the remote branch
  git push origin --delete feature/dark-mode

  # To undo a close: simply reopen
  gitee pr reopen 42 --repo owner/repo

## See also
- ` + "`" + `gitee pr view <N>` + "`" + `        — verify state before closing
- ` + "`" + `gitee pr reopen <N>` + "`" + `     — undo a close (the PR comes back)
- ` + "`" + `gitee pr merge <N>` + "`" + `      — alternative: merge instead of close
- ` + "`" + `gitee pr comment <N>` + "`" + `    — leave a reason for the author
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands with agent help
`
