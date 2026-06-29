package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var listFlags struct {
	owner      string
	visibility string
	limit      int
	sort       string
	direction  string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories",
	Example: `  gitee repo list
  gitee repo list --owner gitee --visibility public
  gitee repo list --sort stars --direction desc`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoListAgentHelp)
		}
		return nil
	},
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.StringVar(&listFlags.owner, "owner", "", "Filter by owner username")
	f.StringVar(&listFlags.visibility, "visibility", "all", "Filter by visibility: public|private|internal|all")
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of repositories")
	f.StringVar(&listFlags.sort, "sort", "full_name", "Sort by: created|updated|pushed|full_name|stars")
	f.StringVar(&listFlags.direction, "direction", "asc", "Sort direction: asc|desc")
}

func runList(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	var path string
	if listFlags.owner != "" {
		path = api.ReposList(listFlags.owner)
	} else {
		path = api.ReposList("")
	}
	path += fmt.Sprintf("?visibility=%s&sort=%s&direction=%s&per_page=%d",
		listFlags.visibility, listFlags.sort, listFlags.direction, listFlags.limit)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repos []types.Repository
	if err := client.Get(ctx, path, &repos); err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(repos, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVISIBILITY\tLANGUAGE\tSTARS\tFORKS\tUPDATED")
	for _, r := range repos {
		vis := "public"
		if r.Private {
			vis = "private"
		}
		lang := r.Language
		if lang == "" {
			lang = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%s\n",
			r.FullName, vis, lang, r.StarsCount, r.ForksCount, utils.TruncateDate(r.UpdatedAt))
	}
	return w.Flush()
}

const repoListAgentHelp = `# gitee repo list — Agent Help

## Purpose
List repositories on Gitee. Without --owner this lists the authenticated
user's own repositories; with --owner it lists any user's (or org's) repos
they have access to. Use this for inventory, candidate selection, or
batch pre-flight checks before invoking other repo commands.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- If --owner is set: the target user/org exists and is visible to the token
- API host is reachable (default: gitee.com)

## Optional flags
  --owner               Filter by owner username (user or org). Omit for "me".
  --visibility          Filter: public | private | internal | all (default: all)
  --limit / -L          Maximum number of repositories to return (default: 30)
  --sort                Sort key: created | updated | pushed | full_name | stars
  --direction           Sort direction: asc | desc

## Idempotency
YES. This command only reads; running it twice returns the same data
(assuming no upstream repo changes between calls). Safe to retry.

## Atomicity
Atomic per request. A single API call returns one page of results.
The full set is NOT returned in one call if --limit exceeds per_page (100)
— pagination is NOT implemented in this verb, so requests beyond 100 are
truncated server-side. For more than 100 repos, narrow --owner.

## Side effects
None. Read-only.

## Post-conditions
- Stdout is a table (default) or JSON array (-o json) of repository records
- Process exits 0 even when zero repos match (prints "No repositories found")
- No state change on the server, no local cache write

## Output (stdout, table mode)
  NAME                VISIBILITY  LANGUAGE  STARS  FORKS  UPDATED
  owner/repo          public      Go        42     3      2026-06-01

Columns: NAME, VISIBILITY, LANGUAGE, STARS, FORKS, UPDATED (YYYY-MM-DD).

## Output (stdout, json mode)
  [
    {
      "id": 12345,
      "full_name": "owner/repo",
      "private": false,
      "description": "...",
      "language": "Go",
      "stargazers_count": 42,
      "forks_count": 3,
      "updated_at": "2026-06-01T12:34:56Z",
      "html_url": "https://gitee.com/owner/repo"
    }
  ]

Field names follow the Gitee API v5 /repos response. Use the FullName
field (e.g. "owner/repo") as input to other commands.

## Common errors
- 401 Unauthorized:  token expired or invalid   →  gitee auth login
- 404 Not Found:     --owner refers to non-existent user/org
- 422 Unprocessable: invalid --visibility value  →  use one of public|private|internal|all
- Network/timeout:   transient; safe to retry

## Next steps (orchestration hints)
After listing repos, an agent typically wants to:
1. Pick a target by FullName (column 1 in table mode) and feed it to:
   - ` + "`" + `gitee repo view owner/repo` + "`" + `         — inspect one repo
   - ` + "`" + `gitee repo clone owner/repo` + "`" + `         — download locally
   - ` + "`" + `gitee repo edit owner/repo ...` + "`" + `     — modify settings
   - ` + "`" + `gitee issue list --repo owner/repo` + "`" + ` — drill into issues
2. To list organizations the token can see, use ` + "`" + `gitee org list` + "`" + `
   first, then re-run ` + "`" + `gitee repo list --owner <org-name>` + "`" + `
3. To find repos across the platform (not just one user), use
   ` + "`" + `gitee search repos <query>` + "`" + ` instead.

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # List my own repos, sorted by most-starred first
  gitee repo list --sort stars --direction desc --limit 50 -o json

  # List an org's public repos
  gitee repo list --owner gitee --visibility public --limit 20

  # Pipe the JSON to jq and drill into one
  gitee repo list --owner myorg -o json | jq -r '.[0].full_name' \\
    | xargs -I {} gitee repo view {}

## See also
- ` + "`" + `gitee repo view` + "`" + `           — show one repo's details
- ` + "`" + `gitee repo clone` + "`" + `          — clone to local disk
- ` + "`" + `gitee repo create` + "`" + `         — create a new repo
- ` + "`" + `gitee search repos` + "`" + `       — search across all of Gitee
- ` + "`" + `gitee org list` + "`" + `           — list orgs to use with --owner
- ` + "`" + `gitee --agent-help` + "`" + `        — overview of all commands with agent help
`
