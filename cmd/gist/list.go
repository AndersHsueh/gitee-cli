package gist

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var listFlags struct {
	limit int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your gists",
	Example: `  gitee gist list
  gitee gist list --limit 20`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gistListAgentHelp)
		}
		return nil
	},
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of gists")
}

func runList(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s?per_page=%d", api.GistsList(), listFlags.limit)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var gists []types.Gist
	if err := client.Get(ctx, path, &gists); err != nil {
		// Gitee API returns 404 when there are no gists, treat as empty result
		if api.IsNotFound(err) {
			gists = []types.Gist{}
		} else {
			return fmt.Errorf("listing gists: %w", err)
		}
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(gists, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(gists) == 0 {
		fmt.Println("No gists found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDESCRIPTION\tPUBLIC\tFILES\tUPDATED")
	for _, g := range gists {
		vis := "public"
		if !g.Public {
			vis = "private"
		}
		desc := utils.Truncate(g.Description, 50)
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			g.ID, desc, vis, len(g.Files), utils.TruncateDate(g.UpdatedAt))
	}
	return w.Flush()
}

// gistListAgentHelp is the agent-targeted help for `gitee gist list`.
// Maintainer note: keep this in sync with the actual command behavior.
const gistListAgentHelp = `# gitee gist list — Agent Help

## Purpose
List the authenticated user's gists (most recently updated first).
Use this when an agent needs to discover which gists exist before
acting on one (e.g. picking an id for ` + "`" + `gitee gist view` + "`" + `, ` + "`" + `gitee gist edit` + "`" + `,
or ` + "`" + `gitee gist delete` + "`" + `).

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The token's owner must have at least one gist, OR the empty-result
  branch will be taken (no error, just "No gists found")

## Optional flags
  --limit / -L   Maximum number of gists to return (default: 30).
                 Caps the per_page query parameter on the API call.

## Idempotency
Pure read. Safe to call repeatedly; the API does not record "last read"
state. The same call on the same dataset returns the same gists
(unless one was created/deleted/starred in between).

## Atomicity
Atomic. Either all gists in the requested page are returned or the
call fails. There is no partial-success state.

## Side effects
None. Read-only call against the API. Does NOT mutate gist state,
does NOT trigger webhooks, does NOT send notifications.

## Post-conditions (on success)
- stdout contains either a tabwriter-formatted table of gists, or
  an empty "No gists found" line, or a JSON array (when -o json)
- exit code 0

## Output (stdout, table mode)
  ID              DESCRIPTION                              PUBLIC    FILES  UPDATED
  abc123def       My useful snippet                        public    1      2025-12-01
  4f5e6d7c        (no description)                         private   3      2025-11-28

Columns: ID, DESCRIPTION (truncated to 50 chars, "-" if empty),
PUBLIC ("public" / "private"), FILES (count), UPDATED (YYYY-MM-DD).

## Output (stdout, json mode)
A JSON array of ` + "`" + `types.Gist` + "`" + ` objects (full API payload, not
projected). Each entry includes id, description, public, files,
updated_at, html_url, etc. Use this when an agent needs raw data
without re-parsing the table.

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     token lacks gist scope   →  check token permissions
- 5xx Server error:  transient                →  retry with backoff

## Next steps (orchestration hints)
After listing, an agent typically wants to:
1. Pick an ` + "`" + `ID` + "`" + ` from the table (or any field in the JSON array)
2. Then:
   - ` + "`" + `gitee gist view <id>` + "`" + `         — read files
   - ` + "`" + `gitee gist edit <id>` + "`" + `         — modify description/files
   - ` + "`" + `gitee gist clone <id>` + "`" + `        — bring it local
   - ` + "`" + `gitee gist delete <id>` + "`" + `       — remove (irreversible)
   - ` + "`" + `gitee gist star <id>` + "`" + `         — add to starred list

## OAuth scope required
projects (read) — or gists (read) on the older token model

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example (closing the loop)
  # 1. List gists
  gitee gist list --limit 5

  # 2. Pick the first id (e.g. "abc123def")

  # 3. View or clone it
  gitee gist view abc123def
  # or
  gitee gist clone abc123def

## See also
- ` + "`" + `gitee gist view` + "`" + `   — read a single gist's files
- ` + "`" + `gitee gist create` + "`" + ` — make a new gist
- ` + "`" + `gitee gist clone` + "`" + `  — fetch a gist to local disk
- ` + "`" + `gitee --agent-help` + "`" + ` — overview of all commands with agent help
`
