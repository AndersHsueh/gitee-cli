package extension

import (
	"fmt"
	"os"

	"gitee-cli/cmd"
	"gitee-cli/internal/extension"
	"gitee-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var removeFlags struct {
	force bool
}

var removeCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove an installed extension",
	Args:    cobra.ExactArgs(1),
	Example: `  gitee extension remove myext
  gitee extension rm myext --force
  gitee extension remove myext                       # prompts y/N`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(extensionRemoveAgentHelp)
			os.Exit(0)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		raw := args[0]
		name, err := extension.ResolveName(raw)
		if err != nil {
			return err
		}
		// Confirm unless --force. Use the canonical name in the prompt.
		if !removeFlags.force {
			yes, err := utils.ConfirmYN(
				fmt.Sprintf("%s Remove extension %q?",
					color.YellowString("?"), name),
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
		if err := extension.Remove(name); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("extension %q is not installed", name)
			}
			return fmt.Errorf("removing %q: %w", name, err)
		}
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Printf("%s removed %s\n", green("✓"), name)
		return nil
	},
}

func init() {
	f := removeCmd.Flags()
	f.BoolVarP(&removeFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

const extensionRemoveAgentHelp = `# gitee extension remove — Agent Help

## Purpose
Uninstall an extension by removing its directory under
~/.gitee-cli/extensions/. The manifest is removed with it (the
manifest lives inside the dir). After this, ` + "`" + `gitee <name>` + "`" + ` is no
longer available.

## Pre-conditions
- The extension is currently installed (otherwise errors)

## Required arguments
  <name>   The extension name (with or without "gitee-" prefix; "gh-" also accepted)

## Optional flags
  --force / -f   Skip the y/N confirmation prompt

## Idempotency
NOT idempotent. Re-running on a removed extension returns an error.

## Atomicity
Atomic. Either the dir is gone, or no change.

## Side effects
- ~/.gitee-cli/extensions/<name>/ is deleted (recursively)
- No remote interaction; the source repo on Gitee is untouched

## Post-conditions (on success)
- ` + "`" + `gitee <name>` + "`" + ` is "unknown command" again
- ` + "`" + `gitee extension list` + "`" + ` no longer shows it

## Common errors
- "not installed"        -> already removed or never installed
- "invalid name"         -> name didn't pass the [a-z0-9_-] validator

## Example
  gitee extension remove myext
  gitee myext foo
  # Error: unknown command "myext" for "gitee"

## See also
- ` + "`" + `gitee extension list` + "`" + `   — confirm it's gone
- ` + "`" + `gitee extension install` + "`" + ` — re-add it
`
