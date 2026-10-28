package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Style helpers.
var (
	issueStatusHeaderStyle = color.New(color.Bold).SprintFunc()
	issueStatusDimStyle    = color.New(color.Faint).SprintFunc()
)

type issueStatusSection struct {
	Title string            `json:"title"`
	Count int               `json:"count"`
	Items []issueStatusItem `json:"items"`
}

type issueStatusItem struct {
	Number string `json:"number"`
	Title  string `json:"title"`
	Repo   string `json:"repo"`
	URL    string `json:"url,omitempty"`
	State  string `json:"state,omitempty"`
}

var issueStatusFlags struct {
	limit int
}

var issueStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cross-repo issue status for the current user",
	Long: `Show a summary of issues across your Gitee repositories:
Issues assigned to you and issues you've opened.

Uses the Gitee search API for efficient cross-repo queries
rather than iterating each repo individually.`,
	Example: `  gitee issue status
  gitee issue status --output json
  gitee issue status --limit 200`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(issueStatusAgentHelp)
		}
		return nil
	},
	RunE: runIssueStatus,
}

func init() {
	f := issueStatusCmd.Flags()
	f.IntVarP(&issueStatusFlags.limit, "limit", "L", 50, "Maximum results per section (0 for unlimited)")

	IssueCmd.AddCommand(issueStatusCmd)
}

func runIssueStatus(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// Resolve current username via shared helper.
	username, err := cmd.ResolveCurrentUsername(ctx, client)
	if err != nil {
		return err
	}
	if username == "" {
		return fmt.Errorf("could not determine current user")
	}

	perPage := issueStatusFlags.limit
	if perPage <= 0 {
		perPage = 100
	}

	// Search: issues assigned to current user.
	assignedItems, err := searchIssues(ctx, client, "state:open+assignee:"+url.QueryEscape(username), perPage)
	if err != nil {
		return fmt.Errorf("searching issues assigned to you: %w", err)
	}

	// Search: issues opened by current user.
	openedItems, err := searchIssues(ctx, client, "state:open+author:"+url.QueryEscape(username), perPage)
	if err != nil {
		return fmt.Errorf("searching issues you opened: %w", err)
	}

	return renderIssueStatus(username, assignedItems, openedItems)
}

func searchIssues(ctx context.Context, client *api.Client, q string, limit int) ([]issueStatusItem, error) {
	qv := url.Values{}
	qv.Set("q", q)
	qv.Set("per_page", fmt.Sprintf("%d", limit))
	path := "/search/issues?" + qv.Encode()

	var result interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		return nil, err
	}

	var rawItems []interface{}
	if arr, ok := result.([]interface{}); ok {
		rawItems = arr
	} else if m, ok := result.(map[string]interface{}); ok {
		if arr, ok := m["items"].([]interface{}); ok {
			rawItems = arr
		} else {
			rawItems = []interface{}{m}
		}
	}

	var items []issueStatusItem
	for _, raw := range rawItems {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		repo, _ := m["repo"].(string)
		title, _ := m["title"].(string)
		state, _ := m["state"].(string)
		htmlURL, _ := m["html_url"].(string)
		number := fmt.Sprintf("%v", m["number"])

		items = append(items, issueStatusItem{
			Number: number,
			Title:  title,
			Repo:   repo,
			URL:    htmlURL,
			State:  state,
		})
	}

	// Warn if results were potentially truncated (v0.10.0).
	if len(items) >= limit && limit > 0 {
		fmt.Fprintf(os.Stderr, "warning: results truncated at %d; use --limit to fetch more (e.g. --limit 200)\n", limit)
	}

	return items, nil
}

func renderIssueStatus(username string, assigned, opened []issueStatusItem) error {
	if cmd.GlobalOutput == "json" {
		out := map[string]issueStatusSection{
			"assigned_to_you": {
				Title: "Issues assigned to you",
				Count: len(assigned),
				Items: assigned,
			},
			"opened_by_you": {
				Title: "Issues you opened",
				Count: len(opened),
				Items: opened,
			},
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("%s %s\n\n", issueStatusDimStyle("current user:"), username)

	if len(assigned) == 0 && len(opened) == 0 {
		fmt.Printf("  %s\n", issueStatusDimStyle("no open issues found"))
		return nil
	}

	renderIssueSection("Issues assigned to you", assigned)
	renderIssueSection("Issues you opened", opened)

	return nil
}

func renderIssueSection(title string, items []issueStatusItem) {
	fmt.Printf("%s  %s\n", issueStatusHeaderStyle(title), issueStatusHeaderStyle(fmt.Sprintf("(%d)", len(items))))
	if len(items) == 0 {
		fmt.Printf("  %s\n\n", issueStatusDimStyle("none"))
		return
	}
	w := output.NewTableWriter()
	fmt.Fprintln(w, "REPOSITORY\t#\tTITLE\tSTATE")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			item.Repo, item.Number,
			utils.Truncate(item.Title, 50),
			item.State)
	}
	_ = w.Flush()
	fmt.Println()
}

const issueStatusAgentHelp = `# gitee issue status — Agent Help

## Purpose
Show a cross-repository summary of open issues for the authenticated user.
Uses the Gitee search API (` + "`" + `GET /search/issues` + "`" + `) for efficient cross-repo
queries, issuing two searches: ` + "`" + `assignee:<user>` + "`" + ` and ` + "`" + `author:<user>` + "`" + `.
Results are grouped into two sections: issues assigned to you, and issues
you opened.

## Pre-conditions
- Authenticated (token must be valid)
- Gitee instance has search index enabled (on by default for gitee.com)

## Required flags
None.

## Optional flags
  --limit / -L        Max results per section (default 50). Set to 0 for unlimited.
  --output / -o       table|json (default table)

## Idempotency
YES. Pure read — no state is modified.

## Atomicity
Not applicable (read-only).

## Side effects
None. No emails, no webhooks, no notifications.

## Post-conditions (on success)
- Stdout contains a table (or JSON) of open issues grouped by category.
- If results equal the --limit value, a truncation warning is printed to stderr.

## Output (stdout, table mode)

  Issues assigned to you  (5)
  REPOSITORY              #        TITLE                                STATE
  owner/repo1             IJP83Z   Fix login bug                       open
  owner/repo2             42       Add tests                           open

  Issues you opened  (2)
  REPOSITORY              #        TITLE                                STATE
  owner/repo3             7        Update README                       open

## Common errors
- 401 Unauthorized: run ` + "`" + `gitee auth login` + "`" + ` or set GITEE_TOKEN
- Empty results: user has no open issues assigned to or opened by them
- "results truncated at N" warning (stderr): increase --limit to fetch more

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour authenticated (two search API calls per invocation).
Search endpoints may have stricter rate limits than regular API calls.

## Race conditions
- An issue created/closed between the two search calls may appear in one
  section but not the other, or appear in both.
- Search index may lag behind real-time state by a few seconds.

## Example
  gitee issue status
  gitee issue status --output json | jq '.assigned_to_you.items[] | {repo, number, title}'

## See also
- ` + "`" + `gitee pr status` + "`" + `         — cross-repo PR summary
- ` + "`" + `gitee status` + "`" + `           — single-repo status (issues + PRs in one view)
- ` + "`" + `gitee search issues` + "`" + `     — raw search with custom query
- ` + "`" + `gitee issue list` + "`" + `        — list issues in a specific repo
`
