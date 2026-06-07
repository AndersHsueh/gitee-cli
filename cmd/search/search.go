package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/config"
	"gitee-cli/internal/utils"

	"github.com/spf13/cobra"
)

var SearchCmd = &cobra.Command{
	Use:   "search <subcommand>",
	Short: "Search for repositories, issues, or users on Gitee",
	Example: `  gitee search repos gitee
  gitee search issues "bug fix"
  gitee search users johndoe`,
}

// newSearchClient creates a client; token is optional for public search endpoints
func newSearchClient() *api.Client {
	token := cmd.GlobalToken
	if token == "" {
		token = config.GetToken(cmd.GlobalHostname)
	}
	return api.NewClient(token, cmd.GlobalAPIURL, cmd.GlobalVerbose, cmd.GlobalTimeout)
}

// ----- repos -----

var reposFlags struct {
	language string
	sort     string
	order    string
	limit    int
}

var searchReposCmd = &cobra.Command{
	Use:   "repos <query>",
	Short: "Search repositories",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearchRepos,
}

func init() {
	// repos flags
	{
		f := searchReposCmd.Flags()
		f.StringVar(&reposFlags.language, "language", "", "Filter by programming language")
		f.StringVar(&reposFlags.sort, "sort", "", "Sort by: stars|forks|updated")
		f.StringVar(&reposFlags.order, "order", "desc", "Sort order: asc|desc")
		f.IntVarP(&reposFlags.limit, "limit", "L", 20, "Maximum number of results")
	}
	// issues flags
	{
		f := searchIssuesCmd.Flags()
		f.StringVar(&issuesFlags.state, "state", "open", "Filter by state: open|closed|all")
		f.StringVar(&issuesFlags.labels, "labels", "", "Filter by labels")
		f.IntVarP(&issuesFlags.limit, "limit", "L", 20, "Maximum number of results")
	}
	// users flags
	{
		f := searchUsersCmd.Flags()
		f.IntVarP(&usersFlags.limit, "limit", "L", 20, "Maximum number of results")
	}
	// code flags (v0.10.0)
	{
		f := searchCodeCmd.Flags()
		f.StringVar(&codeFlags.language, "language", "", "Filter by programming language")
		f.StringVar(&codeFlags.owner, "owner", "", "Filter by repository owner")
		f.StringVar(&codeFlags.sort, "sort", "", "Sort by: indexed|match")
		f.StringVar(&codeFlags.order, "order", "desc", "Sort order: asc|desc")
		f.IntVarP(&codeFlags.limit, "limit", "L", 20, "Maximum number of results")
	}
	// commits flags (v0.10.0)
	{
		f := searchCommitsCmd.Flags()
		f.IntVarP(&commitsFlags.limit, "limit", "L", 20, "Maximum number of results")
	}

	SearchCmd.AddCommand(searchReposCmd, searchIssuesCmd, searchUsersCmd, searchCodeCmd, searchCommitsCmd)
}

func runSearchRepos(c *cobra.Command, args []string) error {
	q := args[0]
	client := newSearchClient()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	qv := url.Values{}
	qv.Set("q", q)
	qv.Set("per_page", fmt.Sprintf("%d", reposFlags.limit))
	if reposFlags.sort != "" {
		qv.Set("sort", reposFlags.sort)
	}
	if reposFlags.order != "" {
		qv.Set("order", reposFlags.order)
	}
	if reposFlags.language != "" {
		qv.Set("language", reposFlags.language)
	}
	path := "/search/repositories?" + qv.Encode()

	var result []interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		return fmt.Errorf("search repos: %w", err)
	}

	return printSearchResults(result, "repos")
}

// ----- issues -----

var issuesFlags struct {
	state  string
	labels string
	limit  int
}

var searchIssuesCmd = &cobra.Command{
	Use:   "issues <query>",
	Short: "Search issues across repositories",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearchIssues,
}

func runSearchIssues(c *cobra.Command, args []string) error {
	q := args[0]
	client := newSearchClient()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	qv := url.Values{}
	qv.Set("q", q)
	qv.Set("per_page", fmt.Sprintf("%d", issuesFlags.limit))
	if issuesFlags.state != "" && issuesFlags.state != "all" {
		qv.Set("state", issuesFlags.state)
	}
	if issuesFlags.labels != "" {
		qv.Set("labels", issuesFlags.labels)
	}
	path := "/search/issues?" + qv.Encode()

	var result interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		return fmt.Errorf("search issues: %w", err)
	}

	var items []interface{}
	if arr, ok := result.([]interface{}); ok {
		items = arr
	} else if m, ok := result.(map[string]interface{}); ok {
		if arr, ok := m["items"].([]interface{}); ok {
			items = arr
		} else {
			items = []interface{}{m}
		}
	}

	return printSearchResults(items, "issues")
}

// ----- users -----

var usersFlags struct {
	limit int
}

var searchUsersCmd = &cobra.Command{
	Use:   "users <query>",
	Short: "Search users",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearchUsers,
}

func runSearchUsers(c *cobra.Command, args []string) error {
	q := args[0]
	client := newSearchClient()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	qv := url.Values{}
	qv.Set("q", q)
	qv.Set("per_page", fmt.Sprintf("%d", usersFlags.limit))
	path := "/search/users?" + qv.Encode()

	var result interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		return fmt.Errorf("search users: %w", err)
	}

	// Convert to array for unified printing
	var items []interface{}
	if arr, ok := result.([]interface{}); ok {
		items = arr
	} else if m, ok := result.(map[string]interface{}); ok {
		if arr, ok := m["items"].([]interface{}); ok {
			items = arr
		} else {
			items = []interface{}{m}
		}
	}

	return printSearchResults(items, "users")
}

// ----- code (v0.10.0) -----

var codeFlags struct {
	language string
	owner    string
	sort     string
	order    string
	limit    int
}

var searchCodeCmd = &cobra.Command{
	Use:   "code <query>",
	Short: "Search code across repositories",
	Long: `Search for code across Gitee repositories.

Note: Code search availability depends on your Gitee instance.
If the API returns a 404, code search is not enabled for that instance.`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(searchCodeAgentHelp)
		}
		return nil
	},
	RunE: runSearchCode,
}

func runSearchCode(c *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("query is required (usage: gitee search code <query>)")
	}
	q := args[0]
	client := newSearchClient()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	qv := url.Values{}
	qv.Set("q", q)
	qv.Set("per_page", fmt.Sprintf("%d", codeFlags.limit))
	if codeFlags.language != "" {
		qv.Set("language", codeFlags.language)
	}
	if codeFlags.owner != "" {
		qv.Set("owner", codeFlags.owner)
	}
	if codeFlags.sort != "" {
		qv.Set("sort", codeFlags.sort)
	}
	if codeFlags.order != "" {
		qv.Set("order", codeFlags.order)
	}
	path := "/search/code?" + qv.Encode()

	var result interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		if api.IsNotFound(err) {
			return fmt.Errorf("code search is not available on this Gitee instance (404); try `gitee search repos %s` instead", q)
		}
		return fmt.Errorf("search code: %w", err)
	}

	var items []interface{}
	if arr, ok := result.([]interface{}); ok {
		items = arr
	} else if m, ok := result.(map[string]interface{}); ok {
		if arr, ok := m["items"].([]interface{}); ok {
			items = arr
		} else {
			items = []interface{}{m}
		}
	}

	return printSearchResults(items, "code")
}

// ----- commits (v0.10.0) -----

var commitsFlags struct {
	limit int
}

var searchCommitsCmd = &cobra.Command{
	Use:   "commits <query>",
	Short: "Search commits across repositories",
	Long: `Search for commits across Gitee repositories.

Note: Commit search availability depends on your Gitee instance.
If the API returns a 404, commit search is not enabled for that instance.`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(searchCommitsAgentHelp)
		}
		return nil
	},
	RunE: runSearchCommits,
}

func runSearchCommits(c *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("query is required (usage: gitee search commits <query>)")
	}
	q := args[0]
	client := newSearchClient()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	qv := url.Values{}
	qv.Set("q", q)
	qv.Set("per_page", fmt.Sprintf("%d", commitsFlags.limit))
	path := "/search/commits?" + qv.Encode()

	var result interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		if api.IsNotFound(err) {
			return fmt.Errorf("commit search is not available on this Gitee instance (404); try `gitee search issues %s` instead", q)
		}
		return fmt.Errorf("search commits: %w", err)
	}

	var items []interface{}
	if arr, ok := result.([]interface{}); ok {
		items = arr
	} else if m, ok := result.(map[string]interface{}); ok {
		if arr, ok := m["items"].([]interface{}); ok {
			items = arr
		} else {
			items = []interface{}{m}
		}
	}

	return printSearchResults(items, "commits")
}

// printSearchResults prints search results in a unified format
func printSearchResults(result interface{}, searchType string) error {
	var items []interface{}

	// Handle different response formats
	switch v := result.(type) {
	case []interface{}:
		items = v
	case map[string]interface{}:
		if arr, ok := v["items"].([]interface{}); ok {
			items = arr
		} else {
			items = []interface{}{v}
		}
	case nil:
		items = []interface{}{}
	default:
		items = []interface{}{}
	}

	// JSON output
	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(items, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	// Table output
	if len(items) == 0 {
		fmt.Println("(no results)")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	switch searchType {
	case "repos":
		fmt.Fprintln(w, "NAME\tLANGUAGE\tSTARS\tFORKS\tDESCRIPTION")
		for _, raw := range items {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := m["full_name"].(string)
			lang, _ := m["language"].(string)
			if lang == "" {
				lang = "-"
			}
			stars := fmt.Sprintf("%v", m["stargazers_count"])
			forks := fmt.Sprintf("%v", m["forks_count"])
			desc, _ := m["description"].(string)
			desc = utils.Truncate(desc, 50)
			if desc == "" {
				desc = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, lang, stars, forks, desc)
		}
	case "issues":
		fmt.Fprintln(w, "REPOSITORY\tNUMBER\tTITLE\tSTATE\tLABELS\tUPDATED")
		for _, raw := range items {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			repo, _ := m["repo"].(string)
			number := fmt.Sprintf("%v", m["number"])
			title, _ := m["title"].(string)
			state, _ := m["state"].(string)
			labelsRaw, _ := m["labels"].([]interface{})
			var labels []string
			for _, l := range labelsRaw {
				if lm, ok := l.(map[string]interface{}); ok {
					if name, ok := lm["name"].(string); ok {
						labels = append(labels, name)
					}
				}
			}
			labelStr := utils.Truncate(strings.Join(labels, ","), 20)
			if labelStr == "" {
				labelStr = "-"
			}
			updated, _ := m["updated_at"].(string)
			updated = utils.TruncateDate(updated)
			title = utils.Truncate(title, 40)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", repo, number, title, state, labelStr, updated)
		}
	case "users":
		fmt.Fprintln(w, "USERNAME\tNAME\tFOLLOWERS\tREPOS\tBIO")
		for _, raw := range items {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			login, _ := m["login"].(string)
			name, _ := m["name"].(string)
			if name == "" {
				name = "-"
			}
			followers := fmt.Sprintf("%v", m["followers"])
			repos := fmt.Sprintf("%v", m["public_repos"])
			bio, _ := m["bio"].(string)
			bio = utils.Truncate(bio, 30)
			if bio == "" {
				bio = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", login, name, followers, repos, bio)
		}
	case "code":
		fmt.Fprintln(w, "REPOSITORY\tFILE\tLANGUAGE\tURL")
		for _, raw := range items {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			repo, _ := m["repo"].(string)
			filePath, _ := m["path"].(string)
			lang, _ := m["language"].(string)
			if lang == "" {
				lang = "-"
			}
			htmlURL, _ := m["html_url"].(string)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", repo, filePath, lang, htmlURL)
		}
	case "commits":
		fmt.Fprintln(w, "REPOSITORY\tAUTHOR\tDATE\tMESSAGE")
		for _, raw := range items {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			repo, _ := m["repo"].(string)
			author := "-"
			if commit, ok2 := m["commit"].(map[string]interface{}); ok2 && commit != nil {
				if a, ok3 := commit["author"].(map[string]interface{}); ok3 && a != nil {
					if aName, ok4 := a["name"].(string); ok4 && aName != "" {
						author = aName
					}
				}
			}
			date, _ := m["updated_at"].(string)
			date = utils.TruncateDate(date)
			msg, _ := m["message"].(string)
			msg = utils.Truncate(msg, 60)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", repo, author, date, msg)
		}
	}

	_ = w.Flush()
	return nil
}

const searchCodeAgentHelp = `# gitee search code — Agent Help

## Purpose
Search for code across Gitee repositories using the Gitee v5 search API
(` + "`" + `GET /search/code` + "`" + `). Returns matching files with repository, path,
language, and URL.

Note: Code search availability depends on your Gitee instance. If your
instance does not support code search, the API returns 404 and gitee-cli
prints a friendly message suggesting ` + "`" + `gitee search repos` + "`" + ` instead.

## Pre-conditions
- Gitee instance has code search enabled (default for gitee.com)
- Token is optional for public repositories; required for private repos

## Required flags
  <query> (positional)   Search keywords

## Optional flags
  --language             Filter by programming language (e.g. go, python, java)
  --owner                Filter by repository owner
  --sort                 Sort by: indexed|match (default: indexed)
  --order                Sort order: asc|desc (default: desc)
  --limit / -L           Maximum number of results (default 20)
  --output / -o          table|json (default table)

## Idempotency
YES. Pure read — no state is modified.

## Atomicity
Not applicable (read-only).

## Side effects
None.

## Post-conditions (on success)
- Stdout contains a table (or JSON) of matching code results.

## Output (stdout, table mode)

  REPOSITORY         FILE               LANGUAGE   URL
  owner/repo         main.go            Go         https://gitee.com/...
  owner/repo         app_test.go        Go         https://gitee.com/...

## Common errors
- 404 Not Found: code search is not enabled on this Gitee instance
- 401 Unauthorized: token missing or invalid (for private repos)
- Empty results: no code matching the query was found

## OAuth scope required
projects (read) — only if searching private repos

## Rate limit
Search endpoints may have stricter rate limits than regular API calls.
For gitee.com: subject to the same 1000 req/hour limit for authenticated
users, or a lower anonymous limit for unauthenticated requests.

## Example
  gitee search code "func main" --language go --owner myorg
  gitee search code "import fmt" --output json | jq '.[].path'

## See also
- ` + "`" + `gitee search repos` + "`" + `      — search repositories by name
- ` + "`" + `gitee search commits` + "`" + `    — search commits
- ` + "`" + `gitee search issues` + "`" + `     — search issues
`

const searchCommitsAgentHelp = `# gitee search commits — Agent Help

## Purpose
Search for commits across Gitee repositories using the Gitee v5 search
API (` + "`" + `GET /search/commits` + "`" + `). Returns matching commits with repository,
author, date, and message.

Note: Commit search availability depends on your Gitee instance. If your
instance does not support commit search, the API returns 404 and gitee-cli
prints a friendly message suggesting ` + "`" + `gitee search issues` + "`" + ` instead.

## Pre-conditions
- Gitee instance has commit search enabled (default for gitee.com)
- Token is optional for public repositories; required for private repos

## Required flags
  <query> (positional)   Search keywords

## Optional flags
  --limit / -L           Maximum number of results (default 20)
  --output / -o          table|json (default table)

## Idempotency
YES. Pure read — no state is modified.

## Atomicity
Not applicable (read-only).

## Side effects
None.

## Post-conditions (on success)
- Stdout contains a table (or JSON) of matching commit results.

## Output (stdout, table mode)

  REPOSITORY         AUTHOR       DATE        MESSAGE
  owner/repo         octocat      2026-06-08  Fix login bug
  owner/repo         andershsueh  2026-06-07  Add tests

## Common errors
- 404 Not Found: commit search is not enabled on this Gitee instance
- 401 Unauthorized: token missing or invalid (for private repos)
- Empty results: no commits matching the query were found

## OAuth scope required
projects (read) — only if searching private repos

## Rate limit
Same as ` + "`" + `gitee search code` + "`" + ` — search endpoints may have stricter limits.

## Example
  gitee search commits "fix bug" --limit 10
  gitee search commits "security" --output json | jq '.[].repo'

## See also
- ` + "`" + `gitee search code` + "`" + `        — search code
- ` + "`" + `gitee search repos` + "`" + `       — search repositories
- ` + "`" + `gitee search issues` + "`" + `      — search issues
`
