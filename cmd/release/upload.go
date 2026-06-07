package release

import (
	"context"
	"fmt"
	"os"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var uploadFlags struct {
	repo  string
	tag   string
	file  string
	label string
}

var uploadCmd = &cobra.Command{
	Use:   "upload <tag>",
	Short: "Upload an asset to a release",
	Long: `Upload a file as a release asset.

The release is identified by its tag name. The file is uploaded
via multipart/form-data to the Gitee v5 API. An optional --label
can be set for the asset's display name.`,
	Example: `  gitee release upload v1.0.0 --repo owner/repo --file ./dist/app.tar.gz
  gitee release upload v1.0.0 -R owner/repo --file ./dist/app.tar.gz --label "Linux binary"`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(releaseUploadAgentHelp)
		}
		return nil
	},
	RunE: runUpload,
}

func init() {
	f := uploadCmd.Flags()
	f.StringVarP(&uploadFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&uploadFlags.tag, "tag", "t", "", "Release tag name (or pass as first arg)")
	f.StringVarP(&uploadFlags.file, "file", "f", "", "Path to the asset file to upload (required)")
	f.StringVar(&uploadFlags.label, "label", "", "Optional display label for the asset")

	ReleaseCmd.AddCommand(uploadCmd)
}

func runUpload(cobraCmd *cobra.Command, args []string) error {
	tag := uploadFlags.tag
	if tag == "" && len(args) >= 1 {
		tag = args[0]
	}
	if tag == "" {
		return fmt.Errorf("tag is required (use --tag or pass as first arg)")
	}
	if uploadFlags.file == "" {
		return fmt.Errorf("--file is required")
	}
	// Verify file exists.
	if _, err := os.Stat(uploadFlags.file); err != nil {
		return fmt.Errorf("file %q: %w", uploadFlags.file, err)
	}

	owner, repoName, err := resolveReleaseRepo(uploadFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// Step 1: Get release to resolve its numeric ID.
	var release types.Release
	if err := client.Get(ctx, api.ReleasesGet(owner, repoName, tag), &release); err != nil {
		return fmt.Errorf("getting release %s: %w", tag, err)
	}

	// Step 2: Upload the asset.
	params := map[string]string{}
	if uploadFlags.label != "" {
		params["label"] = uploadFlags.label
	}

	var asset types.Asset
	uploadPath := api.ReleaseAssetUpload(owner, repoName, release.ID)
	if err := client.UploadFile(ctx, uploadPath, "file", uploadFlags.file, params, &asset); err != nil {
		return fmt.Errorf("uploading asset: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Uploaded %s to release %s\n", green("✓"), bold(asset.Name), bold(tag))
	fmt.Printf("  Size: %d bytes\n", asset.Size)
	if asset.DownloadURL != "" {
		fmt.Printf("  URL:  %s\n", color.YellowString(asset.DownloadURL))
	}
	return nil
}

const releaseUploadAgentHelp = `# gitee release upload — Agent Help

## Purpose
Upload a local file as an asset attached to an existing Gitee release.
The release is resolved by tag name, then the file is POSTed via
multipart/form-data to ` + "`" + `POST /repos/{owner}/{repo}/releases/{id}/assets` + "`" + `.

This is a write command with external side effects (the asset becomes
publicly downloadable).

## Pre-conditions
- Authenticated with a token that has repo write scope
- The target release exists (tag must have an associated release)
- The local file exists and is readable
- Gitee v5 asset upload endpoint is available

## Required flags
  --file / -f         Path to the local file to upload
  --repo / -R         Target repository (owner/repo)
  --tag / -t          Release tag name (or pass as first positional arg)

## Optional flags
  --label             Display label for the asset (defaults to filename)

## Idempotency
NO. Each invocation creates a new asset. Re-running with the same file
will create a duplicate asset entry (different asset ID).

## Atomicity
Not atomic. The release is fetched first (GET), then the file is uploaded
(POST). If the network drops between these steps, the release ID is still
valid but no file was uploaded (safe to retry).

## Side effects
- A new asset appears on the release page on gitee.com
- Subscribers to the repo may receive a notification (depending on
  Gitee notification settings)
- The asset is publicly downloadable immediately

## Post-conditions (on success)
- The file is attached to the specified release
- Stdout prints the asset name, size in bytes, and download URL

## Output (stdout)
  ✓ Uploaded app.tar.gz to release v1.0.0
    Size: 4096 bytes
    URL:  https://gitee.com/owner/repo/attachments/...

## Common errors
- 404 Not Found: the tag has no associated release
- 413 Payload Too Large: file exceeds Gitee's upload size limit
- 422 Unprocessable Entity: file type not allowed or release is in draft state
- "file ... no such file or directory": --file path is invalid

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour authenticated (two API calls per upload: one GET + one POST).

## Example
  gitee release upload v1.0.0 -R myorg/myrepo -f ./dist/app.tar.gz --label "Linux x86_64"
  gitee release upload v1.0.0 -R myorg/myrepo -f ./dist/app.deb

## See also
- ` + "`" + `gitee release view` + "`" + `        — list assets on a release
- ` + "`" + `gitee release delete-asset` + "`" + ` — remove an asset
- ` + "`" + `gitee release create` + "`" + `       — create a new release
`
