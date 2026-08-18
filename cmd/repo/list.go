package repo

import (
	"fmt"
	"net/url"
	"strconv"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
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

// validRepoVisibilities / validRepoSorts constrain the --visibility and
// --sort flags so a typo (or a value containing "&"/"#") cannot inject
// extra query parameters or silently request an unsupported sort key.
var validRepoVisibilities = map[string]bool{"public": true, "private": true, "internal": true, "all": true}
var validRepoSorts = map[string]bool{"created": true, "updated": true, "pushed": true, "full_name": true, "stars": true}

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

	if !validRepoVisibilities[listFlags.visibility] {
		return fmt.Errorf("invalid --visibility %q: use one of public|private|internal|all", listFlags.visibility)
	}
	if !validRepoSorts[listFlags.sort] {
		return fmt.Errorf("invalid --sort %q: use one of created|updated|pushed|full_name|stars", listFlags.sort)
	}
	if listFlags.direction != "asc" && listFlags.direction != "desc" {
		return fmt.Errorf("invalid --direction %q: use asc or desc", listFlags.direction)
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}
	opts := cmd.NewCmdkitOptions(client)
	return cmdkit.RunList[types.Repository](opts, cmdkit.ListSpec[types.Repository]{
		// repo list has no target repo — it lists by --owner (or the
		// authenticated user). Return empty owner/repo; Path ignores them.
		ResolveOwnerRepo: func() (string, string, error) {
			return "", "", nil
		},
		Path: func(_, _ string) string {
			path := api.ReposList(listFlags.owner)
			q := url.Values{}
			q.Set("visibility", listFlags.visibility)
			q.Set("sort", listFlags.sort)
			q.Set("direction", listFlags.direction)
			return path + "?" + q.Encode()
		},
		// repo list caps at listFlags.limit (default 30), not GlobalLimit.
		Limit:   func() int { return listFlags.limit },
		Columns: []string{"NAME", "VISIBILITY", "LANGUAGE", "STARS", "FORKS", "UPDATED"},
		Row: func(r types.Repository) []string {
			vis := "public"
			if r.Private {
				vis = "private"
			}
			lang := r.Language
			if lang == "" {
				lang = "-"
			}
			return []string{
				r.FullName, vis, lang,
				strconv.Itoa(r.StarsCount),
				strconv.Itoa(r.ForksCount),
				utils.TruncateDate(r.UpdatedAt),
			}
		},
		EmptyText: func(_, _ string) string {
			return "No repositories found"
		},
		ErrCtx: "listing repositories",
	})
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

## Race condition warning ⚠️
The shared mutable state is the account's repository set: repos are
created, deleted, renamed, and visibility-flipped by other agents and
humans continuously. This list is a point-in-time snapshot — the
classic failure is list→act: you pick ` + "`" + `owner/repo` + "`" + ` from the output
and call ` + "`" + `gitee repo edit` + "`" + ` / ` + "`" + `delete` + "`" + ` / ` + "`" + `clone` + "`" + `, but in the gap
the repo was renamed (old name 404s) or deleted. Conversely, a repo
created by a concurrent process may be MISSING from your listing.
STARS/FORKS/UPDATED columns are especially volatile — never make
ordering decisions ("the most recently updated repo is X") on a stale
list. Mitigation: act on full_name values promptly after listing;
on 404 from the follow-up command, re-list rather than retrying the
same name; and re-list immediately before any destructive batch loop.
Retrying THIS command is always safe (read-only).

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
