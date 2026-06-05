package repo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var syncFlags struct {
	repo   string
	branch string
	force  bool
}

var syncCmd = &cobra.Command{
	Use:   "sync [<owner>/<fork>]",
	Short: "Sync a forked repository with its upstream",
	Long: `Sync a forked repository with its upstream source. This pulls the
latest commits from the upstream repo's default branch (or the branch
specified by --branch) into the fork.

Use this after the upstream has new commits you want to integrate.`,
	Example: `  gitee repo sync my-fork
  gitee repo sync --repo owner/my-fork
  gitee repo sync my-fork --branch develop`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(repoSyncAgentHelp)
			os.Exit(0)
		}
		return nil
	},
	RunE: runSync,
}

func init() {
	f := syncCmd.Flags()
	f.StringVarP(&syncFlags.repo, "repo", "R", "", "Fork repository (owner/repo) to sync")
	f.StringVar(&syncFlags.branch, "branch", "", "Branch to sync (default: upstream's default branch)")
	f.BoolVarP(&syncFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

func runSync(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveRepo("", syncFlags.repo, args)
	if err != nil {
		return err
	}

	if !syncFlags.force {
		branchDesc := syncFlags.branch
		if branchDesc == "" {
			branchDesc = "(upstream default)"
		}
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Sync fork %s/%s with upstream branch %s?", color.YellowString("?"), owner, repoName, branchDesc),
			false,
		)
		if err != nil {
			return err
		}
		if !yes {
			fmt.Println("Cancelled")
			return nil
		}
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReposSync(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Post(ctx, path, nil, nil); err != nil {
		// Gitee returns 422 if there's nothing to sync (fork already up to date).
		// Don't treat that as a fatal error.
		if strings.Contains(err.Error(), "422") {
			fmt.Println("Fork is already up to date with upstream")
			return nil
		}
		return fmt.Errorf("syncing fork: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Synced %s/%s with upstream\n", green("✓"), bold(owner), bold(repoName))
	return nil
}

const repoSyncAgentHelp = `# gitee repo sync — Agent Help

## Purpose
Pull the latest commits from a fork's upstream repository into the fork.
This refreshes the fork to match upstream without manually configuring
a second remote.

## Pre-conditions
- Authenticated with write access to the fork
- The fork was created via ` + "`" + `gitee repo fork` + "`" + ` (i.e. it has an
  upstream relationship in Gitee's database)

## Required arguments
  [<owner>/<fork>]   Fork to sync (positional OR --repo)

## Optional flags
  --repo / -R        Fork repository (alternative to positional)
  --branch           Branch to sync (default: upstream's default branch)
  --force / -f       Skip the y/N confirmation prompt

## Idempotency
YES. Re-running is safe. If the fork is already up to date, the
server returns 422 and the command exits with a friendly message
("Fork is already up to date with upstream").

## Atomicity
NOT atomic. Gitee creates a merge commit on the fork's branch to
integrate upstream's changes. If the merge fails (e.g. conflicts),
the fork may be left in a half-synced state; check ` + "`" + `gitee repo view` + "`" + `
or the web UI to confirm.

## Side effects
- 0..1 new merge commit on the fork's branch
- The fork's HEAD advances to upstream HEAD
- Webhooks subscribed to "Push" / "Pull" events may fire

## Post-conditions (on success)
- The fork's branch contains all upstream commits
- Stdout reports success

## Output (stdout)
  ✓ Synced owner/my-fork with upstream

  OR (if nothing to do):
  Fork is already up to date with upstream

## Common errors
- 404 Not Found: repo isn't actually a fork (or fork relation was lost)
- 403 Forbidden:  token lacks write scope on the fork
- 409 Conflict:   merge conflict (rare; check the web UI)

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour.

## Race condition warning ⚠️
DO NOT run this concurrently with another write to the same fork. The
merge commit ordering is server-determined; concurrent syncs can
result in duplicate merge commits on the fork's history.

## Example
  # Sync your fork with the latest upstream changes
  gitee repo sync myuser/gitee-cli-fork --force

  # Sync a specific branch
  gitee repo sync myuser/gitee-cli-fork --branch develop --force

## See also
- ` + "`" + `gitee repo fork` + "`" + `     — create a fork in the first place
- ` + "`" + `gitee repo view` + "`" + `     — see fork metadata
`
