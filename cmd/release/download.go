package release

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var downloadFlags struct {
	repo     string
	tag      string
	asset    string
	dest     string
	listOnly bool
}

var downloadCmd = &cobra.Command{
	Use:   "download <tag>",
	Short: "Download release assets",
	Example: `  gitee release download v1.0.0 --repo owner/repo --asset gitee-cli-linux-amd64.tar.gz
  gitee release download v1.0.0 -R owner/repo --list
  gitee release download v1.0.0 -R owner/repo --asset foo.tar.gz --dest ./build/`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(releaseDownloadAgentHelp)
		}
		return nil
	},
	RunE: runDownload,
}

func init() {
	f := downloadCmd.Flags()
	f.StringVarP(&downloadFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&downloadFlags.tag, "tag", "t", "", "Tag name (or pass as first arg)")
	f.StringVarP(&downloadFlags.asset, "asset", "a", "", "Asset name to download (filename)")
	f.StringVarP(&downloadFlags.dest, "dest", "d", ".", "Destination directory")
	f.BoolVar(&downloadFlags.listOnly, "list", false, "List assets without downloading")
}

func runDownload(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	tag := downloadFlags.tag
	if tag == "" && len(args) >= 1 {
		tag = args[0]
	}
	if tag == "" {
		return fmt.Errorf("tag is required (use --tag or pass as first arg)")
	}

	owner, repoName, err := resolveReleaseRepo(downloadFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.ReleasesGet(owner, repoName, tag)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var release types.Release
	if err := client.Get(ctx, path, &release); err != nil {
		return fmt.Errorf("getting release: %w", err)
	}

	if len(release.Assets) == 0 {
		fmt.Println("No assets attached to this release")
		return nil
	}

	// --list: just show assets
	if downloadFlags.listOnly {
		bold := color.New(color.Bold).SprintFunc()
		fmt.Printf("%s Assets for %s:\n", bold("─"), release.TagName)
		for _, a := range release.Assets {
			fmt.Printf("  %-40s  %8d bytes  %s\n", a.Name, a.Size, color.CyanString(a.DownloadURL))
		}
		return nil
	}

	// --asset <name>: download the matching one
	if downloadFlags.asset == "" {
		return fmt.Errorf("either --asset or --list is required (see --help for details)")
	}

	var target *types.Asset
	for i := range release.Assets {
		if release.Assets[i].Name == downloadFlags.asset {
			target = &release.Assets[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("asset %q not found in release %s; use --list to see available", downloadFlags.asset, tag)
	}

	destDir := downloadFlags.dest
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating destination dir: %w", err)
	}
	destPath := filepath.Join(destDir, target.Name)

	if err := downloadTo(target.DownloadURL, destPath); err != nil {
		return fmt.Errorf("downloading asset: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Downloaded %s → %s\n", green("✓"), bold(target.Name), color.YellowString(destPath))
	return nil
}

// downloadTo performs a streaming GET to the public download URL.
func downloadTo(rawURL, destPath string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(rawURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}

	// Copy first; on failure Close and remove the partial file so a
	// later run (or script) doesn't mistake a truncated body for the
	// full asset. On success an explicit Close flushes write buffers —
	// a deferred Close would discard any delayed write error.
	_, copyErr := io.Copy(f, resp.Body)
	if closeErr := f.Close(); closeErr != nil {
		// Close error takes precedence: even a "successful" copy is
		// suspect if the file couldn't be flushed to disk.
		if copyErr == nil {
			_ = os.Remove(destPath)
			return fmt.Errorf("closing %s: %w", destPath, closeErr)
		}
		_ = os.Remove(destPath)
		return fmt.Errorf("writing %s: copy=%v close=%w", destPath, copyErr, closeErr)
	}
	if copyErr != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("downloading %s: %w", destPath, copyErr)
	}
	return nil
}

const releaseDownloadAgentHelp = `# gitee release download — Agent Help

## Purpose
List or download assets (binaries, archives) attached to a release. The
underlying Gitee download URLs are public, so no token is required for
the GET itself — but the API call to enumerate assets IS authenticated.

## Pre-conditions
- Authenticated (only needed for --list; the actual download URL is public)
- Release exists and has at least one asset

## Required flags
  --tag / -t  (or first positional arg)   Tag name

## Optional flags
  --repo / -R      owner/repo
  --asset / -a     Asset name to download
  --dest / -d      Destination directory (default: current dir)
  --list           List assets without downloading (mutually exclusive with --asset)

## Idempotency
YES. Re-running downloads overwrite the destination file.

## Atomicity
NOT atomic. If the network drops mid-download, you get a partial file.
A subsequent run overwrites it cleanly.

## Side effects
- File at <dest>/<asset-name> is created or overwritten
- No changes to the Gitee repo

## Post-conditions (on success)
- File exists at the destination, size matches the asset metadata

## Output (stdout, on success)
  ✓ Downloaded gitee-cli-linux-amd64.tar.gz → ./build/

## Common errors
- 404 Not Found: tag has no release, or asset name typo
- 403 Forbidden (download URL): private repo, token not sent (rare for release assets)

## Race condition warning ⚠️
The asset list from the enumerate step is a SNAPSHOT: an asset shown by
--list can be deleted or re-uploaded before your download starts,
leaving you with a 404 or different bytes than expected. If freshness
matters, re-run --list immediately before downloading. Two agents
downloading the same asset into the same --dest directory race on the
destination file (create truncates, writes interleave) — use per-agent
--dest paths. A killed download can leave a truncated file behind;
re-running is safe and overwrites it (see Idempotency).

## OAuth scope required
projects (read) for the list/enumerate step; download URLs themselves are public.

## Rate limit
1000 req/hour for the API call; downloads themselves are CDN-served.

## Example
  # 1. See what assets exist
  gitee release download v1.0.0 -R owner/repo --list

  # 2. Download a specific one
  gitee release download v1.0.0 -R owner/repo -a gitee-cli-linux-amd64.tar.gz -d ./dist/

## See also
- ` + "`" + `gitee release view` + "`" + `   — see body and asset URLs
- ` + "`" + `gitee release list` + "`" + `   — find the tag
`
