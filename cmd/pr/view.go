package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var viewFlags struct {
	owner string
	repo  string
}

var viewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View pull request details",
	Example: `  gitee pr view 42 --repo owner/repo
  gitee pr view 42`,
	// PreRunE fires BEFORE the cobra arg check, so an agent can request
	// --agent-help even when <number> is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prViewAgentHelp)
		}
		return nil
	},
	RunE: runView,
}

func init() {
	f := viewCmd.Flags()
	f.StringVar(&viewFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&viewFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(viewFlags.owner, viewFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: must be an integer", args[0])
	}

	path := api.PRsGet(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Get(ctx, path, &pr); err != nil {
		return fmt.Errorf("viewing PR: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(pr, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	w := output.NewTableWriter()
	state := pr.State
	if pr.Merged {
		state = "merged"
	}
	fmt.Fprintf(w, "Title:\t%s\n", pr.Title)
	fmt.Fprintf(w, "State:\t%s\n", state)
	fmt.Fprintf(w, "Author:\t%s\n", pr.User.Login)
	baseRef := "-"
	if pr.Base != nil {
		baseRef = pr.Base.Ref
	}
	headRef := "-"
	if pr.Head != nil {
		headRef = pr.Head.Ref
	}
	fmt.Fprintf(w, "Base:\t%s\n", baseRef)
	fmt.Fprintf(w, "Head:\t%s\n", headRef)
	fmt.Fprintf(w, "URL:\t%s\n", pr.HTMLURL)
	if pr.Body != "" {
		fmt.Fprintf(w, "\n%s\n", pr.Body)
	}
	if pr.Comments > 0 || pr.Commits > 0 || pr.Additions > 0 {
		fmt.Fprintf(w, "\nStats:\t+%d/-%d in %d commits with %d comments\n",
			pr.Additions, pr.Deletions, pr.Commits, pr.Comments)
	}
	return w.Flush()
}

// prViewAgentHelp is the agent-targeted help for `gitee pr view`.
// Read-only metadata fetch — the next step in any PR workflow.
const prViewAgentHelp = `# gitee pr view — Agent Help

## Purpose
Fetch and print the full metadata of a single pull request: title,
state, author, base/head refs, URL, body, and stats (additions,
deletions, commits, comments). Use this after ` + "`" + `gitee pr list` + "`" + `
to drill into one specific PR, or as the first step of a merge/review
workflow.

## Pre-conditions
- Authenticated (token in config, env, or --token)
- --repo (or --owner) points to an existing, accessible repository
- <number> is a valid integer referencing an existing PR
- Read access to the repo

## Required arguments
  <number>   PR number (e.g. 42)

## Required flags
  None strictly required, but the command is useless without a
  positional <number>. --repo / -R (or global --repo) is needed to
  disambiguate the target repo.

## Optional flags
  --owner       Repository owner (overrides the owner part of --repo)
  --repo / -R   Repository name or owner/repo
  --output json Emit a JSON object of the PR instead of a key/value table

## Idempotency
YES. Read-only. Calling it twice yields the same metadata (unless the
PR is updated in between). Safe to retry on transient errors.

## Atomicity
A single GET to the Gitee API. Either the PR metadata is returned or
the call fails — there is no partial state.

## Side effects
- None on the Gitee repo. No notifications, no webhooks fire.
- The token's read access is recorded in Gitee's access log.
- Rate-limit counter advances by 1.

## Post-conditions (on success)
- Stdout contains a key/value table (default) or a JSON object
  (with --output json) describing the PR
- Exit status 0
- No file or network state mutated

## Output (stdout, table mode)
  Title:   Add dark mode
  State:   open
  Author:  alice
  Base:    main
  Head:    dark
  URL:     https://gitee.com/owner/repo/pulls/42

  The body (if any) follows the header block as Markdown text.
  Stats appear when additions / commits / comments > 0:
    Stats:   +120/-30 in 3 commits with 2 comments

## Output (stdout, json mode)
  {
    "number": 42,
    "title": "Add dark mode",
    "state": "open",
    "merged": false,
    "user": { "login": "alice", "...": "..." },
    "head": { "ref": "dark", "...": "..." },
    "base": { "ref": "main", "...": "..." },
    "html_url": "https://gitee.com/owner/repo/pulls/42",
    "body": "...",
    "additions": 120,
    "deletions": 30,
    "commits": 3,
    "comments": 2,
    "...": "..."
  }

## Common errors
- 401 Unauthorized:  token missing or expired  →  ` + "`" + `gitee auth login` + "`" + `
- 403 Forbidden:     no read access to the repo
- 404 Not Found:     PR number does not exist (or repo is wrong)
- 422 Unprocessable: <number> was not an integer (caught client-side)
- Network timeouts:  bump ` + "`" + `--timeout` + "`" + ` (global flag) and retry

## Race condition warning ⚠️
The returned metadata is a point-in-time snapshot. The shared mutable
state is the PR itself: state, merged, mergeable, head SHA, and stats
all change under you. Two agents acting on the same cached view output
can both attempt ` + "`" + `gitee pr merge <N>` + "`" + ` — one wins, the other gets 422 —
or approve-then-merge a PR whose head was force-pushed after the view.
If your next command is a state change (merge / close / reopen /
review), re-run ` + "`" + `gitee pr view <N> --output json` + "`" + ` immediately before
it and verify .state == "open" and .merged == false on the FRESH
output, not on a cached copy. This command is read-only and safe to
retry.

## Next steps (orchestration hints)
After viewing a PR, an agent typically wants to:
1. Check CI status: ` + "`" + `gitee pr checks <N>` + "`" + `
2. Read existing comments: ` + "`" + `gitee pr comments <N>` + "`" + `
3. Check out the branch locally: ` + "`" + `gitee pr checkout <N>` + "`" + ` (or ` + "`" + `gitee co <N>` + "`" + `)
4. View the diff: ` + "`" + `gitee pr diff <N>` + "`" + `
5. Approve / request changes / comment: ` + "`" + `gitee pr review <N>` + "`" + `
6. Merge (only if checks pass and reviews approve): ` + "`" + `gitee pr merge <N>` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour. A single view is 1 call.

## Example (agent-friendly)
  # 1. View a PR as JSON for downstream jq processing
  gitee pr view 42 --repo owner/repo --output json > pr42.json

  # 2. Pull out the head branch name
  HEAD=$(jq -r '.head.ref' pr42.json)

  # 3. Decide next step: check CI then review
  gitee pr checks 42 --repo owner/repo
  gitee pr review 42 --repo owner/repo --approve --body "LGTM"

## See also
- ` + "`" + `gitee pr list` + "`" + `         — enumerate PRs
- ` + "`" + `gitee pr diff <N>` + "`" + `     — show the file-level diff
- ` + "`" + `gitee pr comments <N>` + "`" + ` — list conversation comments
- ` + "`" + `gitee pr checkout <N>` + "`" + ` — fetch the PR branch locally
- ` + "`" + `gitee pr review <N>` + "`" + `  — approve / request changes / comment
- ` + "`" + `gitee pr merge <N>` + "`" + `    — merge the PR
- ` + "`" + `gitee --agent-help` + "`" + `    — overview of all commands with agent help
`
