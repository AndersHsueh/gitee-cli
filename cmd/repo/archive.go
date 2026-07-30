package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var archiveFlags struct {
	repo      string
	unarchive bool
	force     bool
}

var archiveCmd = &cobra.Command{
	Use:   "archive [<owner>/<repo>]",
	Short: "Archive a repository (or unarchive with --unarchive)",
	Long: `Archive a repository. Archived repos become read-only by default and
are hidden from public listings. Use --unarchive to restore.

This is a soft delete: issues, PRs, wiki, and code are all preserved,
just hidden from the main repo list and protected from new pushes.`,
	Example: `  gitee repo archive owner/repo
  gitee repo archive --repo owner/repo
  gitee repo archive owner/repo --unarchive`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoArchiveAgentHelp)
		}
		return nil
	},
	RunE: runArchive,
}

func init() {
	f := archiveCmd.Flags()
	f.StringVarP(&archiveFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.BoolVar(&archiveFlags.unarchive, "unarchive", false, "Restore the repo from archived state")
	f.BoolVarP(&archiveFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

func runArchive(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveRepo("", archiveFlags.repo, args)
	if err != nil {
		return err
	}

	action := "archive"
	if archiveFlags.unarchive {
		action = "unarchive"
	}

	if !archiveFlags.force {
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s %s %s/%s?", color.YellowString("?"), action, owner, repoName),
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

	req := types.RepositoryEditRequest{
		// Gitee API v5 rejects PATCH /repos/{owner}/{repo} with HTTP 400
		// "name is missing" when the body omits `name` (issue #IK5FYW).
		// Always send the current repo name; archive operation doesn't
		// rename, so this just keeps the payload valid.
		Name:     repoName,
		Archived: utils.BoolPtr(!archiveFlags.unarchive), // true for archive, false for unarchive
	}

	path := api.ReposArchive(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Patch(ctx, path, req, &repo); err != nil {
		return fmt.Errorf("%sing repository: %w", action, err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	actionLabel := strings.ToUpper(action[:1]) + action[1:]
	fmt.Printf("%s %s %s/%s\n", green("✓"), actionLabel, bold(owner), bold(repoName))
	if repo.HTMLURL != "" {
		fmt.Printf("  URL: %s\n", color.CyanString(repo.HTMLURL))
	}
	return nil
}

const repoArchiveAgentHelp = `# gitee repo archive — Agent Help

## Purpose
Archive (or unarchive) a repository. Archiving makes the repo read-only
and hides it from public listings. Code, issues, PRs, and wiki are
preserved; new pushes are blocked until unarchived.

## Pre-conditions
- Authenticated with admin access
- The target repo exists

## Required arguments
  [<owner>/<repo>]   Positional (or use --repo). One is required.

## Optional flags
  --repo / -R        owner/repo (alternative to positional)
  --unarchive        Restore the repo (default: archive)
  --force / -f       Skip the y/N confirmation prompt

## Idempotency
YES. Re-running archive on an already-archived repo is a no-op
(server returns the same state). Same for unarchive.

## Atomicity
Atomic. Either the archived flag flips, or no change happens.

## Side effects
- Repo becomes read-only (archive) or read-write (unarchive)
- Webhooks subscribed to repository events may fire
- The repo is hidden from the public listing (archive only)

## Post-conditions (on success)
- Repo state on Gitee reflects the new archive flag
- Stdout reports success and the HTML URL

## Output (stdout)
  ✓ Archive owner/repo
    URL: https://gitee.com/owner/repo

## Common errors
- 404 Not Found: repo doesn't exist or is private and unreadable
- 403 Forbidden:  token lacks admin scope
- 401 Unauthorized: token expired

## OAuth scope required
projects (admin)

## Rate limit
1000 req/hour.

## Race condition warning ⚠️
DO NOT immediately push code after unarchive. Wait a few seconds for
Gitee's CDN to propagate the new state; pushing too fast can result
in a 403 "repo is archived" error on the push.

## Example
  # Archive a finished project
  gitee repo archive myorg/legacy-app --force

  # Restore it later
  gitee repo archive myorg/legacy-app --unarchive --force

## See also
- ` + "`" + `gitee repo view` + "`" + `     — check current archive status
- ` + "`" + `gitee repo delete` + "`" + `   — hard delete (irreversible)
- ` + "`" + `gitee repo rename` + "`" + `   — change the repo name
`
