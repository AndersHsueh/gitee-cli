package gist

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

var editFlags struct {
	description string
	files       []string
}

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a gist",
	Example: `  gitee gist edit abc123 --description "New description"
  gitee gist edit abc123 --file readme.md="New README content"
  gitee gist edit abc123 --file file1.txt="Content 1" --file file2.txt="@/path/to/file2.txt"
  gitee gist edit abc123 --file readme.md="Updated" --file newfile.md="New file"`,
	Args:    cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gistEditAgentHelp)
		}
		return nil
	},
	RunE: runEdit,
}

func init() {
	f := editCmd.Flags()
	f.StringVar(&editFlags.description, "description", "", "New description")
	f.StringArrayVar(&editFlags.files, "file", []string{}, "File to edit: filename=content or filename=@filepath (can be specified multiple times)")
}

func runEdit(cobraCmd *cobra.Command, args []string) error {
	gistID := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.GistEditRequest{
		Description: editFlags.description,
	}

	// Parse files
	if len(editFlags.files) > 0 {
		req.Files = make(map[string]types.GistEditFile)

		for _, fileFlag := range editFlags.files {
			filename, content, err := utils.ParseGistFileFlag(fileFlag)
			if err != nil {
				return err
			}
			req.Files[filename] = types.GistEditFile{
				Content: content,
			}
		}
	}

	path := api.GistEdit(gistID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var result types.Gist
	if err := client.Patch(ctx, path, req, &result); err != nil {
		return fmt.Errorf("editing gist: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Gist %s updated\n", green("✓"), gistID)

	// Show updated files if --file was used
	if len(editFlags.files) > 0 {
		fmt.Println("\nUpdated files:")
		for filename := range result.Files {
			fmt.Printf("  - %s\n", filename)
		}
	}

	return nil
}

// gistEditAgentHelp is the agent-targeted help for `gitee gist edit`.
// Maintainer note: keep this in sync with the actual command behavior.
const gistEditAgentHelp = `# gitee gist edit — Agent Help

## Purpose
Modify an existing gist: change its description and/or replace/add/remove
files. Use this when an agent needs to update a snippet it (or the
user) previously created — e.g. fixing a typo, updating a script, or
renaming a file.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The gist must exist (verify with: gitee gist list or gitee gist view)
- The token's owner must have write access to the gist (i.e. own it)

## Required arguments
  <id>   Gist id (positional, required)

## Optional flags
  --description   New description (replaces the existing one)
  --file          One file entry. Format:
                    "name=content"       update / add a file
                    "name=@/path"        update / add from a local file
                  Repeatable. Each entry is a complete file spec; files
                  not mentioned are LEFT UNCHANGED.

## Idempotency
IDEMPOTENT for the file+description state. Re-running the same call
with the same flags produces the same end state (a no-op against
the API). NOT idempotent for "no-op detection": the server records
the update regardless of whether content actually changed.

## Atomicity
Atomic. Either every field in the request is applied or none of
them are. There is no partial-success state.

## Side effects
- Gist description and/or file contents change on the server
- The gist's ` + "`" + `updated_at` + "`" + ` timestamp advances
- Webhooks subscribed to "Gist" events may fire
- The old file content for any updated file is LOST (the gist has
  no version history; consider ` + "`" + `gitee gist clone` + "`" + ` first if
  you need to roll back)

## Post-conditions (on success)
- The gist exists with the new description and/or new file contents
- stdout includes "Gist <id> updated" and the list of files
- exit code 0

## Output (stdout, table mode)
  ✓ Gist abc123 updated

  Updated files:
    - readme.md

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     token is not the gist owner  →  cannot edit
- 404 Not Found:     gist id is wrong or was deleted  →  gitee gist list
- 422 Unprocessable: payload rejected          →  check filenames + content
- "invalid --file format": missing "=" in a --file value

## Race condition warning ⚠️
The gist's description and file contents are shared mutable state,
and edit is LAST-WRITER-WINS with whole-file replacement:
- Two writers editing the SAME file concurrently do not get merged —
  the later call silently overwrites the earlier one's content.
  (Files not named in --file are preserved.) Serialize edits in
  multi-agent setups.
- Content captured by an earlier ` + "`" + `gitee gist view` + "`" + ` or
  ` + "`" + `gitee gist clone` + "`" + ` is stale the moment this call succeeds —
  re-read before building on old content.
- Retrying with the same flags after a timeout/5xx is safe: the
  call is idempotent for the same end state (only ` + "`" + `updated_at` + "`" + `
  advances on a no-change re-apply).

## Next steps (orchestration hints)
After a successful edit, an agent typically wants to:
1. ` + "`" + `gitee gist view <id>` + "`" + `    — verify the new content
2. ` + "`" + `gitee gist clone <id>` + "`" + `   — pull a fresh local copy
3. ` + "`" + `gitee gist delete <id>` + "`" + `  — remove (irreversible; only if
   the gist is no longer needed)

## OAuth scope required
projects (write) — or gists (write) on the older token model

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example (agent-friendly, closing the loop)
  # 1. Update description only
  gitee gist edit abc123def --description "Updated repro for issue #42"

  # 2. Update one file inline
  gitee gist edit abc123def --file main.go="package main
func main() { println(\"hi\") }"

  # 3. Update one file from disk and add a new one
  gitee gist edit abc123def \
    --file main.go=@./main.go \
    --file CHANGELOG.md="# Changelog
- fixed crash on startup"

  # 4. Verify
  gitee gist view abc123def

## See also
- ` + "`" + `gitee gist view` + "`" + `    — read a gist (before/after edit)
- ` + "`" + `gitee gist create` + "`" + `  — make a NEW gist (not this one)
- ` + "`" + `gitee gist clone` + "`" + `   — pull a local copy first if you
                       want a rollback path
- ` + "`" + `gitee --agent-help` + "`" + ` — overview of all commands with agent help
`
