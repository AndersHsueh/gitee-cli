package gist

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var cloneFlags struct {
	dir string
}

var cloneCmd = &cobra.Command{
	Use:   "clone <gist-id> [<directory>]",
	Short: "Clone a gist to a local directory",
	Long: `Fetch a gist and place it on the local filesystem.

If the gist is git-backed (the API response includes a git_pull_url),
this runs ` + "`" + `git clone <git_pull_url> <dir>` + "`" + `. Otherwise, it falls back
to downloading every file via raw_url into a local directory.

The default target directory is the gist id itself.`,
	Example: `  gitee gist clone abc123def
  gitee gist clone abc123def ~/snippets/foo
  gitee gist clone abc123def --dir ./my-snippet`,
	Args:    cmd.AgentHelpAwareArgs(cobra.RangeArgs(1, 2)),
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(gistCloneAgentHelp)
		}
		return nil
	},
	RunE: runClone,
}

func init() {
	f := cloneCmd.Flags()
	f.StringVarP(&cloneFlags.dir, "dir", "d", "", "Target directory (default: <gist-id>)")
}

func runClone(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	gistID := args[0]

	dir := cloneFlags.dir
	if dir == "" && len(args) >= 2 {
		dir = args[1]
	}
	if dir == "" {
		dir = gistID
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	// Fetch gist metadata. The id is URL-escaped by the endpoint builder.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var gist types.Gist
	if err := client.Get(ctx, api.GistGet(url.QueryEscape(gistID)), &gist); err != nil {
		return fmt.Errorf("getting gist %s: %w", gistID, err)
	}

	// Branch 1: gist is git-backed → use `git clone` against GitPullURL.
	if gist.GitPullURL != "" {
		if err := gitClone(gist.GitPullURL, dir); err != nil {
			return fmt.Errorf("git clone failed: %w", err)
		}
		green := color.New(color.FgGreen).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()
		fmt.Printf("%s Cloned gist %s to %s (via git)\n", green("✓"), bold(gistID), bold(dir))
		return nil
	}

	// Branch 2: gist is not git-backed → download every file via raw_url.
	if err := downloadGistFiles(client, &gist, dir); err != nil {
		return fmt.Errorf("downloading gist files: %w", err)
	}
	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Downloaded gist %s (%d file(s)) to %s\n",
		green("✓"), bold(gistID), len(gist.Files), bold(dir))
	return nil
}

// gitClone shells out to `git clone` and streams output to the user.
// Equivalent to the function in cmd/pr/checkout.go but kept here as a
// local copy because the pr package is not importable from gist (and
// the 5 lines are not worth a shared utility yet).
func gitClone(url, dir string) error {
	c := exec.Command("git", "clone", url, dir)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// downloadGistFiles fetches every file in the gist and writes it to
// <dir>/<filename>. The directory is created if missing. Uses a fresh
// context per file so a slow file can't blow through a shared deadline.
func downloadGistFiles(client *api.Client, gist *types.Gist, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	for filename, file := range gist.Files {
		if file.RawURL == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
		body, err := client.GetRaw(ctx, file.RawURL)
		cancel()
		if err != nil {
			return fmt.Errorf("downloading %s: %w", filename, err)
		}
		dest := filepath.Join(dir, filename)
		if err := os.WriteFile(dest, body, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dest, err)
		}
	}
	return nil
}

const gistCloneAgentHelp = `# gitee gist clone — Agent Help

## Purpose
Fetch a gist and place it on the local filesystem. Two paths:

  1. **git-backed (preferred)**: if the API response includes a
     ` + "`" + `git_pull_url` + "`" + ` field, this runs ` + "`" + `git clone <url> <dir>` + "`" + ` (gives you
     full version history and pushes).
  2. **fallback (snippets)**: if the gist has no git URL, every file is
     downloaded via ` + "`" + `raw_url` + "`" + ` into a local directory. No version history.

## Pre-conditions
- Authenticated (only to fetch the gist metadata; the actual download
  URL is public, like GitHub)
- A git working copy not required (this creates a fresh dir)
- ` + "`" + `git` + "`" + ` in PATH (only for git-backed gists)

## Required arguments
  <gist-id>     Gist id (or first positional)

## Optional flags
  --dir / -d   Target directory (default: <gist-id>)
                Or pass as second positional: ` + "`" + `gitee gist clone <id> <dir>` + "`" + `

## Idempotency
PARTIAL. The git path fails if <dir> already exists (git's normal
behavior; pass a different --dir or ` + "`" + `rm -rf` + "`" + ` the existing one).
The fallback path overwrites existing files in <dir> without warning.

## Atomicity
NOT atomic. If the network drops mid-clone, you get a partial
directory. Re-run with ` + "`" + `rm -rf <dir>` + "`" + ` first to start clean.

## Side effects
- Creates a local directory (and subdirectory ` + "`" + `.git` + "`" + ` for git-backed gists)
- May write/overwrite files inside <dir>
- Does NOT touch the Gitee API beyond the read-only metadata fetch

## Post-conditions (on success)
- <dir> contains all files in the gist
- For git-backed gists: <dir>/.git is a real repo (pushable if you have
  push access)

## Output (stdout)
  ✓ Cloned gist abc123def to abc123def (via git)
    — OR (non-git fallback) —
  ✓ Downloaded gist abc123def (2 file(s)) to abc123def

## Common errors
- 404 Not Found: gist id doesn't exist or is private
- "git not in PATH": only matters for git-backed gists; install git
  or use the fallback path (it'll happen automatically if the API
  response has no git_pull_url)
- "destination path already exists": the git path's normal error;
  pass --dir to use a fresh path

## Race condition warning ⚠️
The download is a SNAPSHOT at fetch time. If the gist is edited after
clone, your local copy is stale. Re-run the command to refresh.

## Next steps (orchestration hints)
1. ` + "`" + `gitee gist clone <id>` + "`" + ` to fetch
2. ` + "`" + `cd <id>` + "`" + ` and edit files
3. For git-backed: ` + "`" + `git add && git commit && git push` + "`" + `
4. For fallback: just edit files locally; the gist is unchanged on
   Gitee unless you ` + "`" + `gitee gist edit` + "`" + ` it

## OAuth scope required
projects (read) — only used to fetch the gist metadata

## Rate limit
1000 req/hour for the API call; the actual download is CDN-served.

## Example (closing the loop)
  # 1. Clone a snippet
  gitee gist clone abc123def

  # 2. Inspect
  ls abc123def
  cat abc123def/snippet.py

  # 3. Make it yours
  cp abc123def/snippet.py ~/projects/myapp/utils.py

## See also
- ` + "`" + `gitee gist view` + "`" + `   — read the files in stdout without writing to disk
- ` + "`" + `gitee gist list` + "`" + `   — find the id
- ` + "`" + `gitee gist create` + "`" + ` — make a new gist
`
