package alias

import (
	"fmt"
	"os"
	"text/tabwriter"

	"gitee-cli/cmd"
	"gitee-cli/internal/alias"
	"gitee-cli/internal/handler"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all defined aliases",
	Example: `  gitee alias list
  gitee alias list --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(aliasListAgentHelp)
			os.Exit(0)
		}
		return nil
	},
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
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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

## See also
- ` + "`gitee alias set`" + `    — add or change one
- ` + "`gitee alias delete`" + ` — remove one
`
