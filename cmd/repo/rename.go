package repo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var renameFlags struct {
	repo      string
	newName   string
	oldName   string // for resolveRepoArgs positional fallback (1st arg = old, 2nd = new)
}

var renameCmd = &cobra.Command{
	Use:   "rename [<owner>/<old-name>] <new-name>",
	Short: "Rename a repository",
	Long: `Rename a Gitee repository. The new name must be unique within
the owner's namespace. Renaming does NOT change the URL path of
existing clones — you must update your local remotes after a rename.`,
	Example: `  gitee repo rename owner/old-name new-name
  gitee repo rename --repo owner/old-name --new-name new-name`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(repoRenameAgentHelp)
			os.Exit(0)
		}
		return nil
	},
	RunE: runRename,
}

func init() {
	f := renameCmd.Flags()
	f.StringVarP(&renameFlags.repo, "repo", "R", "", "Current repository (owner/old-name)")
	f.StringVar(&renameFlags.newName, "new-name", "", "New repository name")
}

func runRename(cobraCmd *cobra.Command, args []string) error {
	// Two ways to invoke:
	//   1. `gitee repo rename owner/old new`       (positional)
	//   2. `gitee repo rename --repo owner/old --new-name new`
	oldOwner, oldName, newName, err := resolveRenameArgs(args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.RepositoryEditRequest{
		Name: newName,
	}

	path := api.ReposRename(oldOwner, oldName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Patch(ctx, path, req, &repo); err != nil {
		return fmt.Errorf("renaming repository: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s Renamed %s/%s → %s/%s\n", green("✓"), bold(oldOwner), bold(oldName), bold(oldOwner), bold(newName))
	fmt.Printf("  New URL:  %s\n", color.CyanString(repo.HTMLURL))
	fmt.Printf("  %s Update your local remotes:\n", yellow("!"))
	fmt.Printf("     git remote set-url origin %s\n", repo.HTMLURL)
	return nil
}

// resolveRenameArgs accepts the (1) positional or --repo form, and
// (2) positional or --new-name form. Returns (owner, oldName, newName, err).
func resolveRenameArgs(args []string) (string, string, string, error) {
	// Path 1: --repo + --new-name
	if renameFlags.repo != "" {
		if renameFlags.newName == "" {
			return "", "", "", fmt.Errorf("--new-name is required when --repo is used")
		}
		owner, name, ok := parseOwnerRepoStrict(renameFlags.repo)
		if !ok {
			return "", "", "", fmt.Errorf("invalid --repo format %q, expected owner/name", renameFlags.repo)
		}
		return owner, name, renameFlags.newName, nil
	}
	// Path 2: positional `owner/old new`
	if len(args) == 2 {
		owner, name, ok := parseOwnerRepoStrict(args[0])
		if !ok {
			return "", "", "", fmt.Errorf("invalid first arg %q, expected owner/old-name", args[0])
		}
		return owner, name, args[1], nil
	}
	return "", "", "", fmt.Errorf("usage: gitee repo rename <owner>/<old> <new>\n   or: gitee repo rename --repo <owner>/<old> --new-name <new>")
}

// parseOwnerRepoStrict is the strict variant of utils.ParseOwnerRepo: it
// requires a single "/" with non-empty parts on both sides, and returns
// (owner, name, ok) instead of (owner, name). Used for the rename command,
// which must reject malformed input.
func parseOwnerRepoStrict(s string) (string, string, bool) {
	idx := strings.Index(s, "/")
	if idx <= 0 || idx == len(s)-1 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}

const repoRenameAgentHelp = `# gitee repo rename — Agent Help

## Purpose
Change a repository's name in place. The new name must be unique
within the owner's namespace. After renaming, you must update your
local git remotes (the URL path changes; the old URL 404s).

## Pre-conditions
- Authenticated with admin access to the repo
- The new name is NOT already taken in the same owner's namespace

## Required arguments
Two equivalent forms:

  Form 1 (positional):
    gitee repo rename <owner>/<old-name> <new-name>

  Form 2 (flags):
    gitee repo rename --repo <owner>/<old-name> --new-name <new-name>

## Idempotency
NO. Re-running on a renamed repo returns 422 (new-name already taken
because the OLD repo is now named new-name). The new repo can't be
"renamed to itself". Use ` + "`" + `gitee repo edit` + "`" + ` for in-place tweaks
that ARE idempotent.

## Atomicity
Atomic. Either the rename happens, or no change. There is no
"intermediate" name state.

## Side effects
- The repo's name changes; the old URL starts returning 404
- All forks automatically point at the new URL
- The web UI redirects from old URL to new URL
- Local clones DO NOT auto-redirect — you must update the remote

## Post-conditions (on success)
- The repo has the new name (same owner)
- Old URL returns 404 (web UI shows the new URL on the old path)
- Local clones still work against the OLD remote URL until you update

## Output (stdout)
  ✓ Renamed owner/old-name → owner/new-name
    New URL:  https://gitee.com/owner/new-name
    ! Update your local remotes:
       git remote set-url origin https://gitee.com/owner/new-name

## Common errors
- 404 Not Found: source repo doesn't exist
- 422 Unprocessable: new name is taken or invalid (e.g. contains /)
- 403 Forbidden:  token lacks admin scope

## OAuth scope required
projects (admin)

## Rate limit
1000 req/hour.

## Race condition warning ⚠️
Other tools or scripts that cached the old URL will break. Update
CI configs, deployment scripts, and any docs that hardcode the
repo URL.

## Next steps (orchestration hints)
1. ` + "`" + `gitee repo rename --repo owner/old --new-name new` + "`" + `
2. Update local clones:
   ` + "`" + `git remote set-url origin https://gitee.com/owner/new` + "`" + `
3. Update CI / deployment scripts that hardcode the old URL
4. Update README links

## Example
  gitee repo rename myorg/old-name new-name

  # Then on each clone:
  git remote set-url origin https://gitee.com/myorg/new-name

## See also
- ` + "`" + `gitee repo view` + "`" + `   — confirm the new state
- ` + "`" + `gitee repo edit` + "`" + `   — non-name changes (description, visibility)
- ` + "`" + `gitee repo archive` + "`" + ` — soft delete
`
