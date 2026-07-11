package org

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

var orgIssueFlags struct {
	org    string
	state  string
	labels string
	limit  int
}

var issueCmd = &cobra.Command{
	Use:   "issues <org>",
	Short: "List organization issues",
	Example: `  gitee org issues my-org
  gitee org issues my-org --state closed --labels bug`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(orgIssueAgentHelp)
		}
		return nil
	},
	RunE: runOrgIssues,
}

func init() {
	f := issueCmd.Flags()
	f.StringVar(&orgIssueFlags.org, "org", "", "Organization name")
	f.StringVar(&orgIssueFlags.state, "state", "open", "Filter by state: open|closed|all")
	f.StringVar(&orgIssueFlags.labels, "labels", "", "Filter by labels")
	f.IntVarP(&orgIssueFlags.limit, "limit", "L", 30, "Maximum number of issues")
}

func runOrgIssues(cobraCmd *cobra.Command, args []string) error {
	org := orgIssueFlags.org
	if len(args) > 0 {
		org = args[0]
	}
	if org == "" {
		return fmt.Errorf("organization name is required")
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	qv := url.Values{}
	qv.Set("state", orgIssueFlags.state)
	qv.Set("per_page", fmt.Sprintf("%d", orgIssueFlags.limit))
	if orgIssueFlags.labels != "" {
		qv.Set("labels", orgIssueFlags.labels)
	}
	path := api.OrgIssues(org) + "?" + qv.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var issues []types.Issue
	if err := client.Get(ctx, path, &issues); err != nil {
		return fmt.Errorf("listing org issues: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(issues, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(issues) == 0 {
		fmt.Println("No issues found")
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
		title := utils.Truncate(i.Title, 50)
		updated := i.UpdatedAt.Format("2006-01-02")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", i.Number, title, i.State, labelStr, updated)
	}
	return w.Flush()
}

// orgIssueAgentHelp is the agent-targeted help for `gitee org issues`.
const orgIssueAgentHelp = `# gitee org issues — Agent Help

## Purpose
List issues across every repository in a Gitee organization, with
state and label filtering. Use this to triage the org-wide backlog,
or to find issues matching a label across the whole org.

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- The org name resolves to an existing organization

## Required arguments
  <org>   Organization login (e.g. "my-org")

## Optional flags
  --org          Equivalent to passing <org> as a positional arg
  --state        Filter: "open" (default), "closed", or "all"
  --labels       Filter by label name (matches repos that have the label)
  --limit / -L   Maximum number of issues to return (default 30, max 100)
  --output       "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET on /orgs/{org}/issues.

## Post-conditions (on success)
- Stdout contains a table of issues (#, TITLE, STATE, LABELS, UPDATED)
  or a JSON array of Issue objects
- Repo state is unchanged
- If no issues match, "No issues found" is printed

## Output (stdout, table mode)
  #     TITLE                              STATE  LABELS              UPDATED
  142   Crash on startup                   open   bug,priority/high   2026-06-15
  138   Add dark mode                      open   enhancement         2026-06-10

## Output (stdout, json mode)
  [
    {
      "number": "142",
      "title": "Crash on startup",
      "state": "open",
      "labels": [{"name": "bug"}, {"name": "priority/high"}],
      "updated_at": "2026-06-15T...",
      ...
    }
  ]

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 404 Not Found:    org name doesn't exist
- 422 Unprocessable: invalid --state value (must be open/closed/all)

## Race condition warning
DO NOT rely on the issue count returned here as a stable signal. New
issues may be created concurrently in any repo in the org. Use the
specific issue number from a previous ` + "`" + `issue create` + "`" + ` call, or
from ` + "`" + `gitee issue list -R owner/repo` + "`" + `, when chaining commands.

## Next steps (orchestration hints)
After listing org issues, an agent typically wants to:
1. Pick a ` + "`" + `number` + "`" + ` and run ` + "`" + `gitee issue view {N} -R {org}/{repo}` + "`" + `
   (note: org issues don't carry the repo name in the number — resolve it
   from the JSON's ` + "`" + `repository.full_name` + "`" + ` field)
2. Or ` + "`" + `gitee issue comment {N} -R {org}/{repo} --body "..."` + "`" + ` to reply
3. Or ` + "`" + `gitee issue close {N} -R {org}/{repo}` + "`" + ` to resolve

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  # All open bugs across the org
  gitee org issues my-org --state open --labels bug

  # Pull the latest 5 closed issues as JSON
  gitee org issues my-org --state closed --limit 5 --output json

## See also
- ` + "`" + `gitee issue list` + "`" + `   — list issues in a single repo
- ` + "`" + `gitee issue view` + "`" + `   — read one issue
- ` + "`" + `gitee org repos` + "`" + `   — enumerate the org's repos first
`
