package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/config"
	"gitee-cli/internal/output"
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

// newSearchClient creates a client; token is optional for public search endpoints.
// Uses ResolveAPIBaseURL so an enterprise host (via --hostname or auth switch)
// resolves to the right API base, rather than falling back to the hardcoded
// gitee.com default inside api.NewClient.
func newSearchClient() *api.Client {
	hostname := cmd.ResolveHostname()
	token := cmd.GlobalToken
	if token == "" {
		token = config.GetToken(hostname)
	}
	return api.NewClient(token, cmd.ResolveAPIBaseURL(hostname), cmd.GlobalVerbose, cmd.GlobalTimeout)
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
	Args:  cmd.AgentHelpAwareArgs(cobra.MinimumNArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(searchReposAgentHelp)
		}
		return nil
	},
	RunE: runSearchRepos,
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
	_ = cmd.RuntimeFromCmd(c)
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
	Args:  cmd.AgentHelpAwareArgs(cobra.MinimumNArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(searchIssuesAgentHelp)
		}
		return nil
	},
	RunE: runSearchIssues,
}

func runSearchIssues(c *cobra.Command, args []string) error {
	_ = cmd.RuntimeFromCmd(c)
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

	return printSearchResults(result, "issues")
}

// ----- users -----

var usersFlags struct {
	limit int
}

var searchUsersCmd = &cobra.Command{
	Use:   "users <query>",
	Short: "Search users",
	Args:  cmd.AgentHelpAwareArgs(cobra.MinimumNArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(searchUsersAgentHelp)
		}
		return nil
	},
	RunE: runSearchUsers,
}

func runSearchUsers(c *cobra.Command, args []string) error {
	_ = cmd.RuntimeFromCmd(c)
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

	return printSearchResults(result, "users")
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
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(searchCodeAgentHelp)
		}
		return nil
	},
	RunE: runSearchCode,
}

func runSearchCode(c *cobra.Command, args []string) error {
	_ = cmd.RuntimeFromCmd(c)
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

	return printSearchResults(result, "code")
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
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(searchCommitsAgentHelp)
		}
		return nil
	},
	RunE: runSearchCommits,
}

func runSearchCommits(c *cobra.Command, args []string) error {
	_ = cmd.RuntimeFromCmd(c)
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

	return printSearchResults(result, "commits")
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

	w := output.NewTableWriter()

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

## Race condition warning ⚠️
Read-only, but the shared state behind the results — the code index
and the repos it indexes — keeps moving. A hit means the code existed
at index time: the file may have been edited, renamed, or deleted (or
the repo made private) by the time you follow the URL. Conversely,
code just pushed may be absent until the index catches up. If a
follow-up action depends on a match (e.g. patching every file
containing a symbol), re-fetch the file content immediately before
acting; do not treat the search hit as current. Safe to retry freely
(see Idempotency).

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

## Race condition warning ⚠️
Read-only; the shared mutable state is the commit index plus the repos
behind it. A commit you just pushed may be absent from results for a
while (indexing lag) — do not re-search to confirm a commit you just
created; use the sha from your local git output directly. Conversely,
a result's repo may have been force-pushed, deleted, or made private
between this search and a follow-up fetch, leaving the sha
unreachable. Re-run the search rather than trusting cached results if
time has passed. Safe to retry freely (see Idempotency).

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

const searchReposAgentHelp = `# gitee search repos — Agent Help

## Purpose
Search for repositories on Gitee using the v5 search API
(` + "`" + `GET /search/repositories` + "`" + `). Returns a ranked list of repos
matching the query, optionally filtered by language and sorted by stars,
forks, or last-updated.

Use this to discover candidate repos before cloning, forking, or
reading source code. Compare to ` + "`" + `gitee search code` + "`" + `, which searches
inside files; ` + "`" + `gitee search repos` + "`" + ` searches repo metadata.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Token is OPTIONAL for public repos. The Gitee v5 search endpoint
  returns public results without a token, but the anonymous rate limit
  is much lower.

## Required arguments
  <query> (positional)   Search keywords (e.g. "go cobra cli")

## Optional flags
  --language             Filter by programming language (e.g. go, python, java)
  --sort                 Sort by: stars|forks|updated (default: Gitee ranking)
  --order                Sort order: asc|desc (default: desc)
  --limit / -L           Maximum number of results (default 20)
  --output / -o          table|json (default table)

## Idempotency
YES. Pure read — no state is modified. Calling this command any number
of times with the same arguments returns equivalent (but time-shifted)
results because the Gitee index is constantly updated.

## Atomicity
Not applicable (read-only). The response is a single API call; either
the whole list is returned or the call fails.

## Side effects
None. No writes, no notifications, no webhooks fired.

## Post-conditions (on success)
- Stdout contains a table of matching repositories (or JSON array)
- Exit code is 0
- The query is not recorded anywhere client-side

## Output (stdout, table mode)

  NAME                  LANGUAGE   STARS   FORKS  DESCRIPTION
  owner/repo            Go         1234    56     A short description
  owner/other-repo      Python     890     12     Another description

## Output (stdout, json mode)
Returns a JSON array of repository objects with the Gitee v5 schema
(full_name, language, stargazers_count, forks_count, description,
html_url, etc.). Each object is exactly the same shape as
` + "`" + `gitee repo view` + "`" + ` returns for a single repository.

## Parsing the output
- table: pipe through ` + "`" + `awk` + "`" + ` or ` + "`" + `column -t -s $'\t'` + "`" + ` for human consumption
- json:  ` + "`" + `jq '.[].full_name'` + "`" + ` to extract a list of owner/repo slugs

  REPOS=$(gitee search repos "go cli" --limit 10 --output json | jq -r '.[].full_name')
  for r in $REPOS; do
    gitee repo view "$r"
  done

## Common errors
- 401 Unauthorized:  token expired or invalid          →  gitee auth login
- 403 Forbidden:     rate limit exceeded for anonymous  →  authenticate
- 404 Not Found:     malformed query or bad URL        →  check query syntax
- Empty results:     no repos matched the query         →  try broader terms

## Race condition warning ⚠️
Read-only, but repo metadata is shared, moving state: stars, forks,
and rankings shift constantly, and a repo in the results can be
renamed, deleted, or switched to private between this search and a
follow-up ` + "`" + `gitee repo view` + "`" + ` / clone — expect occasional 404s and
re-resolve the slug instead of retrying the old one. A just-created
repo may also be absent from results due to index lag; use the
full_name from THIS output (or from the create output) directly rather
than re-searching to confirm it exists. Safe to retry freely
(see Idempotency).

## Next steps (orchestration hints)
After ` + "`" + `gitee search repos` + "`" + `, an agent typically wants to:
1. Pick a candidate from the table and call ` + "`" + `gitee repo view owner/repo` + "`" + `
   to read the README / description / topics
2. If interested in contributing: ` + "`" + `gitee repo fork owner/repo` + "`" + ` then clone
3. If the user wants to read the source: ` + "`" + `gitee repo clone owner/repo` + "`" + `
4. To find issues to work on inside one of these repos, follow up with
   ` + "`" + `gitee issue list --repo owner/repo` + "`" + ` (NOT another search call)

## OAuth scope required
None for public repos. For searching private repos you need a token
with at least ` + "`" + `projects` + "`" + ` (read).

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users. Anonymous
requests are capped at a much lower IP-based limit and will return 403
if exhausted. Each call to this command counts as 1 request.

## Example
  # Discover Go CLI tools on Gitee
  gitee search repos "go cli" --language go --sort stars --limit 10

  # Machine-friendly: pull top 5 star counts as JSON
  gitee search repos "machine learning" --language python \
    --sort stars --order desc --limit 5 --output json

## See also
- ` + "`" + `gitee repo view` + "`" + `       — read a single repo's metadata
- ` + "`" + `gitee repo clone` + "`" + `      — clone a repo locally
- ` + "`" + `gitee repo fork` + "`" + `       — fork a repo to your account
- ` + "`" + `gitee search code` + "`" + `      — search code INSIDE repos
- ` + "`" + `gitee search issues` + "`" + `   — search issues across repos
- ` + "`" + `gitee --agent-help` + "`" + `    — overview of all commands with agent help
`

const searchIssuesAgentHelp = `# gitee search issues — Agent Help

## Purpose
Search for issues across all Gitee repositories using the v5 search
API (` + "`" + `GET /search/issues` + "`" + `). Returns a ranked list of issues
matching the query, with optional state and label filters.

Use this to discover issues by keyword across the entire Gitee
ecosystem — for example "memory leak", "translation", or a specific
error message string. For issues in a single repo, prefer
` + "`" + `gitee issue list --repo owner/repo` + "`" + ` (it's faster and respects
the repo's local view).

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Token is OPTIONAL for public issues. The Gitee v5 search endpoint
  returns public issues without a token, but the anonymous rate limit
  is much lower.

## Required arguments
  <query> (positional)   Search keywords (e.g. "memory leak in loader")

## Optional flags
  --state                Filter by state: open|closed|all (default: open)
  --labels               Filter by label names (comma-separated, AND-ed)
  --limit / -L           Maximum number of results (default 20)
  --output / -o          table|json (default table)

## Idempotency
YES. Pure read — no state is modified.

## Atomicity
Not applicable (read-only). The response is a single API call.

## Side effects
None. No writes, no notifications, no webhooks fired.

## Post-conditions (on success)
- Stdout contains a table of matching issues (or JSON array)
- Exit code is 0

## Output (stdout, table mode)

  REPOSITORY          NUMBER  TITLE                          STATE    LABELS              UPDATED
  owner/repo          42      Memory leak in loader          open     bug                 2026-06-08
  owner/other         17      Translation fr-CA missing      closed   i18n,help wanted   2026-06-07

## Output (stdout, json mode)
Returns a JSON array of issue objects with the Gitee v5 schema
(repo, number, title, state, labels, updated_at, etc.).

## Parsing the output
- table: ` + "`" + `awk '{print $1"#"$2}'` + "`" + ` to extract ` + "`" + `owner/repo#N` + "`" + ` references
- json:  ` + "`" + `jq '.[] | "\(.repo)#\(.number)"'` + "`" + ` to build the same refs

  REFS=$(gitee search issues "memory leak" --output json \
    | jq -r '.[] | "\(.repo)#\(.number)"')
  for ref in $REFS; do
    gitee issue view "$ref"
  done

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     rate limit exceeded for anonymous  →  authenticate
- 404 Not Found:     malformed query or bad URL  →  check query syntax
- Empty results:     no issues matched the query   →  try broader terms

## Race condition warning ⚠️
Read-only, but issues are highly mutable shared state. Between this
search and any follow-up action, an issue's state, labels, or title
can flip — another agent may have closed it, or someone may already
be working on it. If your action depends on state (e.g. "only comment
if still open"), re-check with ` + "`" + `gitee issue view owner/repo#N` + "`" + `
immediately before acting. Conversely, a just-created issue may be
absent from results due to search-index lag — do not re-search to
verify a creation; use the number returned by ` + "`" + `gitee issue create` + "`" + `.
Safe to retry freely (see Idempotency).

## Next steps (orchestration hints)
After ` + "`" + `gitee search issues` + "`" + `, an agent typically wants to:
1. Pick a candidate issue and call ` + "`" + `gitee issue view owner/repo#N` + "`" + `
   to read the full body and any existing comments
2. If the issue is a duplicate, add a comment via
   ` + "`" + `gitee issue comment N --repo owner/repo --body "..."` + "`" + `
3. If you intend to fix it: ` + "`" + `gitee repo fork owner/repo` + "`" + `, then
   create a PR with ` + "`" + `Closes owner/repo#N` + "`" + ` in the body
4. If the issue is a false positive: skip — do not "close" issues you
   did not author without explicit user consent

## OAuth scope required
None for public issues. For searching issues in private repos you need
a token with at least ` + "`" + `projects` + "`" + ` (read).

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users. Anonymous
requests are capped at a much lower IP-based limit and will return 403
if exhausted.

## Example
  # Find open translation issues
  gitee search issues "translation" --state open --limit 20

  # Find issues tagged "bug" in any repo
  gitee search issues "crash" --labels bug --limit 10 --output json

## See also
- ` + "`" + `gitee issue list` + "`" + `       — list issues in ONE repo (faster)
- ` + "`" + `gitee issue view` + "`" + `       — read a single issue with body
- ` + "`" + `gitee issue comment` + "`" + `   — add a comment to an issue
- ` + "`" + `gitee search repos` + "`" + `     — find repos by metadata
- ` + "`" + `gitee --agent-help` + "`" + `    — overview of all commands with agent help
`

const searchUsersAgentHelp = `# gitee search users — Agent Help

## Purpose
Search for Gitee users by username, display name, or bio using the
v5 search API (` + "`" + `GET /search/users` + "`" + `). Returns a ranked list of
matching user profiles.

Use this to discover potential collaborators, find a user whose work
you want to follow, or look up someone's public repos and gists.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Token is OPTIONAL for public user profiles. The Gitee v5 search
  endpoint returns public users without a token, but the anonymous
  rate limit is much lower.

## Required arguments
  <query> (positional)   Search keywords (username fragment, name, or bio text)

## Optional flags
  --limit / -L           Maximum number of results (default 20)
  --output / -o          table|json (default table)

## Idempotency
YES. Pure read — no state is modified.

## Atomicity
Not applicable (read-only). The response is a single API call.

## Side effects
None. No writes, no notifications. Note that user-search is rate
limited more aggressively than repo/issue search on the Gitee API.

## Post-conditions (on success)
- Stdout contains a table of matching users (or JSON array)
- Exit code is 0

## Output (stdout, table mode)

  USERNAME         NAME             FOLLOWERS   REPOS  BIO
  andershsueh      Anders Hsueh     142         28     Maintainer of gitee-cli
  octocat          Octocat          9999        8      The GitHub/Gitee mascot

## Output (stdout, json mode)
Returns a JSON array of user objects with the Gitee v5 schema
(login, name, followers, public_repos, bio, html_url, etc.).

## Parsing the output
- table: ` + "`" + `awk '{print $1}'` + "`" + ` to extract the login column
- json:  ` + "`" + `jq -r '.[].login'` + "`" + ` to get just the usernames

  USERS=$(gitee search users "go" --output json | jq -r '.[].login')
  for u in $USERS; do
    gitee repo list --owner "$u" --limit 5
  done

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     rate limit exceeded for anonymous  →  authenticate,
                                                          or wait 60s
- 404 Not Found:     malformed query  →  check query syntax
- Empty results:     no users matched the query   →  try broader terms

## Race condition warning ⚠️
Read-only, but user profiles are mutable: a user can rename their
login, edit their bio, or make their profile private between this
search and any follow-up action — after a rename, the old login 404s.
If a workflow depends on the account (mentioning @USER, listing their
repos), use the login from THIS output and tolerate 404 by re-searching.
Newly registered users may not appear in results immediately due to
index lag. Safe to retry freely (see Idempotency).

## Next steps (orchestration hints)
After ` + "`" + `gitee search users` + "`" + `, an agent typically wants to:
1. Pick a candidate and explore their public repos:
   ` + "`" + `gitee repo list --owner USERNAME --limit 10` + "`" + `
2. If the user is the owner of a repo you want to fork:
   ` + "`" + `gitee repo fork owner/repo` + "`" + `
3. To send the user a notification (issue mention / PR review request),
   reference them with ` + "`" + `@USERNAME` + "`" + ` in the body text

## OAuth scope required
None for public user profiles. For full email/follower details of
private accounts, a token with ` + "`" + `user_info` + "`" + ` scope may be required.

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users on the global
search endpoint. User search specifically is throttled at ~60 req/min
per token. Each call to this command counts as 1 request.

## Example
  # Find Go developers on Gitee
  gitee search users "golang" --limit 10

  # Machine-friendly: extract login list
  gitee search users "machine learning" --limit 5 --output json \
    | jq -r '.[].login'

## See also
- ` + "`" + `gitee repo list` + "`" + `        — list a user's repos (faster than searching)
- ` + "`" + `gitee org list` + "`" + `        — list orgs the authenticated user belongs to
- ` + "`" + `gitee search repos` + "`" + `    — search repos by metadata
- ` + "`" + `gitee --agent-help` + "`" + `    — overview of all commands with agent help
`
