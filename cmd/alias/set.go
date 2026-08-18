package alias

import (
	"fmt"
	"strings"

	"gitee-cli/cmd"
	"gitee-cli/internal/alias"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set <name> <expansion>",
	Short: "Define a new alias, or overwrite an existing one",
	Args: cmd.AgentHelpAwareArgs(func(c *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("accepts 2 arg(s), received %d\n\nUsage: gitee alias set <name> <expansion...>\n\nExample: gitee alias set co 'pr checkout'", len(args))
		}
		return nil
	}),
	Example: `  gitee alias set co 'pr checkout'
  gitee alias set issues 'issue list --limit 50'
  gitee alias set mine 'issue list --assignee @me'`,
	PreRunE: preRunAgentHelp(aliasSetAgentHelp),
	RunE: func(c *cobra.Command, args []string) error {
		name := args[0]
		// Join remaining args with single spaces, then collapse runs
		// of whitespace. This is more forgiving than requiring the
		// user to quote the whole expansion: both
		//   gitee alias set co pr checkout
		// and
		//   gitee alias set co "pr checkout"
		// produce expansion="pr checkout".
		expansion := strings.Join(strings.Fields(strings.Join(args[1:], " ")), " ")

		prev, hadPrev, err := alias.Set(name, expansion)
		if err != nil {
			return fmt.Errorf("setting alias: %w", err)
		}
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		if hadPrev {
			fmt.Printf("%s alias %q updated: %s (was %q)\n", yellow("~"), name, expansion, prev)
		} else {
			fmt.Printf("%s alias %q created: %s\n", green("✓"), name, expansion)
		}
		return nil
	},
}

const aliasSetAgentHelp = `# gitee alias set — Agent Help

## Purpose
Define a new command shortcut, or overwrite an existing one. After
this command, typing ` + "`gitee <name> ...`" + ` at the shell is rewritten to
the stored expansion before cobra parses argv. Built-in commands
always take precedence over aliases of the same name.

## Pre-conditions
- None (no auth, no network)

## Required arguments
  <name>        Alias name. Same rules as extension names: [a-z0-9_-],
                1-40 chars, must not start with '.' or '-'.
  <expansion>   The rest of the args, joined with single spaces.
                Example: ` + "`gitee alias set co pr checkout`" + ` produces
                expansion="pr checkout". Quoting is optional; runs
                of whitespace are collapsed.

## Optional flags
  (inherits global flags; none specific to this verb)

## Idempotency
YES. Re-running with the same args is a no-op. Re-running with a
different expansion overwrites (returns "updated" rather than error).

## Atomicity
Atomic. The aliases file is replaced via temp-file + rename; partial
writes are never visible to other processes.

## Side effects
- ~/.gitee-cli/aliases.json is updated (chmod 0600)
- No remote interaction, no Gitee API calls

## Post-conditions (on success)
- ` + "`gitee alias list`" + ` shows the new entry
- ` + "`gitee <name> ...`" + ` invokes the expansion

## Common errors
- "alias name is empty"          -> first arg missing
- "invalid character"            -> name contains uppercase / dot / space
- "alias expansion is empty"     -> only the name was given
- "accepts 2 arg(s), received 1" -> missing expansion (run --help)

## Race condition warning ⚠️
The shared mutable state is ~/.gitee-cli/aliases.json, which this
command loads, modifies, and saves. Two concurrent ` + "`gitee alias set`" + `
invocations (even for DIFFERENT names) can lose one update: each loads
the file before the other saves, and the last save wins. The file
itself is never corrupt — writes are atomic (temp-file + rename) — but
a lost update means an alias you just set may silently revert to its
previous value. If multiple agents manage aliases, serialize set/delete
calls. Safe to retry: re-running with the same args converges to the
same state (see Idempotency).

## OAuth scope required
N/A (local command) — no Gitee API request is made; works fully offline.

## Rate limit
N/A (local command) — no Gitee API request is made.

## Example
  gitee alias set co 'pr checkout'
  gitee alias set issues 'issue list --limit 50'
  # ` + "`gitee co 123`" + ` now runs ` + "`gitee pr checkout 123`" + `

## See also
- ` + "`gitee alias list`" + `   — confirm it took effect
- ` + "`gitee alias delete`" + ` — remove an alias
`
