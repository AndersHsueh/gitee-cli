package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var listFlags struct {
	state     string
	labels    string
	owner     string
	assignee  string
	milestone string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues in a repository",
	Example: `  gitee issue list --repo owner/repo
  gitee issue list --repo owner/repo --state closed --labels bug
  gitee issue list --repo owner/repo --paginate          # fetch ALL open issues
  gitee issue list --repo owner/repo --paginate --limit 0  # same, explicit`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(issueListAgentHelp)
		}
		return nil
	},
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.StringVar(&listFlags.state, "state", "open", "Filter by state: open|closed|all")
	f.StringVar(&listFlags.labels, "labels", "", "Filter by labels (comma-separated)")
	f.StringVar(&listFlags.owner, "owner", "", "Repository owner")
	f.StringVar(&listFlags.assignee, "assignee", "", "Filter by assignee username")
	f.StringVar(&listFlags.milestone, "milestone", "", "Filter by milestone")
	// --limit / --per-page / --paginate are global persistent flags.
	// See cmd/root.go for their definitions. --limit here is a TOTAL cap
	// across pages when --paginate is set (was previously a per-page size,
	// which was a bug — see v0.4.1 changelog).
}

func runList(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveRepo(listFlags.owner, cmd.GlobalRepo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	// Build base query (without page/per_page — FetchAllPages handles those)
	qv := url.Values{}
	qv.Set("state", listFlags.state)
	if listFlags.labels != "" {
		qv.Set("labels", listFlags.labels)
	}
	if listFlags.assignee != "" {
		qv.Set("assignee", listFlags.assignee)
	}
	if listFlags.milestone != "" {
		qv.Set("milestone", listFlags.milestone)
	}
	basePath := api.IssuesList(owner, repoName) + "?" + qv.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// FetchListOrAll handles perPage clamp (1-100), --paginate dispatch, and
	// treats a 404 as an empty result (previously issue list errored on 404
	// while pr list returned empty — now consistent).
	issues, err := api.FetchListOrAll[types.Issue](ctx, client, basePath,
		cmd.GlobalPaginate, cmd.GlobalPerPage, cmd.GlobalLimit, cmd.GlobalPage,
		"listing issues")
	if err != nil {
		return err
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(issues, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(issues) == 0 {
		fmt.Printf("No %s issues found in %s/%s\n", listFlags.state, owner, repoName)
		return nil
	}

	w := output.NewTableWriter()
	fmt.Fprintln(w, "#\tTITLE\tSTATE\tLABELS\tUPDATED")
	for _, i := range issues {
		lbls := make([]string, 0, len(i.Labels))
		for _, l := range i.Labels {
			lbls = append(lbls, l.Name)
		}
		labelStr := strings.Join(lbls, ",")
		if labelStr == "" {
			labelStr = "-"
		}
		title := utils.Truncate(i.Title, 60)
		updated := i.UpdatedAt.Format("2006-01-02")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", i.Number, title, i.State, labelStr, updated)
	}
	_ = w.Flush()
	return nil
}

// issueListAgentHelp is the agent-targeted help for `gitee issue list`.
// It is intentionally long, structured, and includes machine-relevant
// details (idempotency, side effects, orchestration hints) that the
// default --help omits because they're noise for humans.
//
// Maintainer note: keep this in sync with the actual command behavior.
// When flags change, update the relevant section here.
const issueListAgentHelp = `# gitee issue list — Agent Help

## Purpose
List issues in a Gitee repository, with optional filters. Use this to
enumerate issues by state, labels, assignee, or milestone — typically as
a pre-step before reading a specific issue (` + "`" + `gitee issue view` + "`" + `)
or batch-editing them.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Repository exists: --repo (or --owner + --repo) points to an accessible repo
- Read access: token must have read permission on the target repo
  (private repos require projects or higher)

## Optional flags
  --state           Filter by state: open (default) | closed | all
  --labels          Filter by labels (comma-separated, all must match)
  --owner           Repository owner (only needed if --repo is just a name)
  --assignee        Filter by assignee username
  --milestone       Filter by milestone number
  --repo / -R       owner/repo format (falls back to global --repo, then git remote)
  --output / -o     text (default) | json — when json, the issue list is dumped
                    as a JSON array of Issue objects (full struct, not a table)
  --page            Page number for the single-page path (default 1)
  --per-page        Per-page size (max 100, default 30)
  --paginate        Walk all pages up to --limit (recommended for agents)
  --limit           Total cap when --paginate is set; 0 = no cap

## Idempotency
Idempotent (read-only). Re-running this command does not change any
remote state. Repeated calls may return different results if the
underlying issues have been edited between calls.

## Atomicity
Atomic at the response level: a single --paginate run either returns
the full set of pages or fails on the first page. There is no
partial-success state where some pages are returned and others errored.

## Side effects
None. This is a pure read. No notifications, no webhooks, no
mutations on the repo.

## Post-conditions (on success)
- Stdout contains the issue list (table or JSON) — see Output sections
- No state change on the repo
- The set of issues returned reflects the API at the time of the call
  (subject to Gitee's eventual consistency for very recent writes)

## Output (stdout, table mode)
  #    TITLE                              STATE    LABELS              UPDATED
  42   Bug: crash on startup              open     bug,priority/high   2026-06-28
  41   Feature: dark mode                  open     enhancement         2026-06-27
  ...

  Empty result:
  No open issues found in owner/repo

  Columns: #, TITLE (truncated to 60 chars), STATE, LABELS (- if none), UPDATED (date)

## Output (stdout, json mode)
  -o json prints the full types.Issue struct array, one issue per element,
    with all fields (number, title, state, labels, assignees, body, html_url,
    created_at, updated_at, etc.). Suitable for downstream pipelines.

## Parsing hints
  # Extract the numbers of all open issues labeled "bug":
  gitee issue list --repo owner/repo --state open --labels bug -o json \
    | jq -r '.[].number'

  # Iterate over each issue and view it:
  for n in $(gitee issue list --repo owner/repo -o json | jq -r '.[].number'); do
    gitee issue view "$n" --repo owner/repo
  done

## Common errors
- 401 Unauthorized:  token expired or invalid    →  gitee auth login
- 403 Forbidden:     token lacks read scope      →  check token permissions
- 404 Not Found:     repo doesn't exist          →  verify owner/repo spelling
- 422 Unprocessable: bad filter combination      →  check --state, --labels values

## Race condition warning ⚠️
A follow-up ` + "`" + `gitee issue list` + "`" + ` may NOT return an issue that was
just created via ` + "`" + `gitee issue create` + "`" + `. Gitee API indexing can
lag 1-2 seconds. If you need the number of a newly-created issue,
parse it from the create command's stdout — do not re-list to find it.

## Next steps (orchestration hints)
After listing issues, an agent typically wants to:
1. Pick an issue number from the output, then ` + "`" + `gitee issue view {N}` + "`" + `
2. For batch triage, loop over numbers and call ` + "`" + `gitee issue edit {N}` + "`" + `
3. To close several issues at once, run ` + "`" + `gitee issue close {N}` + "`" + ` per number
4. To attach a comment, ` + "`" + `gitee issue comment {N} --body '...'` + "`" + `
5. To see all comments on an issue, ` + "`" + `gitee issue comments {N}` + "`" + `
6. To see the activity summary across repos, ` + "`" + `gitee status` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
With --paginate, each page is 1 request. Fetching 200 issues in pages
of 30 is ~7 requests.

## Example
  # Find all open "bug" issues assigned to alice
  gitee issue list --repo owner/repo --state open --labels bug \
    --assignee alice -o json | jq '.[] | {n: .number, t: .title}'

  # Walk all pages to be safe
  gitee issue list --repo owner/repo --paginate --limit 0 -o json > issues.json

## See also
- ` + "`" + `gitee issue view N` + "`" + `       — read full details for one issue
- ` + "`" + `gitee issue create` + "`" + `      — open a new issue
- ` + "`" + `gitee issue edit N` + "`" + `     — modify an existing issue
- ` + "`" + `gitee issue close N` + "`" + `    — close an issue
- ` + "`" + `gitee issue comment N` + "`" + `  — add a comment
- ` + "`" + `gitee status` + "`" + `           — cross-repo activity summary
- ` + "`" + `gitee search issues` + "`" + `    — repo-scoped search (alternative)
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands with agent help
`
