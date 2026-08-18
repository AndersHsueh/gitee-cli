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

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Per the P1-6 spec, the star subcommand group lives entirely in this
// one file (`cmd/repo/star.go`) and registers three verbs under the
// `gitee repo star` umbrella:
//   star    — PUT   /user/starred/{owner}/{repo}
//   unstar  — DELETE /user/starred/{owner}/{repo}
//   list    — GET   /user/starred            (repos the caller has starred)
//
// All three share the --repo / -R flag pattern for star/unstar; list
// has no repo argument (it's "what have I starred?").

// starFlags is shared across star/unstar/list so the user can invoke
// `gitee repo star --repo owner/repo` (Cobra binds by pointer).
var starFlags struct {
	repo  string
	limit int
	sort  string
}

var starCmd = &cobra.Command{
	Use:   "star <subcommand>",
	Short: "Star, unstar, or list starred repositories",
	Long: `Manage stars on Gitee repositories.

Starring a repository bookmarks it to your personal profile and signals
support for the author. Use these verbs to add a star, remove an
existing star, or enumerate the repositories you have already starred.

  gitee repo star <owner>/<repo>   — star a repository to support its author
  gitee repo unstar <owner>/<repo> — remove a star
  gitee repo star list             — list repositories you have starred

Note: starring is independent of watching. Star a repo to bookmark it;
use gitee repo watch to receive notifications about its activity.`,
	Example: `  gitee repo star myorg/myrepo
  gitee repo star myorg/myrepo --repo myorg/myrepo
  gitee repo unstar myorg/myrepo
  gitee repo star list
  gitee repo star list --limit 50`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoStarAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		// No subcommand given — show help so users discover star/unstar/list.
		return c.Help()
	},
}

var starStarCmd = &cobra.Command{
	Use:   "star [owner/repo]",
	Short: "Star a repository to support its author",
	Example: `  gitee repo star myorg/myrepo
  gitee repo star --repo myorg/myrepo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoStarStarAgentHelp)
		}
		return nil
	},
	RunE: runRepoStar,
}

var starUnstarCmd = &cobra.Command{
	Use:   "unstar [owner/repo]",
	Short: "Remove a star from a repository",
	Example: `  gitee repo unstar myorg/myrepo
  gitee repo unstar --repo myorg/myrepo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoStarUnstarAgentHelp)
		}
		return nil
	},
	RunE: runRepoUnstar,
}

var starListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories you have starred",
	Example: `  gitee repo star list
  gitee repo star list --limit 50`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoStarListAgentHelp)
		}
		return nil
	},
	RunE: runRepoStarList,
}

func init() {
	// star flags
	sf := starStarCmd.Flags()
	sf.StringVarP(&starFlags.repo, "repo", "R", "", "Repository (owner/repo); falls back to global --repo, then to the positional arg")

	// unstar flags
	uf := starUnstarCmd.Flags()
	uf.StringVarP(&starFlags.repo, "repo", "R", "", "Repository (owner/repo); falls back to global --repo, then to the positional arg")

	// list flags
	lf := starListCmd.Flags()
	lf.IntVarP(&starFlags.limit, "limit", "L", 30, "Maximum number of starred repositories to fetch")
	lf.StringVar(&starFlags.sort, "sort", "created", "Sort key: created | updated (when the star was created vs when the repo was last updated)")

	starCmd.AddCommand(starStarCmd)
	starCmd.AddCommand(starUnstarCmd)
	starCmd.AddCommand(starListCmd)
}

// resolveStarRepoArg returns the (owner, repo) pair for star/unstar,
// preferring the positional argument (e.g. `gitee repo star owner/repo`),
// then the local --repo flag, then the global --repo flag.
func resolveStarRepoArg(args []string) ([2]string, error) {
	// Positional arg wins when present.
	if len(args) > 0 && args[0] != "" {
		return resolveRepoArg(args[0])
	}
	return resolveRepoArg(starFlags.repo)
}

func runRepoStar(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveStarRepoArg(args)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.RepoStar(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// PUT returns 204 No Content; we don't decode a body.
	if err := client.Put(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("starring repository: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s Starred %s/%s — star a repository to support its author\n",
		green("✓"), yellow(owner+"/"+repoName), "")
	return nil
}

func runRepoUnstar(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveStarRepoArg(args)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.RepoUnstar(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("unstarring repository: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s Unstarred %s/%s\n", green("✓"), yellow(owner+"/"+repoName), "")
	return nil
}

func runRepoStarList(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s?sort=%s&per_page=%d",
		api.UserStarredList(), starFlags.sort, starFlags.limit)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repos []types.Repository
	if err := client.Get(ctx, path, &repos); err != nil {
		return fmt.Errorf("listing starred repositories: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(repos, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Println("Repositories you have starred:")
	if len(repos) == 0 {
		fmt.Println("  (none)")
		return nil
	}
	w := output.NewTableWriter()
	fmt.Fprintln(w, "NAME\tLANGUAGE\tSTARS\tDESCRIPTION")
	for _, r := range repos {
		lang := r.Language
		if lang == "" {
			lang = "-"
		}
		desc := r.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			r.FullName, lang, r.StarsCount, utils.Truncate(desc, 60))
	}
	return w.Flush()
}

const repoStarAgentHelp = `# gitee repo star — Agent Help

## Purpose
Top-level umbrella for starring operations: add a star to a Gitee
repository (to bookmark it and signal support for the author), remove an
existing star, or enumerate the repositories you have already starred.

Starring is a personal signal — it does not subscribe you to
notifications. To subscribe to notifications about a repository, use
` + "`" + `gitee repo watch` + "`" + ` (separate verb, separate API path).

## Subcommands
  star [owner/repo]   Star a repository. See ` + "`" + `gitee repo star --agent-help` + "`" + `.
  unstar [owner/repo] Remove a star. See ` + "`" + `gitee repo unstar --agent-help` + "`" + `.
  list                List your starred repos. See
                     ` + "`" + `gitee repo star list --agent-help` + "`" + `.

## Pre-conditions
- Authenticated (gitee auth status)
- For star/unstar: --repo owner/repo (positional arg, --repo flag, or
  global --repo)
- For star: the repo must exist and not already be starred by you
- For unstar: the repo must exist and currently be starred by you
  (otherwise Gitee returns 404, but the post-condition still holds)

## Idempotency
- star:    YES. Re-starring an already-starred repo is a no-op on the
           server.
- unstar:  YES. Re-unstarring an already-unstarred repo returns 404,
           but the post-condition (no star) holds.
- list:    YES. Read-only.

## Atomicity
Each verb is a single API call. Either the star state changes
atomically or nothing changes.

## Side effects
- star:    The repo appears in your starred list; the author may be
           notified by email depending on their settings; the repo's
           star counter on Gitee increments by 1.
- unstar:  The repo disappears from your starred list; the repo's
           star counter on Gitee decrements by 1.
- list:    None.

## Output (stdout, table mode)
- star/unstar: single-line success message with the repo full name
- list:        NAME / LANGUAGE / STARS / DESCRIPTION table

## OAuth scope required
projects (write) for star/unstar; projects (read) for list

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## See also
- ` + "`" + `gitee repo watch` + "`" + `     — subscribe to notifications (independent of star)
- ` + "`" + `gitee repo view` + "`" + `     — inspect a repo's metadata
- ` + "`" + `gitee auth status` + "`" + `  — verify token scope before calling
- ` + "`" + `gitee --agent-help` + "`" + `  — overview of all commands with agent help
`

const repoStarStarAgentHelp = `# gitee repo star — Agent Help

## Purpose
Star a Gitee repository. Star a repository to support its author and
bookmark it to your personal profile. The star counter on the repo
increments by 1 and the repo appears in your starred list (visible to
you at https://gitee.com/{your-username}).

Starring is independent of watching: starring does NOT subscribe you to
notifications. To subscribe to notifications, use ` + "`" + `gitee repo watch` + "`" + `.

## Pre-conditions
- Authenticated (gitee auth status)
- The target repo exists and is visible to the token
- --repo / -R is set (owner/repo) OR the repo is passed as a positional
  argument: ` + "`" + `gitee repo star owner/repo` + "`" + `
- The repo is not already starred by you (idempotent — re-starring is
  a no-op on the server)

## Required flags / arguments
  owner/repo            Repository full name as a positional argument,
                        e.g. ` + "`" + `gitee repo star myorg/myrepo` + "`" + `.
                        Equivalent to ` + "`" + `--repo owner/repo` + "`" + `.

## Optional flags
  --repo / -R           Repository in ` + "`" + `owner/repo` + "`" + ` form. Takes
                        precedence only when the positional argument
                        is omitted; if both are set and disagree, the
                        positional argument wins.

## Idempotency
YES. Re-starring an already-starred repo is a no-op on the server; the
star counter is unchanged. Safe to retry on transient errors.

## Atomicity
Atomic. Either the repo is starred after the call or nothing changed.
There is no partial state.

## Side effects
- The repo appears in your starred list immediately
- The repo's star counter on Gitee increments by 1
- The author may receive an email notification (depends on author
  settings; cannot be suppressed via the API)
- Webhooks subscribed to "Star" or "Watch" events on the repo fire

## Post-conditions (on success)
- Stdout prints "✓ Starred owner/repo — star a repository to support
  its author"
- The repo is now in your starred list

## Output (stdout, table mode)
  ✓ Starred myorg/myrepo — star a repository to support its author

## Common errors
- 401 Unauthorized:  token missing or expired → gitee auth login
- 403 Forbidden:     token lacks scope to star this repo (very rare)
- 404 Not Found:     owner/name doesn't exist; check spelling
- "invalid --repo format": must contain exactly one '/'
- "username is required": must be supplied as positional arg or --repo

## Race condition warning ⚠️
The shared mutable state is YOUR starred set and the repo's public
star counter. Two interleavings: (1) CONCURRENT STAR/UNSTAR: if
another process (or the user in the web UI) toggles the same repo
concurrently, the operations commute but the final state is last-
writer-wins — a bulk "star these 50 repos" loop racing a human
unstarring one of them will silently re-star it. (2) VERIFY GAP:
checking ` + "`" + `gitee repo star list` + "`" + ` first and then starring is
unnecessary and racy — just call this verb; it is idempotent on the
server and the 204 is the check. Note the repo's star COUNTER is a
shared global: its value in ` + "`" + `gitee repo view` + "`" + ` output reflects
everyone's stars, not just yours, and moves constantly — never diff
counter values across two reads to infer your own star state.
Retrying after a transient error is safe.

## Next steps (orchestration hints)
After starring a repository, an agent typically wants to:
1. Verify: gitee repo star list | grep owner/repo
2. Subscribe to notifications (separate verb, separate intent):
   gitee repo watch owner/repo --type watching
3. Clone it locally to start contributing:
   gitee repo clone owner/repo

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Star via positional argument (most concise)
  gitee repo star myorg/myrepo

  # Star via --repo flag (works in scripts where positional parsing
  # is awkward)
  gitee repo star --repo myorg/myrepo

  # Pipeline: star a list of repos from a file
  cat favourites.txt | while read r; do
    gitee repo star "$r"
  done

## See also
- ` + "`" + `gitee repo unstar` + "`" + `  — remove a star
- ` + "`" + `gitee repo star list` + "`" + ` — verify the star was recorded
- ` + "`" + `gitee repo watch` + "`" + `  — subscribe to notifications (independent of star)
- ` + "`" + `gitee --agent-help` + "`" + `  — overview of all commands with agent help
`

const repoStarUnstarAgentHelp = `# gitee repo unstar — Agent Help

## Purpose
Remove a star from a Gitee repository. After this, the repo is no
longer in your personal starred list, and its public star counter
decrements by 1.

## Pre-conditions
- Authenticated (gitee auth status)
- The target repo exists and is visible to the token
- --repo / -R is set (owner/repo) OR the repo is passed as a positional
  argument: ` + "`" + `gitee repo unstar owner/repo` + "`" + `
- The repo is currently starred by you (idempotent — unstarring an
  unstarred repo returns 404, but the post-condition holds)

## Required flags / arguments
  owner/repo            Repository full name as a positional argument,
                        e.g. ` + "`" + `gitee repo unstar myorg/myrepo` + "`" + `.
                        Equivalent to ` + "`" + `--repo owner/repo` + "`" + `.

## Optional flags
  --repo / -R           Repository in ` + "`" + `owner/repo` + "`" + ` form. Takes
                        precedence only when the positional argument
                        is omitted; if both are set and disagree, the
                        positional argument wins.

## Idempotency
YES. Re-unstarring an already-unstarred repo returns 404, but the
post-condition (no star) holds. Safe to retry.

## Atomicity
Atomic. Either the repo is unstarred after the call or nothing changed.

## Side effects
- The repo disappears from your starred list immediately
- The repo's star counter on Gitee decrements by 1
- No email is sent to the author (unlike the star path)
- Webhooks subscribed to "Star" events fire

## Post-conditions (on success)
- Stdout prints "✓ Unstarred owner/repo"
- The repo is no longer in your starred list

## Output (stdout, table mode)
  ✓ Unstarred myorg/myrepo

## Common errors
- 401 Unauthorized:  token missing or expired → gitee auth login
- 403 Forbidden:     token lacks scope to unstar this repo (very rare)
- 404 Not Found:     owner/name doesn't exist, or you have not starred
                     it (idempotent — already unstarred)
- "invalid --repo format": must contain exactly one '/'
- "username is required": must be supplied as positional arg or --repo

## Race condition warning ⚠️
The shared mutable state is YOUR starred set. Concurrent toggles are
last-writer-wins: a bulk unstar loop racing the user re-starring the
same repo in the web UI silently drops their new star. The 404 case
("not currently starred") is benign and idempotent — treat it as
"already in the desired state", not as a failure to retry. List→act
gap: a ` + "`" + `gitee repo star list` + "`" + ` snapshot used to drive a bulk
unstar can be stale (stars added after the list are missed, stars
removed after the list yield harmless 404s) — that is usually fine
for cleanup jobs, but if completeness matters, re-list and loop until
the list is stable. Retrying THIS command is always safe.

## Next steps (orchestration hints)
After unstarring a repository, an agent typically wants to:
1. Verify: gitee repo star list | grep owner/repo
   (should produce no output)
2. If you also want to stop receiving notifications, follow up with:
   gitee repo unwatch owner/repo
3. To leave the project entirely (lose access), see also:
   gitee repo collaborator remove --repo owner/repo <your-username>

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Unstar via positional argument (most concise)
  gitee repo unstar myorg/myrepo

  # Unstar via --repo flag
  gitee repo unstar --repo myorg/myrepo

  # Pipeline: bulk-unstar from a file
  cat to-cleanup.txt | while read r; do
    gitee repo unstar "$r"
  done

## See also
- ` + "`" + `gitee repo star` + "`" + `       — add a star
- ` + "`" + `gitee repo star list` + "`" + ` — verify removal
- ` + "`" + `gitee repo unwatch` + "`" + `   — stop receiving notifications too
- ` + "`" + `gitee --agent-help` + "`" + `  — overview of all commands with agent help
`

const repoStarListAgentHelp = `# gitee repo star list — Agent Help

## Purpose
Enumerate the Gitee repositories you have starred. Use this for
personal inventory, for verifying that a recent ` + "`" + `star` + "`" + ` call succeeded,
or for piping into other commands (e.g. cloning your favourite list).

## Pre-conditions
- Authenticated (gitee auth status)
- API host is reachable (default: gitee.com)

## Optional flags
  --limit / -L    Maximum number of repositories to fetch (default: 30)
  --sort          Sort key: created | updated (default: created)

## Idempotency
YES. Read-only; calling repeatedly yields the same list (assuming no
upstream star/unstar between calls). Safe to retry.

## Atomicity
Atomic per request. A single API call returns one page of starred
repositories. If --limit exceeds per_page (100), the response is
truncated server-side.

## Side effects
None.

## Post-conditions
- Stdout is a plaintext table (default) or JSON array (-o json)
- Process exits 0 even when zero repos are starred (prints "(none)")
- No state change on the server

## Output (stdout, table mode)
  Repositories you have starred:
    NAME                LANGUAGE  STARS  DESCRIPTION
    myorg/myrepo        Go        42     A short description

## Output (stdout, json mode)
  [
    {
      "id": 12345,
      "full_name": "myorg/myrepo",
      "private": false,
      "description": "...",
      "language": "Go",
      "stargazers_count": 42,
      "updated_at": "2026-06-01T12:34:56Z",
      "html_url": "https://gitee.com/myorg/myrepo"
    }
  ]

Field names follow the Gitee API v5 /user/starred response (same shape
as /user/repos). Use ` + "`" + `.full_name` + "`" + ` as input to other commands.

## Common errors
- 401 Unauthorized:  token expired or invalid → gitee auth login
- Network/timeout:   transient; safe to retry

## Race condition warning ⚠️
The shared mutable state is YOUR starred set, which changes whenever
you (or another agent using your token, or a web UI session) star or
unstar a repo. This listing is a point-in-time snapshot — the classic
failure is list→act: piping the output into a bulk clone or bulk
unstar operates on a set that may already have changed (new stars
missed, removed stars causing harmless 404s downstream). The STARS
column is doubly volatile: it counts EVERYONE's stars and moves even
when your own starred set is unchanged — never use counter deltas to
detect your own star/unstar effects. Mitigation: when verifying that
a ` + "`" + `gitee repo star` + "`" + ` call landed, re-run THIS command and grep for
the full_name; for batch jobs, re-list immediately before acting and
tolerate membership drift. Retrying THIS command is always safe
(read-only).

## Next steps (orchestration hints)
After listing your stars, an agent typically wants to:
1. Filter to a specific repo:
   gitee repo star list -o json | jq -e '.[] | select(.full_name == "owner/repo")'
2. Clone a starred repo:
   gitee repo star list -o json | jq -r '.[0].full_name' | xargs gitee repo clone
3. Unstar in bulk:
   gitee repo star list -o json | jq -r '.[].full_name' | xargs -I {} gitee repo unstar {}
4. Subscribe to notifications on a starred repo:
   gitee repo watch owner/repo --type watching

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Plaintext table of your first 30 stars
  gitee repo star list

  # JSON for jq filtering
  gitee repo star list -o json | jq -r '.[].full_name'

  # Clone the most recently starred repo
  TARGET=$(gitee repo star list -o json | jq -r '.[0].full_name')
  gitee repo clone "$TARGET"

## See also
- ` + "`" + `gitee repo star` + "`" + `     — add a star to a repo
- ` + "`" + `gitee repo unstar` + "`" + `   — remove a star
- ` + "`" + `gitee repo watch` + "`" + `    — subscribe to notifications (independent of star)
- ` + "`" + `gitee --agent-help` + "`" + `  — overview of all commands with agent help
`
