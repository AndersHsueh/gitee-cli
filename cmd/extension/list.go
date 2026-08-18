package extension

import (
	"fmt"

	"gitee-cli/cmd"
	"gitee-cli/internal/extension"
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
			return output.PrintJSON(manifests)
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

## Race condition warning ⚠️
This command mutates nothing itself, but it reads shared mutable
state: the ~/.gitee-cli/extensions/ directory. A concurrent
` + "`" + `gitee extension install` + "`" + ` / ` + "`" + `remove` + "`" + ` / ` + "`" + `upgrade` + "`" + ` can add, delete, or
rewrite a manifest mid-scan, so the printed table is a point-in-time
view that may already be stale when you act on it. If a follow-up
action depends on an entry (e.g. "remove X only if still
installed"), re-run ` + "`" + `gitee extension list` + "`" + ` immediately before acting,
and treat a subsequent "not installed" error from the follow-up
command as the expected outcome of the race, not a bug.

## OAuth scope required
N/A (local command) — no Gitee API request is made; works fully offline.

## Rate limit
N/A (local command) — no Gitee API request is made.

## Example
  gitee extension list
  # NAME       SOURCE                INSTALLED
  # myext      andershsueh/myext     2026-06-06

  gitee extension list -o json
  # [{"name":"myext","source":"andershsueh/myext","installed_at":"..."}]

## See also
- ` + "`" + `gitee extension install` + "`" + ` — add one
- ` + "`" + `gitee extension remove` + "`" + `  — drop one
`
