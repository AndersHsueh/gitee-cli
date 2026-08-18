package alias

import (
	"fmt"

	"gitee-cli/cmd"
	"gitee-cli/internal/alias"
	"gitee-cli/internal/handler"
	"gitee-cli/internal/output"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all defined aliases",
	Example: `  gitee alias list
  gitee alias list --output json`,
	PreRunE: preRunAgentHelp(aliasListAgentHelp),
	RunE: func(c *cobra.Command, args []string) error {
		keys, m, err := alias.Sorted()
		if err != nil {
			return fmt.Errorf("listing aliases: %w", err)
		}
		if len(keys) == 0 {
			fmt.Println("No aliases defined. Use 'gitee alias set <name> <expansion>' to add one.")
			return nil
		}
		if cmd.GlobalOutput == "json" {
			// Emit the map itself, not the sorted-keys slice, so the
			// JSON output is the canonical storage shape.
			return handler.PrintJSON(m)
		}
		w := output.NewTableWriter()
		fmt.Fprintln(w, "NAME\tEXPANSION")
		for _, k := range keys {
			fmt.Fprintf(w, "%s\t%s\n", k, m[k])
		}
		return w.Flush()
	},
}

const aliasListAgentHelp = `# gitee alias list — Agent Help

## Purpose
Enumerate every alias currently defined in ~/.gitee-cli/aliases.json.

## Pre-conditions
- None (no auth, no network)

## Optional flags
  (inherits global --output / -o)

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None.

## Post-conditions (on success)
- Stdout contains a table of aliases (NAME, EXPANSION)
- or JSON object {"<name>": "<expansion>"}

## Output (stdout, table mode)
  NAME       EXPANSION
  co         pr checkout
  issues     issue list

## Output (stdout, json mode)
  {"co":"pr checkout","issues":"issue list"}

## Common errors
- (none — empty alias set returns "No aliases defined")

## Race condition warning ⚠️
None known for corruption — this command only reads
~/.gitee-cli/aliases.json. But the output is a point-in-time snapshot
of shared mutable state: a concurrent ` + "`gitee alias set`" + ` or
` + "`gitee alias delete`" + ` (from you in another shell, or another agent)
can change the file the instant after you read it. If you plan to act
on an entry — delete it, or rely on it expanding — re-run this list
immediately before acting, or just attempt the action and handle
"is not defined". Do not trust a list captured in an earlier step of
a long workflow.

## OAuth scope required
N/A (local command) — no Gitee API request is made; works fully offline.

## Rate limit
N/A (local command) — no Gitee API request is made.

## Example
  gitee alias list
  # NAME    EXPANSION
  # co      pr checkout
  gitee alias list -o json
  # {"co":"pr checkout"}

## See also
- ` + "`gitee alias set`" + `    — add or change one
- ` + "`gitee alias delete`" + ` — remove one
`
