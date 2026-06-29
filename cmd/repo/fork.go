package repo

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var forkFlags struct {
	owner string
	repo  string
}

var forkCmd = &cobra.Command{
	Use:   "fork [<owner>/<repo>]",
	Short: "Fork a repository",
	Example: `  gitee repo fork owner/repo
  gitee repo fork --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoForkAgentHelp)
		}
		return nil
	},
	RunE: runFork,
}

func init() {
	f := forkCmd.Flags()
	f.StringVar(&forkFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&forkFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runFork(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	owner, repoName, err := resolveRepo(forkFlags.owner, forkFlags.repo, args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReposFork(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Post(ctx, path, nil, &repo); err != nil {
		return fmt.Errorf("forking repository: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Forked %s/%s to %s\n", green("✓"), bold(owner), bold(repoName), repo.FullName)
	fmt.Printf("  URL: %s\n", color.CyanString(repo.HTMLURL))
	return nil
}

const repoForkAgentHelp = `# gitee repo fork — Agent Help

## Purpose
Fork an existing repository into the authenticated user's own account.
Use this when an agent needs to make a derivative work — typically
before opening a PR back upstream, or before applying local experiments
that may not be welcome on the source repo.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The source repo exists and is visible to the token
  (public forks are always allowed; private forks require source access)
- The token's user does NOT already have a fork with the same name
  in the same scope — rename existing fork first if needed

## Optional flags
  --owner         Source repository owner (when --repo is just the bare name)
  --repo / -R     Source repository name, OR owner/name; overrides positional arg

## Idempotency
NO (server side idempotency may exist, but it is NOT a "no-op").
If the target fork name already exists under the token user, the API
returns an error rather than silently succeeding. Check first with
` + "`" + `gitee repo list` + "`" + ` to detect a pre-existing fork.

## Atomicity
Atomic per request. The fork POST is a single API call — either the new
fork exists, or it doesn't. There is no partial state.

## Side effects
- A new repository is created under the token user's namespace
  (FullName will be ` + "`" + `<my-user>/<repo>` + "`" + `)
- The new fork's default branch matches the source's default branch at
  fork time; subsequent pushes do NOT auto-sync (use ` + "`" + `gitee repo sync` + "`" + `)
- Source repo records a "forked from" parent pointer
- Webhooks subscribed to "Fork" events on the source may fire

## Post-conditions (on success)
- Stdout reports ` + "`" + `✓ Forked owner/name to <my-user>/name` + "`" + ` plus the URL
- The new fork exists at ` + "`" + `https://gitee.com/<my-user>/name` + "`" + `
- Source repo's forks_count may increment
- No git remote is added locally — run ` + "`" + `gitee repo clone` + "`" + ` next

## Common errors
- 401 Unauthorized:  token expired or invalid                →  gitee auth login
- 403 Forbidden:     source is private and not readable
- 404 Not Found:     source owner/name doesn't exist
- 422 Unprocessable: a fork with the same name already exists,
                     or fork limit has been reached on this account
- Network/timeout:   transient; safe to retry but verify no duplicate was created

## Next steps (orchestration hints)
After a successful fork, an agent typically wants to:
1. Clone the fork locally: ` + "`" + `gitee repo clone <my-user>/name` + "`" + `
2. Add the source as a second remote:
   ` + "`" + `git remote add upstream https://gitee.com/<source-owner>/name` + "`" + `
3. Make a feature branch and commit changes locally
4. Push to the fork:        ` + "`" + `git push origin <branch>` + "`" + `
5. Open a PR upstream:
   ` + "`" + `gitee pr create --repo <source-owner>/name --head <my-user>:<branch> ...` + "`" + `
6. Keep the fork current:   ` + "`" + `gitee repo sync <my-user>/name` + "`" + `

## OAuth scope required
projects (write) on the source repo (for private forks), or basic
public-read for public sources.

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Fork via positional arg
  gitee repo fork gitee/laboratory

  # Fork via flag, then clone to local disk
  gitee repo fork --repo gitee/laboratory
  gitee repo clone <my-user>/laboratory

  # Detect pre-existing fork before forking again
  gitee repo list --owner <my-user> -o json \\
    | jq -e --arg n "$NAME" '.[] | select(.name == $n)' >/dev/null \\
    && echo "fork already exists" || gitee repo fork --repo up/name

## See also
- ` + "`" + `gitee repo clone` + "`" + `            — clone the fork (or the source)
- ` + "`" + `gitee repo sync` + "`" + `            — sync fork with upstream
- ` + "`" + `gitee pr create` + "`" + `            — open a PR upstream from a fork
- ` + "`" + `gitee repo view` + "`" + `            — inspect the source first
- ` + "`" + `gitee --agent-help` + "`" + `          — overview of all commands with agent help
`
