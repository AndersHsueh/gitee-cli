package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var viewFlags struct {
	owner string
	repo  string
}

var viewCmd = &cobra.Command{
	Use:   "view [<owner>/<repo>]",
	Short: "View repository details",
	Example: `  gitee repo view owner/repo
  gitee repo view --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoViewAgentHelp)
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
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	owner, repoName, err := resolveRepo(viewFlags.owner, viewFlags.repo, args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReposGet(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Get(ctx, path, &repo); err != nil {
		return fmt.Errorf("viewing repository: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(repo, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	w := output.NewTableWriter()
	vis := "public"
	if repo.Private {
		vis = "private"
	}
	fmt.Fprintf(w, "Name:\t%s\n", repo.FullName)
	fmt.Fprintf(w, "Description:\t%s\n", repo.Description)
	fmt.Fprintf(w, "Visibility:\t%s\n", vis)
	fmt.Fprintf(w, "Language:\t%s\n", repo.Language)
	fmt.Fprintf(w, "Stars:\t%d\n", repo.StarsCount)
	fmt.Fprintf(w, "Forks:\t%d\n", repo.ForksCount)
	fmt.Fprintf(w, "Watchers:\t%d\n", repo.WatchersCount)
	fmt.Fprintf(w, "Default Branch:\t%s\n", repo.DefaultBranch)
	fmt.Fprintf(w, "Open Issues:\t%d\n", repo.OpenIssuesCount)
	fmt.Fprintf(w, "URL:\t%s\n", repo.HTMLURL)
	if repo.PushedAt != "" {
		fmt.Fprintf(w, "Pushed At:\t%s\n", utils.TruncateDate(repo.PushedAt))
	}
	if repo.CreatedAt != "" {
		fmt.Fprintf(w, "Created At:\t%s\n", utils.TruncateDate(repo.CreatedAt))
	}
	return w.Flush()
}

func resolveRepo(owner, repo string, args []string) (string, string, error) {
	return utils.ResolveRepo(owner, repo, args)
}

const repoViewAgentHelp = `# gitee repo view — Agent Help

## Purpose
Fetch and display a single repository's metadata. Use this as the
canonical "what is this repo?" pre-flight before any write verb
(edit, archive, fork, delete) so an agent can confirm identity,
visibility, default branch, and current state.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The target repo exists and is visible to the token
- One of the following must resolve to owner/repo:
    - positional argument:        ` + "`" + `gitee repo view owner/repo` + "`" + `
    - --repo / -R flag:           ` + "`" + `--repo owner/repo` + "`" + `
    - split --owner + --repo:     ` + "`" + `--owner owner --repo name` + "`" + `
    - global --repo fallback
    - local git remote fallback

## Optional flags
  --owner         Repository owner (when --repo is just the bare name)
  --repo / -R     Repository name, OR owner/name; overrides positional arg

## Idempotency
YES. This command only reads; running it twice returns the same data
(assuming no upstream mutations between calls). Safe to retry.

## Atomicity
Atomic per request. A single API call returns one repository record.
Either the whole record is returned, or none.

## Side effects
None. Read-only.

## Post-conditions
- Stdout is a key/value table (default) or one JSON object (-o json)
- Process exits 0 on success, non-zero on API error
- No state change on the server, no local cache write

## Output (stdout, table mode)
  Name:           owner/repo
  Description:    ...
  Visibility:     public
  Language:       Go
  Stars:          42
  Forks:          3
  Watchers:       10
  Default Branch: main
  Open Issues:    7
  URL:            https://gitee.com/owner/repo
  Pushed At:      2026-06-01
  Created At:     2024-03-15

Missing values (e.g. null language) are emitted as empty strings or "-".

## Output (stdout, json mode)
  {
    "id": 12345,
    "full_name": "owner/repo",
    "private": false,
    "description": "...",
    "language": "Go",
    "stargazers_count": 42,
    "forks_count": 3,
    "watchers_count": 10,
    "default_branch": "main",
    "open_issues_count": 7,
    "html_url": "https://gitee.com/owner/repo",
    "pushed_at": "2026-06-01T12:34:56Z",
    "created_at": "2024-03-15T09:00:00Z",
    "parent": null,
    "fork": false,
    "archived": false
  ]

Field names follow the Gitee API v5 /repos/{owner}/{repo} response.

## Common errors
- 401 Unauthorized:  token expired or invalid   →  gitee auth login
- 403 Forbidden:     repo is private and token cannot read it
- 404 Not Found:     owner/name doesn't exist; check spelling
- Network/timeout:   transient; safe to retry

## Race condition warning ⚠️
The shared mutable state is the repo's settings record and its
counters. Everything in this output is a point-in-time snapshot:
stars/forks/watchers/open_issues counts move continuously, and
visibility, default_branch, even the repo's NAME can change between
your view and a follow-up write. Classic failures: (1) you read
` + "`" + `"private": false` + "`" + `, decide it is safe to paste the URL somewhere
public, and in the gap an admin flips it private (or vice versa —
you skip sharing a repo that just went public); (2) you read
` + "`" + `default_branch: main` + "`" + ` and script against main, but an admin
switched the default to master; (3) the repo was renamed after your
view — follow-up calls to the old full_name 404. Mitigation: re-run
THIS command immediately before any decision that depends on these
fields, and prefer values from the most recent read over cached ones.
Retrying is always safe (read-only).

## Next steps (orchestration hints)
After viewing a repo, an agent typically wants to:
1. To modify it:           ` + "`" + `gitee repo edit owner/repo ...` + "`" + `
2. To clone it:            ` + "`" + `gitee repo clone owner/repo` + "`" + `
3. To fork it:             ` + "`" + `gitee repo fork owner/repo` + "`" + `
4. To drill into issues:   ` + "`" + `gitee issue list --repo owner/repo` + "`" + `
5. To drill into PRs:      ` + "`" + `gitee pr list --repo owner/repo` + "`" + `
6. To archive it:          ` + "`" + `gitee repo archive owner/repo` + "`" + `
   (use ` + "`" + `gitee repo view` + "`" + ` first to inspect Archived field for state)
7. To open in browser:     ` + "`" + `gitee repo browse owner/repo` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Inspect via positional arg
  gitee repo view myorg/billing-svc

  # Inspect via --repo, then pipe to jq to extract default_branch
  gitee repo view --repo myorg/billing-svc -o json \\
    | jq -r '.default_branch'

  # Use --owner + --repo when only one is set elsewhere
  gitee repo view --owner myorg --repo billing-svc

## See also
- ` + "`" + `gitee repo list` + "`" + `           — find candidates to view
- ` + "`" + `gitee repo edit` + "`" + `           — modify settings (write)
- ` + "`" + `gitee repo clone` + "`" + `          — fetch to local disk
- ` + "`" + `gitee repo browse` + "`" + `         — open in web browser
- ` + "`" + `gitee --agent-help` + "`" + `        — overview of all commands with agent help
`
