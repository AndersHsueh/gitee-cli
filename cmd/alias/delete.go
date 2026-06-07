package alias

import (
	"fmt"

	"gitee-cli/internal/alias"
	"gitee-cli/internal/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var forceDelete bool

var deleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Aliases: []string{"rm"},
	Short:   "Remove an alias",
	Args:    cobra.ExactArgs(1),
	Example: `  gitee alias delete co
  gitee alias rm co --force
  gitee alias delete co               # prompts y/N`,
	PreRunE: preRunAgentHelp(aliasDeleteAgentHelp),
	RunE: func(c *cobra.Command, args []string) error {
		name := args[0]
		// Pre-check existence so we can give a clean "not defined" error
		// before the user answers the y/N prompt.
		expansion, ok := alias.Get(name)
		if !ok {
			return fmt.Errorf("alias %q is not defined", name)
		}
		if !forceDelete {
			yes, err := utils.ConfirmYN(
				fmt.Sprintf("%s Delete alias %q (currently %q)?",
					color.YellowString("?"), name, expansion),
				false,
			)
			if err != nil {
				return err
			}
			if !yes {
				fmt.Println("Cancelled")
				return nil
			}
		}
		if err := alias.Delete(name); err != nil {
			return fmt.Errorf("deleting alias: %w", err)
		}
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s removed alias %q\n", green("✓"), name)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Skip the y/N confirmation prompt")
}

const aliasDeleteAgentHelp = `# gitee alias delete — Agent Help

## Purpose
Remove an alias from ~/.gitee-cli/aliases.json. After this, the
shortcut no longer expands (a bare ` + "`gitee <name>`" + ` will fall through
to cobra and produce "unknown command" unless something else — an
extension, or a different alias — owns the name).

## Pre-conditions
- The alias is currently defined (otherwise errors with "not defined")

## Required arguments
  <name>   The alias name to remove

## Optional flags
  --force / -f   Skip the y/N confirmation prompt

## Idempotency
NOT idempotent. Re-running on a removed alias returns an error.

## Atomicity
Atomic. The aliases file is replaced atomically.

## Side effects
- ~/.gitee-cli/aliases.json is updated
- No remote interaction

## Post-conditions (on success)
- ` + "`gitee alias list`" + ` no longer shows the entry
- ` + "`gitee <name> ...`" + ` is no longer rewritten

## Common errors
- "is not defined"            -> name was never set, or already deleted
- "alias name ... contains"   -> name failed the [a-z0-9_-] validator

## Example
  gitee alias delete co
  gitee co 123
  # Error: unknown command "co" for "gitee"

## See also
- ` + "`gitee alias list`" + `   — confirm it's gone
- ` + "`gitee alias set`" + `    — re-add it
`
