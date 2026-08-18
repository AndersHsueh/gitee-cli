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

var reopenFlags struct {
	owner string
	repo  string
}

var reopenCmd = &cobra.Command{
	Use:     "reopen <number>",
	Short:   "Reopen a pull request",
	Example: `  gitee pr reopen 42 --repo owner/repo`,
	// PreRunE fires BEFORE the cobra arg check, so an agent can request
	// --agent-help even when <number> is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prReopenAgentHelp)
		}
		return nil
	},
	RunE: runReopen,
}

func init() {
	f := reopenCmd.Flags()
	f.StringVar(&reopenFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&reopenFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runReopen(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(reopenFlags.owner, reopenFlags.repo)
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

	path := api.PRsGet(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	req := map[string]string{"state": "open"}
	var pr types.PullRequest
	if err := client.Patch(ctx, path, req, &pr); err != nil {
		return fmt.Errorf("reopening PR: %w", err)
	}

	fmt.Printf("PR #%d reopened\n", prNum)
	return nil
}

// prReopenAgentHelp is the agent-targeted help for `gitee pr reopen`.
// Re-opens a previously closed PR. Distinct from `gitee pr merge`:
// reopen does NOT merge, it just puts the PR back in "open" state.
const prReopenAgentHelp = `# gitee pr reopen — Agent Help

## Purpose
Reopen a pull request that was previously closed WITHOUT merging
(the inverse of ` + "`" + `gitee pr close` + "`" + `). The PR's title, body,
comments, reviews, and head branch ref are all preserved; reopen
just flips the state back to "open" so the PR is mergeable again.
Use this to recover from an accidental close, or to resume work on
a previously shelved PR.

NOTE: Gitee v5 does NOT allow reopening a MERGED PR. To revert a
merged PR, open a new "revert" PR via the web UI.

## Pre-conditions
- Authenticated (token in config, env, or --token)
- --repo (or --owner) points to an existing, accessible repository
- <number> is a valid PR id
- Token has write access to the repo
- The PR is currently in "closed" state (reopen an open PR = 422;
  reopen a merged PR = not supported by the API)

## Required arguments
  <number>   PR number

## Required flags
  None strictly required, but the command is useless without a
  positional <number> and a --repo / -R (or global --repo).

## Optional flags
  --owner       Repository owner
  --repo / -R   Repository name or owner/repo

## Idempotency
PARTIAL. Reopening an already-open PR returns 422. Reopening a
closed PR is the success path. Safe to retry only if you have not
observed a success line in the previous run's stdout.

## Atomicity
Atomic at the API level. The PR's state either advances to "open"
or the call fails — there is no partial state.

## Side effects
- The PR's state advances to "open" (mergeable again)
- The head branch is NOT modified (no commits added or removed)
- Subscribers (watchers, assignees, PR author) get a notification
- Webhooks subscribed to "PR reopened" events fire
- Prior reviews and comments are preserved as-is
- If the head branch was force-pushed or rebased after the close,
  the diff against the base may now be different; inspect with
  ` + "`" + `gitee pr view <N>` + "`" + ` before merging

## Post-conditions (on success)
- Stdout: ` + "`" + `PR #N reopened` + "`" + `
- The PR is in "open" state on the Gitee web UI
- Exit status 0

## Output (stdout, text mode)
  PR #42 reopened

## Common errors
- 401 Unauthorized:  token missing or expired
- 403 Forbidden:     no write access to the repo
- 404 Not Found:     PR number doesn't exist
- 422 Unprocessable: PR is already open, OR the PR was merged and
                     cannot be reopened (use a revert PR instead)
- Network timeouts:  bump ` + "`" + `--timeout` + "`" + ` and retry

## Race condition warning ⚠️
The shared mutable state is the PR's state field plus its head branch.
Between your state check and this call, another actor can reopen first
(your call then 422s — treat that as "already open" after a fresh view
confirms). More subtly: while the PR was closed, the head branch may
have been force-pushed or rebased — reopen still succeeds, but the PR
now contains different code than what earlier reviewers approved.
After reopen, ALWAYS re-inspect with ` + "`" + `gitee pr diff <N>` + "`" + ` and
re-run ` + "`" + `gitee pr checks <N>` + "`" + ` before merging. Retry after an
ambiguous failure (timeout) is safe only after ` + "`" + `gitee pr view <N>` + "`" + `
shows the state is still "closed".

## Next steps (orchestration hints)
After reopening a PR, an agent typically wants to:
1. Re-check the current state vs. base:
   ` + "`" + `gitee pr view <N>` + "`" + ` and ` + "`" + `gitee pr diff <N>` + "`" + `
2. Re-run CI to make sure it still passes:
   ` + "`" + `gitee pr checks <N>` + "`" + `
3. Request a fresh review if the diff changed:
   ` + "`" + `gitee pr comment <N> --body "Reopening for review"` + "`" + `
4. Merge when ready: ` + "`" + `gitee pr merge <N>` + "`" + `

## OAuth scope required
projects (write) — reopen is a state mutation.

## Rate limit
Gitee API v5: 1000 requests/hour. A single reopen is 1 call.

## Example (agent-friendly)
  # 1. Confirm the PR is closed (and not merged) before reopening
  PR_JSON=$(gitee pr view 42 --repo owner/repo --output json)
  STATE=$(echo "$PR_JSON" | jq -r '.state')
  MERGED=$(echo "$PR_JSON" | jq -r '.merged')

  if [ "$STATE" = "closed" ] && [ "$MERGED" = "false" ]; then
      gitee pr reopen 42 --repo owner/repo
  else
      echo "Cannot reopen: state=$STATE merged=$MERGED" >&2
      exit 1
  fi

  # 2. Verify the diff is still sane
  gitee pr diff 42 --repo owner/repo | head -50

  # 3. Run a comment to ping reviewers
  gitee pr comment 42 --repo owner/repo \
    --body "Reopening this for further review."

## See also
- ` + "`" + `gitee pr view <N>` + "`" + `        — verify state before reopening
- ` + "`" + `gitee pr close <N>` + "`" + `       — the inverse (close without merging)
- ` + "`" + `gitee pr diff <N>` + "`" + `       — re-inspect the diff after reopen
- ` + "`" + `gitee pr merge <N>` + "`" + `      — next step after reopen
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands with agent help
`
