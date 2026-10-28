package release

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var deleteAssetFlags struct {
	repo      string
	tag       string
	force     bool
	assetID   int64
	assetName string
}

var deleteAssetCmd = &cobra.Command{
	Use:   "delete-asset <tag>",
	Short: "Delete an asset from a release",
	Long: `Delete an asset from a release by its ID or filename.

The release is identified by its tag name. The asset can be
specified by --asset-id or --asset-name. Use --force to skip
the confirmation prompt.`,
	Example: `  gitee release delete-asset v1.0.0 --repo owner/repo --asset-name app.tar.gz
  gitee release delete-asset v1.0.0 -R owner/repo --asset-id 12345 --force`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(releaseDeleteAssetAgentHelp)
		}
		return nil
	},
	RunE: runDeleteAsset,
}

func init() {
	f := deleteAssetCmd.Flags()
	f.StringVarP(&deleteAssetFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&deleteAssetFlags.tag, "tag", "t", "", "Release tag name (or pass as first arg)")
	f.BoolVar(&deleteAssetFlags.force, "force", false, "Skip confirmation prompt")
	f.Int64Var(&deleteAssetFlags.assetID, "asset-id", 0, "Numeric ID of the asset to delete")
	f.StringVar(&deleteAssetFlags.assetName, "asset-name", "", "Filename of the asset to delete")

	ReleaseCmd.AddCommand(deleteAssetCmd)
}

func runDeleteAsset(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	tag := deleteAssetFlags.tag
	if tag == "" && len(args) >= 1 {
		tag = args[0]
	}
	if tag == "" {
		return fmt.Errorf("tag is required (use --tag or pass as first arg)")
	}
	if deleteAssetFlags.assetID == 0 && deleteAssetFlags.assetName == "" {
		return fmt.Errorf("either --asset-id or --asset-name is required")
	}

	owner, repoName, err := resolveReleaseRepo(deleteAssetFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	// Get release to find numeric ID + assets.
	var release types.Release
	if err := client.Get(ctx, api.ReleasesGet(owner, repoName, tag), &release); err != nil {
		return fmt.Errorf("getting release %s: %w", tag, err)
	}

	// The `assets` array embedded in the release object only carries name +
	// download URL (no numeric ID), so resolve IDs from the attach_files
	// endpoint before matching.
	var assets []types.Asset
	if err := client.Get(ctx, api.ReleaseAssetsList(owner, repoName, release.ID), &assets); err != nil {
		return fmt.Errorf("listing assets of release %s: %w", tag, err)
	}

	// Find target asset.
	var target *types.Asset
	if deleteAssetFlags.assetID != 0 {
		for i := range assets {
			if assets[i].ID == deleteAssetFlags.assetID {
				target = &assets[i]
				break
			}
		}
	} else {
		for i := range assets {
			if assets[i].Name == deleteAssetFlags.assetName {
				target = &assets[i]
				break
			}
		}
	}

	if target == nil {
		msg := "asset"
		if deleteAssetFlags.assetID != 0 {
			msg += fmt.Sprintf(" ID %d", deleteAssetFlags.assetID)
		} else {
			msg += fmt.Sprintf(" %q", deleteAssetFlags.assetName)
		}
		return fmt.Errorf("%s not found in release %s/%s@%s; use `gitee release view %s --repo %s/%s` to see available assets",
			msg, owner, repoName, tag, tag, owner, repoName)
	}

	// Confirmation.
	if !deleteAssetFlags.force {
		prompt := fmt.Sprintf("%s Delete asset %q (%s) from %s/%s@%s?",
			color.YellowString("?"), target.Name, formatAssetSize(target.Size),
			owner, repoName, tag)
		yes, err := utils.ConfirmYN(prompt, false)
		if err != nil {
			return err
		}
		if !yes {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// Delete.
	if err := client.Delete(ctx, api.ReleaseDeleteAsset(owner, repoName, release.ID, target.ID)); err != nil {
		return fmt.Errorf("deleting asset: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Asset %q deleted from release %s\n", green("✓"), target.Name, tag)
	return nil
}

func formatAssetSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

const releaseDeleteAssetAgentHelp = `# gitee release delete-asset — Agent Help

## Purpose
Delete a single asset from an existing Gitee release. The release is
resolved by tag name, then the asset is identified by numeric ID or
filename. A confirmation prompt is shown unless --force is passed.

This is a DESTRUCTIVE write command. Deleted assets cannot be recovered.

## Pre-conditions
- Authenticated with a token that has repo write scope
- The target release exists (tag must have an associated release)
- The target asset exists in the release (by ID or filename)

## Required flags
  --repo / -R            Target repository (owner/repo)
  --tag / -t             Release tag name (or pass as first positional arg)
  --asset-id             Numeric ID of the asset to delete
    OR
  --asset-name           Filename of the asset to delete

## Optional flags
  --force                Skip confirmation prompt (useful for scripts/CI)

## Idempotency
YES (with caveat). Re-running after a successful delete will fail with
"asset not found" — the asset is already gone. This is safe but will
return a non-zero exit code.

## Atomicity
Not strictly atomic. The release is fetched (GET) first, then the asset
is deleted (DELETE). If the network drops between these steps, no harm
is done (the asset still exists; retry).

## Side effects
- The asset is permanently removed from the release
- The download URL becomes invalid immediately
- Subscribers may receive a notification (depending on Gitee settings)

## Post-conditions (on success)
- The asset no longer exists on the release
- Stdout prints confirmation: ` + "`" + `✓ Asset "..." deleted from release ...` + "`" + `

## Common errors
- 404 Not Found: release tag or asset ID not found
- 403 Forbidden: token lacks write scope or user is not a repo collaborator
- Asset not found by name: use ` + "`" + `gitee release view <tag> --repo owner/repo` + "`" + ` to list
  available assets and their IDs, then use --asset-id

## Race condition warning ⚠️
The shared mutable state is the release's asset list. This command
resolves the asset with a GET, then issues DELETE; between the two
steps another agent can delete or re-upload that asset, and the DELETE
then 404s — treat that as "already gone", not as a retryable failure.
A re-uploaded asset gets a NEW numeric ID, so a stale --asset-id never
matches the replacement (and never deletes it by accident). Two agents
deleting the same filename concurrently both resolve the same ID; one
wins, the other 404s safely. Take --asset-id from a fresh
` + "`" + `gitee release view <tag>` + "`" + ` output, not from a view you ran
minutes ago.

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour authenticated (two API calls per delete: one GET + one DELETE).

## Interactive mode
By default, a y/N confirmation prompt is shown:
  ? Delete asset "app.tar.gz" (4.0 KB) from owner/repo@v1.0.0? [y/N]
Use --force to skip this prompt entirely.

## Example
  gitee release delete-asset v1.0.0 -R myorg/myrepo --asset-name app.tar.gz
  gitee release delete-asset v1.0.0 -R myorg/myrepo --asset-id 12345 --force

## See also
- ` + "`" + `gitee release view` + "`" + `        — list all assets on a release
- ` + "`" + `gitee release upload` + "`" + `       — add an asset
- ` + "`" + `gitee release delete` + "`" + `       — delete an entire release
`
