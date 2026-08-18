package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var mergeFlags struct {
	owner     string
	repo      string
	squash    bool
	squashMsg string
}

var mergeCmd = &cobra.Command{
	Use:   "merge <number>",
	Short: "Merge a pull request",
	Example: `  gitee pr merge 42 --repo owner/repo
  gitee pr merge 42 --repo owner/repo --squash`,
	// PreRunE fires BEFORE the cobra arg check, so an agent can request
	// --agent-help even when <number> is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prMergeAgentHelp)
		}
		return nil
	},
	RunE: runMerge,
}

func init() {
	f := mergeCmd.Flags()
	f.StringVar(&mergeFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&mergeFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.BoolVar(&mergeFlags.squash, "squash", false, "Squash commits when merging")
	f.StringVar(&mergeFlags.squashMsg, "squash-message", "", "Commit message for squash merge")
}

func runMerge(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(mergeFlags.owner, mergeFlags.repo)
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

	req := types.PRMergeRequest{
		Squash:        mergeFlags.squash,
		SquashMessage: mergeFlags.squashMsg,
	}

	path := api.PRsMerge(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var result struct {
		Merged bool `json:"merged"`
	}
	if err := client.Put(ctx, path, req, &result); err != nil {
		return fmt.Errorf("merging PR: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	if result.Merged {
		fmt.Printf("%s PR #%d merged\n", green("✓"), prNum)
	} else {
		fmt.Printf("PR #%d merge status: %v\n", prNum, result.Merged)
	}
	return nil
}

// prMergeAgentHelp is the agent-targeted help for `gitee pr merge`.
// This is a write command: it mutates the target branch. The merge
// itself is reversible on Gitee via a "revert PR" (Gitee v5 does not
// expose a "revert merge" endpoint via this CLI), but the squash flag
// is destructive to individual commit history.
const prMergeAgentHelp = `# gitee pr merge — Agent Help

## Purpose
Merge a pull request into its base branch on Gitee. By default the
merge preserves all PR commits; pass ` + "`" + `--squash` + "`" + ` to collapse
them into a single commit (optionally with a custom message). Use
this to ship reviewed + approved PRs.

## Pre-conditions
- Authenticated (token in config, env, or --token)
- --repo (or --owner) points to an existing, accessible repository
- <number> is a valid PR id with state == "open" and NOT already merged
- All required CI checks have passed (the Gitee API may reject the
  merge with 422 if repo policy requires it)
- Token has write access to the target repo
- The PR's head branch is mergeable (no conflicts with the base)

## Required arguments
  <number>   PR number

## Required flags
  None strictly required, but the command is useless without a
  positional <number> and a --repo / -R (or global --repo).

## Optional flags
  --owner              Repository owner
  --repo / -R          Repository name or owner/repo
  --squash             Collapse all PR commits into a single commit on merge
  --squash-message     Custom commit message for the squash commit
                       (only meaningful with --squash)

## Idempotency
PARTIAL. Merging an already-merged PR fails with 422 from the Gitee
API, so re-running after a successful merge is a no-op (no duplicate
merge, no error message that looks like success). Re-running on an
already-closed PR also fails. Safe to retry ONLY if you have not
observed a success line in the previous run's stdout.

## Atomicity
Atomic at the API level. The merge either happens or it doesn't —
there is no partial state where the PR is half-merged. However,
--squash with --squash-message rewrites the head branch's commit
history on the base: the original PR commits may be lost from the
target branch's history (they remain reachable from the PR's ref
until the PR is deleted, but they are not on the base branch).

## Side effects
- The PR's state advances to "merged" (terminal state)
- The base branch advances: a new commit (or, with --squash, exactly
  one new commit) is added to the base branch
- The head branch is NOT automatically deleted; do that with
  ` + "`" + `git push origin --delete <head>` + "`" + ` if desired
- Subscribers (watchers, assignees, mentionees) get an in-app /
  email notification
- Webhooks subscribed to "Push" and "Pull Request" events fire
- --squash is DESTRUCTIVE to the per-commit history: individual PR
  commits are no longer reachable from the base branch tip
- The merge is REVERSIBLE on Gitee via a manual "revert PR" in the
  web UI, but ` + "`" + `gitee-cli` + "`" + ` does not currently expose a revert
  command — plan accordingly and use --squash only when you're sure

## Post-conditions (on success)
- Stdout: ` + "`" + `✓ PR #N merged` + "`" + `
- The PR is in "merged" state on the Gitee web UI
- The base branch tip includes the new merge commit (or squash commit)
- Exit status 0

## Output (stdout, text mode)
  ✓ PR #42 merged

  When the API returns ` + "`" + `merged: false` + "`" + ` (rare):
  PR #42 merge status: false

## Common errors
- 401 Unauthorized:    token missing or expired
- 403 Forbidden:       no write access to the repo (or branch is protected)
- 404 Not Found:       PR number doesn't exist
- 422 Unprocessable:   PR is closed, already merged, has conflicts, or
                       required CI checks have not passed
- Network timeouts:    bump ` + "`" + `--timeout` + "`" + ` (global flag) and retry

## Race condition warning ⚠️
Mergeability is a moving target. The shared mutable state is the PR's
state plus BOTH branch tips: between your ` + "`" + `gitee pr checks <N>` + "`" + ` /
` + "`" + `gitee pr view <N>` + "`" + ` and this call, the head can be force-pushed (you
would merge code nobody reviewed), the base can advance (new
conflicts), CI can flip red, or a human can merge first (you get 422).
The API re-validates at merge time, so a 422 is the protection working
— do NOT blind-retry it; re-check .state / .merged / .mergeable with a
fresh ` + "`" + `gitee pr view <N> --output json` + "`" + `. After an ambiguous failure
(timeout), verify with view whether the merge actually landed
(.merged == true) before re-running — retrying a landed merge returns
422, which is harmless but masks the real outcome. Keep the
check → merge gap as short as possible; in multi-agent repos,
serialize merges through one agent.

## Next steps (orchestration hints)
After a successful merge, an agent typically wants to:
1. Delete the head branch: ` + "`" + `git push origin --delete <head>` + "`" + `
2. Close any "Closes #N" issues if the PR fixed them and they
   weren't auto-closed (Gitee auto-closes issue refs in PR body
   for ` + "`" + `fix:` + "`" + `, ` + "`" + `fixes:` + "`" + `, ` + "`" + `closes:` + "`" + `, ` + "`" + `resolves:` + "`" + ` keywords)
3. Tag a release if this is a release branch: ` + "`" + `gitee release create` + "`" + `
4. Re-list PRs to confirm the merge: ` + "`" + `gitee pr list --state merged` + "`" + `

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour. A single merge is 1 call.

## Example (agent-friendly)
  # 1. Verify the PR is open and mergeable first
  gitee pr view 42 --repo owner/repo --output json | jq -r '.state, .mergeable'

  # 2. Run a final CI check
  gitee pr checks 42 --repo owner/repo

  # 3. Merge (non-squash, preserves history)
  gitee pr merge 42 --repo owner/repo

  # Or, squash with a custom message:
  gitee pr merge 42 --repo owner/repo --squash \
    --squash-message "feat: add dark mode (#42)"

  # 4. (optional) Clean up the head branch
  git push origin --delete dark

## See also
- ` + "`" + `gitee pr view <N>` + "`" + `         — read PR metadata (verify state first)
- ` + "`" + `gitee pr checks <N>` + "`" + `      — confirm CI is green
- ` + "`" + `gitee pr review <N>` + "`" + `     — approve before merging
- ` + "`" + `gitee pr close <N>` + "`" + `      — close WITHOUT merging (destructive-ish)
- ` + "`" + `gitee pr list --state merged` + "`" + ` — confirm the merge
- ` + "`" + `gitee --agent-help` + "`" + `      — overview of all commands with agent help
`
