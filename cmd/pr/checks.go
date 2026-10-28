package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var checksFlags struct {
	owner string
	repo  string
	id    int64
	limit int
}

var checksCmd = &cobra.Command{
	Use:   "checks <number>",
	Short: "Show the latest check-runs (CI status) for a pull request",
	Example: `  gitee pr checks 42 --repo owner/repo
  gitee pr checks 42 -R owner/repo --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(prChecksAgentHelp)
		}
		return nil
	},
	RunE: runChecks,
}

func init() {
	f := checksCmd.Flags()
	f.StringVar(&checksFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&checksFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.Int64VarP(&checksFlags.id, "id", "i", 0, "PR number (alternative to positional)")
	f.IntVarP(&checksFlags.limit, "limit", "L", 20, "Maximum number of check-runs to display")
}

func runChecks(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	prNum, err := utils.ParseInt64Arg(checksFlags.id, args, "PR")
	if err != nil {
		return err
	}
	if prNum <= 0 {
		return fmt.Errorf("PR number is required (positional or --id)")
	}

	owner, repoName, err := resolvePRRepo(checksFlags.owner, checksFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	// Step 1: fetch the PR to get the head ref (branch name). Gitee's
	// check-runs endpoint is keyed by commit ref, not PR number.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Get(ctx, api.PRsGet(owner, repoName, int(prNum)), &pr); err != nil {
		if api.IsNotFound(err) {
			return fmt.Errorf("PR #%d not found in %s/%s", prNum, owner, repoName)
		}
		return fmt.Errorf("fetching PR: %w", err)
	}
	if pr.Head == nil || pr.Head.Ref == "" {
		return fmt.Errorf("PR #%d has no head ref (cannot resolve check-runs)", prNum)
	}
	headRef := pr.Head.Ref

	// Step 2: fetch the latest check-runs for that ref, scoped to this PR.
	path := api.PRsChecks(owner, repoName, headRef)
	q := url.Values{}
	q.Set("pull_request_id", fmt.Sprintf("%d", prNum))
	q.Set("filter", "latest")
	q.Set("per_page", fmt.Sprintf("%d", checksFlags.limit))
	fullPath := path + "?" + q.Encode()

	var checkRuns []types.CheckRun
	if err := client.Get(ctx, fullPath, &checkRuns); err != nil {
		return fmt.Errorf("fetching check-runs: %w", err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(checkRuns, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(checkRuns) == 0 {
		fmt.Printf("No check runs configured for PR #%d (head=%s)\n", prNum, headRef)
		fmt.Println("(To set up checks, configure a Gitee webhook or Gitee Go pipeline in the repo settings.)")
		return nil
	}

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Printf("Check runs for PR #%d (head=%s) in %s/%s:\n\n", prNum, cyan(headRef), owner, repoName)
	w := output.NewTableWriter()
	fmt.Fprintln(w, "NAME\tSTATUS\tCONCLUSION\tCOMPLETED")
	for _, cr := range checkRuns {
		status := cr.Status
		conclusion := cr.Conclusion
		// Color-code the status / conclusion
		switch status {
		case "completed":
			status = dim("completed")
		case "in_progress":
			status = yellow("in_progress")
		case "queued":
			status = cyan("queued")
		}
		switch conclusion {
		case "success":
			conclusion = green("success")
		case "failure":
			conclusion = red("failure")
		case "neutral", "skipped":
			conclusion = dim(conclusion)
		}
		completed := ""
		if !cr.CompletedAt.IsZero() {
			completed = cr.CompletedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", cr.Name, status, conclusion, completed)
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d check-run(s)\n", len(checkRuns))
	return nil
}

const prChecksAgentHelp = `# gitee pr checks — Agent Help

## Purpose
Show the latest CI check-runs (Gitee v5 "Checks" API) for a pull
request. Use this to quickly see whether the PR is green before
merging, or to surface a specific failed check to the author.

Gitee v5 surfaces CI status via GitHub-style "Checks" at
` + "`" + `/commits/{ref}/check-runs` + "`" + ` (not via the older "commit statuses"
or "combined status" endpoints). Repos using Gitee Go pipelines
populate this surface via webhooks.

For the dedicated ` + "`" + `gitee pipeline` + "`" + ` command group (a future
milestone that wraps Gitee Go's full pipeline surface), see
` + "`" + `common-todos.md` + "`" + ` P3. This ` + "`" + `checks` + "`" + ` verb is a focused
"is it green?" view, not a full pipeline manager.

## Pre-conditions
- Authenticated with read access
- <number> is an existing PR id
- The repo has at least one configured check-run source (Gitee Go
  pipeline, third-party webhook, etc.) — otherwise the response is
  an empty array

## Required arguments
  <number>   PR number

## Optional flags
  --repo / -R    owner/repo
  --id / -i      Alternative to positional
  --limit / -L   Maximum check-runs to display (default 20, max 100)
  --output       "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic (two GETs, no writes).

## Side effects
None. Pure GETs.

## Post-conditions (on success)
- Stdout contains a color-coded table of check-runs
  (NAME / STATUS / CONCLUSION / COMPLETED)
- The PR's state is unchanged
- Empty result is a real signal: "no checks configured", not an error

## Output (stdout, table mode)
  Check runs for PR #42 (head=feature/x) in owner/repo:

  NAME              STATUS        CONCLUSION   COMPLETED
  CI / lint         completed     success      2026-01-15 09:23
  CI / test         in_progress   -            -
  CodeQL            completed     failure      2026-01-15 09:25

  Total: 3 check-run(s)

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    no read access
- 404 Not Found:    PR number doesn't exist (the first GET fails)
- 422 Unprocessable: --limit is out of range

## Race condition warning ⚠️
Gitee updates check-run status asynchronously. A push may register
a "queued" check-run that takes seconds to minutes to start and
finish. If you chain ` + "`" + `gitee pr checks` + "`" + ` immediately after
pushing commits, expect to see ` + "`" + `queued` + "`" + ` or ` + "`" + `in_progress` + "`" + ` rather
than ` + "`" + `success` + "`" + `. Wait 5-30 seconds before reading.

## Next steps (orchestration hints)
1. Run: ` + "`" + `gitee pr checks 42 -R owner/repo` + "`" + `
2. If any CONCLUSION is ` + "`" + `failure` + "`" + `, fetch details with the URL from
   the check-run's ` + "`" + `html_url` + "`" + ` field (visible with ` + "`" + `--output json` + "`" + `).
3. Once all checks are ` + "`" + `success` + "`" + ` and the PR is ready (not draft),
   merge with ` + "`" + `gitee pr merge 42 -R owner/repo` + "`" + `.

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 req/hour. Each call = 2 requests (PR fetch +
check-runs fetch).

## Example (closing the loop)
  # 1. Read the JSON to see all fields (including html_url for failure drilldown)
  gitee pr checks 42 -R owner/repo --output json | jq '.[] | select(.conclusion == "failure") | {name, html_url}'

  # 2. Open the failing check in a browser
  URL=$(gitee pr checks 42 -R owner/repo --output json | jq -r '.[] | select(.conclusion == "failure") | .html_url' | head -1)
  [ -n "$URL" ] && open "$URL"

## See also
- ` + "`" + `gitee pr view` + "`" + `   — PR metadata (no CI status)
- ` + "`" + `gitee pr merge` + "`" + `  — merge when checks pass
- ` + "`" + `gitee pr ready` + "`" + `  — mark draft → ready
- ` + "`" + `common-todos.md` + "`" + `   — P3 pipeline group (future full Gitee Go surface)
`
