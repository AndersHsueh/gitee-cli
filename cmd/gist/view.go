package gist

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var viewFlags struct {
	raw bool
}

var viewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a gist",
	Example: `  gitee gist view abc123def
  gitee gist view abc123def --raw`,
	Args: cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(gistViewAgentHelp)
		}
		return nil
	},
	RunE: runView,
}

func init() {
	viewCmd.Flags().BoolVar(&viewFlags.raw, "raw", false, "Output raw file content")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	gistID := args[0]

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.GistGet(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var gist types.Gist
	if err := client.Get(ctx, path, &gist); err != nil {
		return fmt.Errorf("viewing gist: %w", err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(gist, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("ID: %s\n", gist.ID)
	fmt.Printf("Description: %s\n", gist.Description)
	fmt.Printf("Public: %v\n", gist.Public)
	fmt.Printf("Files: %d\n", len(gist.Files))
	fmt.Printf("URL: %s\n", gist.HTMLURL)

	for name, file := range gist.Files {
		fmt.Printf("\n--- %s (%s, %d bytes) ---\n", name, file.Language, file.Size)
		if viewFlags.raw {
			// Get raw content from raw_url
			if file.RawURL != "" {
				rawResp, err := client.GetRaw(ctx, file.RawURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: fetching raw content for %s: %v\n", name, err)
				} else {
					fmt.Print(string(rawResp))
				}
			}
		} else {
			// Show truncated content
			content := file.Content
			if content == "" && file.RawURL != "" {
				rawResp, err := client.GetRaw(ctx, file.RawURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: fetching raw content for %s: %v\n", name, err)
				} else {
					content = string(rawResp)
				}
			}
			content = utils.Truncate(content, 100)
			fmt.Println(content)
		}
	}
	return nil
}

// gistViewAgentHelp is the agent-targeted help for `gitee gist view`.
// Maintainer note: keep this in sync with the actual command behavior.
const gistViewAgentHelp = `# gitee gist view — Agent Help

## Purpose
Read a single gist: metadata (id, description, visibility) plus the
content of every file in the gist. Use this when an agent needs to
inspect a gist's contents without writing anything to disk (use
` + "`" + `gitee gist clone` + "`" + ` when you want a local copy).

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The gist id must exist. Public gists are readable by anyone;
  private gists require the owner (or an explicit grant)

## Required arguments
  <id>   Gist id (positional, required)

## Optional flags
  --raw   Output raw file content (skips truncation; fetches via raw_url
          for the full file body). Without this, content is truncated to
          ~100 chars per file as a preview.

## Idempotency
Pure read. Safe to call repeatedly; returns the same data each time
for the same gist id (unless the gist was edited in between).

## Atomicity
Atomic. Either the full gist metadata + every file is returned or
the call fails. There is no partial-success state.

## Side effects
None. Read-only call against the API. Does NOT mutate gist state,
does NOT trigger webhooks, does NOT send notifications.

## Post-conditions (on success)
- stdout contains metadata + every file (table mode) OR full JSON
  payload (json mode)
- exit code 0

## Output (stdout, table mode)
  ID: abc123def
  Description: My snippet
  Public: true
  Files: 2
  URL: https://gitee.com/...

  --- main.go (Go, 142 bytes) ---
  package main
  ...

  --- README.md (Markdown, 38 bytes) ---
  # hello

Note: the per-file content is TRUNCATED by default. Use --raw (or
-o json) to get the full body.

## Output (stdout, json mode)
A single ` + "`" + `types.Gist` + "`" + ` object (full API payload). The ` + "`" + `files` + "`" + ` map
contains every file with ` + "`" + `content` + "`" + ` populated when the API
returns it inline (small files), or just ` + "`" + `raw_url` + "`" + ` for
larger files (use the raw_url to fetch the body).

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     gist is private and the token can't read it
- 404 Not Found:     gist id is wrong or was deleted  →  gitee gist list
- 5xx Server error:  transient                →  retry with backoff

## Race condition warning ⚠️
Pure read, so nothing here mutates state — but the output is a
snapshot of a gist others can edit or delete. Two concrete gaps:
1. The gist can change between this call and any follow-up action
   (edit / comment / delete). If the decision depends on current
   content, re-run ` + "`" + `gitee gist view` + "`" + ` immediately before acting.
2. With --raw (or the truncated fallback), each file body is fetched
   via its raw_url AFTER the metadata call. An edit landing mid-run
   means different files can come from different revisions of the
   same gist. For a consistent single-shot snapshot, use -o json
   (one API call) or ` + "`" + `gitee gist clone` + "`" + `.

## Next steps (orchestration hints)
After viewing, an agent typically wants to:
1. ` + "`" + `gitee gist clone <id>` + "`" + `   — bring it local for editing
2. ` + "`" + `gitee gist edit <id>` + "`" + `    — change description or file contents
3. ` + "`" + `gitee gist star <id>` + "`" + `     — bookmark it
4. ` + "`" + `gitee gist delete <id>` + "`" + `   — remove (irreversible)

## OAuth scope required
projects (read) — or gists (read) on the older token model

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
The table-mode view issues 1 + N raw_url fetches (one per file);
the JSON mode is a single API call.

## Example (closing the loop)
  # 1. Read full content as JSON for parsing
  gitee gist view abc123def -o json

  # 2. Or read raw file bodies (no truncation)
  gitee gist view abc123def --raw

  # 3. Then take it local
  gitee gist clone abc123def

## See also
- ` + "`" + `gitee gist list` + "`" + `   — discover ids
- ` + "`" + `gitee gist clone` + "`" + `  — fetch to local disk
- ` + "`" + `gitee gist edit` + "`" + `   — modify a gist
- ` + "`" + `gitee --agent-help` + "`" + ` — overview of all commands with agent help
`
