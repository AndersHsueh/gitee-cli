package release

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var deleteFlags struct {
	repo  string
	tag   string
	force bool
}

var deleteCmd = &cobra.Command{
	Use:   "delete <tag>",
	Short: "Delete a release",
	Example: `  gitee release delete v0.1.0 --repo owner/repo
  gitee release delete v0.1.0 -R owner/repo --force`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(releaseDeleteAgentHelp)
		}
		return nil
	},
	RunE: runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.StringVarP(&deleteFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&deleteFlags.tag, "tag", "t", "", "Tag name (or pass as first arg)")
	f.BoolVarP(&deleteFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

func runDelete(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	tag := deleteFlags.tag
	if tag == "" && len(args) >= 1 {
		tag = args[0]
	}
	if tag == "" {
		return fmt.Errorf("tag is required (use --tag or pass as first arg)")
	}

	owner, repoName, err := resolveReleaseRepo(deleteFlags.repo)
	if err != nil {
		return err
	}

	if !deleteFlags.force {
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Delete release %q from %s/%s?", color.YellowString("?"), tag, owner, repoName),
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

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.ReleaseDelete(owner, repoName, tag)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("deleting release: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Release %q deleted (git tag preserved)\n", green("✓"), tag)
	return nil
}

const releaseDeleteAgentHelp = `# gitee release delete — Agent Help

## Purpose
Remove the release ENVELOPE (the Markdown notes and metadata) from a tag
WITHOUT deleting the underlying git tag. After this command, the git tag
is still on the remote; only the "Release" object attached to it is gone.

## Pre-conditions
- Authenticated with write/admin access
- The release for --tag exists

## Required flags
  --tag / -t  (or first positional arg)   Tag name

## Optional flags
  --repo / -R    owner/repo
  --force / -f   Skip the y/N confirmation prompt (for scripts)

## Idempotency
NO. Re-running on a deleted release returns 404. There is no "create
or update" path on delete; use create if you change your mind.

## Atomicity
Atomic. Either the release is gone, or nothing changed.

## Side effects
- Release page becomes 404 (assets become orphaned, may be GC'd by Gitee)
- Webhooks subscribed to "Release deleted" events fire
- The git tag itself is NOT touched

## Post-conditions (on success)
- The release page no longer exists
- ` + "`" + `git ls-remote --tags origin v1.0.0` + "`" + ` still shows the tag

## Output (stdout)
  ✓ Release "v1.0.0" deleted (git tag preserved)

## Common errors
- 404 Not Found: no release for that tag
- 403 Forbidden:  token lacks admin scope

## Race condition warning ⚠️
The shared mutable state is the release object attached to the tag. Two
agents deleting the same tag concurrently: one succeeds, the other gets
404 — treat that 404 as "already deleted", not as a failure to retry
(see Idempotency). An asset upload in flight while you delete the
release will fail or produce an orphaned asset. After a successful
delete, ` + "`" + `gitee release list` + "`" + ` may still show the release for
1-2 seconds (indexing lag) — do not re-create the release based on
stale list output. If the delete depends on current state (e.g. "only
delete if prerelease"), re-check with ` + "`" + `gitee release view <tag>` + "`" + `
immediately before this call.

## OAuth scope required
projects (admin)

## Rate limit
1000 req/hour.

## Example (with confirmation skip for scripts)
  gitee release delete v0.1.0 -R owner/repo --force

## See also
- ` + "`" + `gitee release view` + "`" + `   — confirm it exists first
- ` + "`" + `gitee release list` + "`" + `   — find the tag name
`
