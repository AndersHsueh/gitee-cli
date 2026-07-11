package extension

import (
	"fmt"

	"gitee-cli/cmd"
	"gitee-cli/internal/extension"
	"gitee-cli/internal/handler"
	"gitee-cli/internal/output"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed extensions",
	Example: `  gitee extension list
  gitee extension list --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(extensionListAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		manifests, err := extension.ListInstalled()
		if err != nil {
			return fmt.Errorf("listing extensions: %w", err)
		}
		if len(manifests) == 0 {
			fmt.Println("No extensions installed. Use 'gitee extension install <owner/repo>' to add one.")
			return nil
		}
		if cmd.GlobalOutput == "json" {
			return handler.PrintJSON(manifests)
		}
		w := output.NewTableWriter()
		fmt.Fprintln(w, "NAME\tSOURCE\tINSTALLED")
		for _, m := range manifests {
			fmt.Fprintf(w, "%s\t%s\t%s\n", m.Name, m.Source, m.InstalledAt.Format("2006-01-02"))
		}
		return w.Flush()
	},
}

const extensionListAgentHelp = `# gitee extension list — Agent Help

## Purpose
Enumerate every extension currently installed under
~/.gitee-cli/extensions/. Use this to audit which third-party
commands the user has wired into gitee-cli.

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
- Stdout contains a table of extensions (NAME, SOURCE, INSTALLED)
- or JSON array of manifest objects

## Output (stdout, table mode)
  NAME       SOURCE                INSTALLED
  myext      andershsueh/myext     2026-06-06

## Output (stdout, json mode)
  [{"name":"myext","source":"andershsueh/myext","installed_at":"...","path":"..."}]

## Common errors
- (none — empty install dir returns "No extensions installed")

## See also
- ` + "`" + `gitee extension install` + "`" + ` — add one
- ` + "`" + `gitee extension remove` + "`" + `  — drop one
`
