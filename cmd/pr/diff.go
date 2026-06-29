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

var diffFlags struct {
	owner string
	repo  string
}

var diffCmd = &cobra.Command{
	Use:     "diff <number>",
	Short:   "Show pull request diff",
	Example: `  gitee pr diff 42 --repo owner/repo`,
	// PreRunE fires BEFORE the cobra arg check, so an agent can request
	// --agent-help even when <number> is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prDiffAgentHelp)
		}
		return nil
	},
	RunE: runDiff,
}

func init() {
	f := diffCmd.Flags()
	f.StringVar(&diffFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&diffFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runDiff(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(diffFlags.owner, diffFlags.repo)
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Get(ctx, api.PRsGet(owner, repoName, prNum), &pr); err != nil {
		return fmt.Errorf("getting PR: %w", err)
	}

	if pr.DiffURL == "" {
		return fmt.Errorf("PR #%d has no diff_url", prNum)
	}

	body, _, err := client.GetRaw(ctx, pr.DiffURL)
	if err != nil {
		return fmt.Errorf("fetching diff: %w", err)
	}

	_, err = cobraCmd.OutOrStdout().Write(body)
	return err
}

// prDiffAgentHelp is the agent-targeted help for `gitee pr diff`.
// Streams the raw unified diff of a PR to stdout. Read-only, but
// potentially large — pipe through `less` or save to a file.
const prDiffAgentHelp = `# gitee pr diff — Agent Help

## Purpose
Stream the raw unified diff of a pull request to stdout. Use this
to inspect exactly what code is changing in a PR, before approving
or merging. The output is the same format ` + "`" + `git diff` + "`" + ` would
produce for the same change, so it can be piped to ` + "`" + `less` + "`" + `,
` + "`" + `git apply` + "`" + `, code-review tools, or saved to a ` + "`" + `.patch` + "`" + ` file.

## Pre-conditions
- Authenticated (token in config, env, or --token)
- --repo (or --owner) points to an existing, accessible repository
- <number> is a valid PR id
- Read access to the repo
- The PR has a non-empty diff (i.e. is not a no-op PR with no
  file changes since base)

## Required arguments
  <number>   PR number

## Required flags
  None strictly required, but the command is useless without a
  positional <number> and a --repo / -R (or global --repo).

## Optional flags
  --owner       Repository owner
  --repo / -R   Repository name or owner/repo

## Idempotency
YES. Read-only. Calling it twice yields the same diff (unless the
PR is force-pushed in between). Safe to retry on transient errors.

## Atomicity
A single GET to the PR's diff_url. The diff either streams to
stdout or the call fails — there is no partial state on the wire.

## Side effects
- None on the Gitee repo. No notifications, no webhooks fire.
- The token's read access is recorded in Gitee's access log.
- Rate-limit counter advances by 1.
- Output may be large (a PR touching many files can produce
  megabytes of diff). Be polite about piping to a pager.

## Post-conditions (on success)
- Stdout contains the raw unified diff (git-style)
- Exit status 0
- No file or network state mutated

## Output (stdout, unified-diff format)
  diff --git a/src/foo.go b/src/foo.go
  index 1234567..89abcde 100644
  --- a/src/foo.go
  +++ b/src/foo.go
  @@ -10,7 +10,7 @@ package foo
  ...hunks...

  The output is NOT a Gitee-rendered diff. It is the raw diff that
  ` + "`" + `git apply` + "`" + ` would consume. To get a colored, paginated view
  in a TTY, pipe through ` + "`" + `less -R` + "`" + ` or ` + "`" + `delta` + "`" + `.

## Common errors
- 401 Unauthorized:   token missing or expired
- 403 Forbidden:      no read access to the repo
- 404 Not Found:      PR number doesn't exist
- 422 Unprocessable:  <number> was not an integer (caught client-side)
- "PR #N has no diff_url": the PR has no diff (rare; usually a
  brand-new PR with no commits yet)
- Network timeouts:   bump ` + "`" + `--timeout` + "`" + ` and retry

## Next steps (orchestration hints)
After reading the diff, an agent typically wants to:
1. Approve if it looks good: ` + "`" + `gitee pr review <N> --approve` + "`" + `
2. Request changes with a body: ` + "`" + `gitee pr review <N> --request-changes -b "..."` + "`" + `
3. Comment with a question: ` + "`" + `gitee pr comment <N> --body "..."` + "`" + `
4. Save the diff as a file for offline inspection:
   ` + "`" + `gitee pr diff <N> --repo owner/repo > pr-42.patch` + "`" + `
5. View existing comments to see what was already discussed:
   ` + "`" + `gitee pr comments <N>` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour. A single diff fetch is 1 call.

## Example (agent-friendly)
  # 1. Save the diff to a file
  gitee pr diff 42 --repo owner/repo > pr-42.patch

  # 2. Quick stats: how many files / lines changed
  grep -c '^diff --git' pr-42.patch
  grep -cE '^\+' pr-42.patch

  # 3. Read just the file headers
  grep '^diff --git' pr-42.patch

  # 4. Pipe to delta for a colored view in a TTY
  gitee pr diff 42 --repo owner/repo | delta

## See also
- ` + "`" + `gitee pr view <N>` + "`" + `         — metadata + stats summary
- ` + "`" + `gitee pr review <N>` + "`" + `     — approve / request changes / comment
- ` + "`" + `gitee pr checkout <N>` + "`" + `  — check out the branch locally
- ` + "`" + `gitee pr comments <N>` + "`" + `   — conversation thread
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands with agent help
`
