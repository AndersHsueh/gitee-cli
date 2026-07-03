package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Per the P1-6 spec, the watch subcommand group lives entirely in this
// one file (`cmd/repo/watch.go`) and registers two verbs under the
// `gitee repo watch` umbrella:
//   watch    — PUT   /user/subscriptions/{owner}/{repo}  (body: watch_type)
//   unwatch  — DELETE /user/subscriptions/{owner}/{repo}
//
// Both share the --repo / -R flag pattern; watch additionally takes a
// --type flag (watching | releases_only | ignoring; default: watching).

// watchFlags is shared across watch/unwatch so the user can invoke
// `gitee repo watch --repo owner/repo --type watching` (Cobra binds
// by pointer).
var watchFlags struct {
	repo string
	typ  string
}

var watchCmd = &cobra.Command{
	Use:   "watch <subcommand>",
	Short: "Subscribe to or unsubscribe from a repository",
	Long: `Manage notification subscriptions on Gitee repositories.

Watching a repository subscribes you to activity notifications —
issues, PRs, comments, releases, depending on the watch level. Watching
is independent of starring: star to bookmark, watch to subscribe.

  gitee repo watch [owner/repo]   — subscribe to activity
  gitee repo unwatch [owner/repo] — unsubscribe (no notifications)

The watch verb accepts a --type flag with three values:
  watching        — receive all activity notifications (default)
  releases_only   — receive notifications only for new releases
  ignoring        — unsubscribe entirely`,
	Example: `  gitee repo watch myorg/myrepo
  gitee repo watch myorg/myrepo --type releases_only
  gitee repo unwatch myorg/myrepo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoWatchAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		// No subcommand given — show help so users discover watch/unwatch.
		return c.Help()
	},
}

var watchWatchCmd = &cobra.Command{
	Use:   "watch [owner/repo]",
	Short: "Subscribe to a repository's activity",
	Example: `  gitee repo watch myorg/myrepo
  gitee repo watch myorg/myrepo --type releases_only`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoWatchWatchAgentHelp)
		}
		return nil
	},
	RunE: runRepoWatch,
}

var watchUnwatchCmd = &cobra.Command{
	Use:   "unwatch [owner/repo]",
	Short: "Unsubscribe from a repository (no notifications)",
	Example: `  gitee repo unwatch myorg/myrepo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoWatchUnwatchAgentHelp)
		}
		return nil
	},
	RunE: runRepoUnwatch,
}

func init() {
	// watch flags
	wf := watchWatchCmd.Flags()
	wf.StringVarP(&watchFlags.repo, "repo", "R", "", "Repository (owner/repo); falls back to global --repo, then to the positional arg")
	wf.StringVarP(&watchFlags.typ, "type", "t", "watching", "Watch level: watching | releases_only | ignoring")

	// unwatch flags
	uwf := watchUnwatchCmd.Flags()
	uwf.StringVarP(&watchFlags.repo, "repo", "R", "", "Repository (owner/repo); falls back to global --repo, then to the positional arg")

	watchCmd.AddCommand(watchWatchCmd)
	watchCmd.AddCommand(watchUnwatchCmd)
}

// validWatchTypes is the closed set Gitee v5 accepts for the
// `watch_type` body field on watch.
var validWatchTypes = map[string]bool{
	"watching":      true,
	"releases_only": true,
	"ignoring":      true,
}

// resolveWatchRepoArg returns the (owner, repo) pair for watch/unwatch,
// preferring the positional argument, then the local --repo flag, then
// the global --repo flag.
func resolveWatchRepoArg(args []string) ([2]string, error) {
	if len(args) > 0 && args[0] != "" {
		return resolveRepoArg(args[0])
	}
	return resolveRepoArg(watchFlags.repo)
}

func runRepoWatch(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveWatchRepoArg(args)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	typ := strings.ToLower(strings.TrimSpace(watchFlags.typ))
	if !validWatchTypes[typ] {
		return fmt.Errorf("invalid --type %q: must be one of watching, releases_only, ignoring", watchFlags.typ)
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	// Gitee accepts the watch_type as a query parameter (no JSON body
	// for this endpoint — confirmed against the swagger spec). PUT with
	// a body of nil + ?watch_type=... gives us idempotent semantics.
	path := fmt.Sprintf("%s?watch_type=%s", api.RepoWatch(owner, repoName), typ)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Put(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("watching repository: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Watching %s (level: %s)\n",
		green("✓"), bold(yellow(owner+"/"+repoName)), typ)
	return nil
}

func runRepoUnwatch(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveWatchRepoArg(args)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.RepoUnwatch(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("unwatching repository: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s Unwatched %s/%s — no further notifications\n",
		green("✓"), yellow(owner+"/"+repoName), "")
	return nil
}

const repoWatchAgentHelp = `# gitee repo watch — Agent Help

## Purpose
Top-level umbrella for notification subscriptions: subscribe to a
Gitee repository's activity (issues, PRs, comments, releases depending
on the watch level), or unsubscribe.

Watching is independent of starring:
  star    — bookmark a repo to your personal profile
  watch   — subscribe to activity notifications

You can do either, both, or neither. Most users do both for repos
they actively contribute to, and just star for repos they want to
remember.

## Subcommands
  watch [owner/repo]   Subscribe to a repo's activity. See
                      ` + "`" + `gitee repo watch --agent-help` + "`" + `.
  unwatch [owner/repo] Unsubscribe. See ` + "`" + `gitee repo unwatch --agent-help` + "`" + `.

## Watch levels (Gitee v5 watch_type field)
  watching        — full activity (issues, PRs, comments, releases).
                    Default.
  releases_only   — release notifications only. Useful for
                    downstream consumers of a project.
  ignoring        — no notifications. Distinct from "not watched":
                    "not watched" means Gitee uses the repo default,
                    while "ignoring" means explicit opt-out.

## Pre-conditions
- Authenticated (gitee auth status)
- For watch/unwatch: --repo owner/repo (positional arg, --repo flag,
  or global --repo)

## Idempotency
- watch:    YES. Re-watching with the same --type is a no-op on the
            server. Re-watching with a different --type updates the
            watch level in place.
- unwatch:  YES. Re-unwatching an already-unsubscribed repo returns
            204 No Content (Gitee semantics — see gitee repo unwatch
            --agent-help for details).

## Atomicity
Each verb is a single API call. Either the subscription state changes
atomically or nothing changes.

## Side effects
- watch:    Gitee will start sending notifications at the requested
            level (in-app and/or email, depending on user settings).
            Webhooks subscribed to "Watch" events fire.
- unwatch:  Notifications stop. Webhooks subscribed to "Watch" events
            fire.

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## See also
- ` + "`" + `gitee repo star` + "`" + `       — bookmark (independent of watch)
- ` + "`" + `gitee repo unstar` + "`" + `     — remove a bookmark
- ` + "`" + `gitee notification list` + "`" + ` — see the notifications Gitee has queued for you
- ` + "`" + `gitee --agent-help` + "`" + `    — overview of all commands with agent help
`

const repoWatchWatchAgentHelp = `# gitee repo watch — Agent Help

## Purpose
Subscribe to activity notifications on a Gitee repository. Choose the
notification level with --type. Watching is independent of starring;
star to bookmark, watch to subscribe.

## Pre-conditions
- Authenticated (gitee auth status)
- The target repo exists and is visible to the token
- --repo / -R is set (owner/repo) OR the repo is passed as a positional
  argument: ` + "`" + `gitee repo watch owner/repo` + "`" + `

## Required flags / arguments
  owner/repo            Repository full name as a positional argument,
                        e.g. ` + "`" + `gitee repo watch myorg/myrepo` + "`" + `.
                        Equivalent to ` + "`" + `--repo owner/repo` + "`" + `.

## Optional flags
  --repo / -R           Repository in ` + "`" + `owner/repo` + "`" + ` form.
  --type / -t           Watch level: watching | releases_only | ignoring
                        (default: watching).

## Idempotency
YES. Re-watching with the same --type is a no-op on the server.
Re-watching with a DIFFERENT --type updates the watch level in place.

## Atomicity
Atomic. Either the subscription is set after the call or nothing
changed. There is no partial state.

## Side effects
- Gitee starts sending notifications at the requested level (in-app
  inbox at https://gitee.com/notifications and/or email, depending on
  your user settings)
- Webhooks subscribed to "Watch" events on the repo fire

## Post-conditions (on success)
- Stdout prints "✓ Watching owner/repo (level: <type>)"
- The repo is now in your subscriptions at the requested level

## Output (stdout, table mode)
  ✓ Watching myorg/myrepo (level: watching)

## Common errors
- 401 Unauthorized:  token missing or expired → gitee auth login
- 403 Forbidden:     token lacks scope to subscribe
- 404 Not Found:     owner/name doesn't exist; check spelling
- 422 Unprocessable: invalid --type value (also caught client-side)
- "invalid --repo format": must contain exactly one '/'
- "invalid --type": must be one of watching, releases_only, ignoring

## Next steps (orchestration hints)
After watching a repository, an agent typically wants to:
1. Verify: gitee notification list --unread | grep owner/repo
2. Adjust the watch level if needed:
   gitee repo watch owner/repo --type releases_only
3. Stop watching later:
   gitee repo unwatch owner/repo

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Watch with default level (all activity)
  gitee repo watch myorg/myrepo

  # Watch releases only (common for downstream consumers)
  gitee repo watch myorg/myrepo --type releases_only

  # Explicitly ignore (distinct from "not watched")
  gitee repo watch myorg/myrepo --type ignoring

## See also
- ` + "`" + `gitee repo unwatch` + "`" + `       — stop watching
- ` + "`" + `gitee repo star` + "`" + `           — bookmark (independent of watch)
- ` + "`" + `gitee notification list` + "`" + `  — see queued notifications
- ` + "`" + `gitee --agent-help` + "`" + `       — overview of all commands with agent help
`

const repoWatchUnwatchAgentHelp = `# gitee repo unwatch — Agent Help

## Purpose
Unsubscribe from a Gitee repository. After this, Gitee stops sending
notifications for the repo at all levels (you are explicitly removed
from the subscription list).

Note: Gitee v5 returns 204 No Content for both successful unwatch and
unwatch-when-already-unsubscribed. The endpoint is therefore not
strictly "404 if missing" — but the post-condition (no notifications)
holds in both cases.

## Pre-conditions
- Authenticated (gitee auth status)
- --repo / -R is set (owner/repo) OR the repo is passed as a positional
  argument: ` + "`" + `gitee repo unwatch owner/repo` + "`" + `

## Required flags / arguments
  owner/repo            Repository full name as a positional argument,
                        e.g. ` + "`" + `gitee repo unwatch myorg/myrepo` + "`" + `.
                        Equivalent to ` + "`" + `--repo owner/repo` + "`" + `.

## Optional flags
  --repo / -R           Repository in ` + "`" + `owner/repo` + "`" + ` form.

## Idempotency
YES. Re-unwatching an already-unsubscribed repo is a no-op (204 No
Content). Safe to retry.

## Atomicity
Atomic. Either the subscription is removed after the call or nothing
changed.

## Side effects
- Notifications for the repo stop immediately
- Webhooks subscribed to "Watch" events fire
- The repo is removed from your subscriptions list
- The repo's star state is UNCHANGED (star is a separate API path;
  use ` + "`" + `gitee repo unstar` + "`" + ` to remove a bookmark too)

## Post-conditions (on success)
- Stdout prints "✓ Unwatched owner/repo — no further notifications"
- The repo is no longer in your subscriptions

## Output (stdout, table mode)
  ✓ Unwatched myorg/myrepo — no further notifications

## Common errors
- 401 Unauthorized:  token missing or expired → gitee auth login
- 403 Forbidden:     token lacks scope
- 404 Not Found:     owner/name doesn't exist (rare on this endpoint —
                     Gitee usually returns 204 even when not currently
                     subscribed; see "Idempotency" above)
- "invalid --repo format": must contain exactly one '/'

## Next steps (orchestration hints)
After unwatching a repository, an agent typically wants to:
1. Verify: gitee notification list --unread | grep owner/repo
   (should produce no output for this repo)
2. To also remove the bookmark, follow up with:
   gitee repo unstar owner/repo
3. To re-subscribe later at a different level:
   gitee repo watch owner/repo --type releases_only

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Unwatch via positional argument (most concise)
  gitee repo unwatch myorg/myrepo

  # Unwatch via --repo flag
  gitee repo unwatch --repo myorg/myrepo

  # Pipeline: bulk-unwatch from a file
  cat to-silence.txt | while read r; do
    gitee repo unwatch "$r"
  done

## See also
- ` + "`" + `gitee repo watch` + "`" + `     — (re)subscribe
- ` + "`" + `gitee repo unstar` + "`" + `   — remove the bookmark (separate verb)
- ` + "`" + `gitee --agent-help` + "`" + `  — overview of all commands with agent help
`