// Package alias implements the `gitee alias` command group: set / list
// / delete. It also exposes a runtime resolver (cmd/alias.Resolve) that
// is wired into cmd.Execute() in main.go so that `gitee co 123` is
// transparently rewritten to `gitee pr checkout 123` when `co` is a
// user-defined alias.
//
// Aliases are stored as a flat JSON map at ~/.gitee-cli/aliases.json.
// See internal/alias for the storage layer.
package alias

import "github.com/spf13/cobra"

// AliasCmd is the top-level command for the gitee alias subcommand group.
var AliasCmd = &cobra.Command{
	Use:   "alias <subcommand>",
	Short: "Create command shortcuts for gitee-cli",
	Long: `Define personal shortcuts that expand to longer gitee-cli commands.

For example, after 'gitee alias set co "pr checkout"', typing
'gitee co 123' is rewritten to 'gitee pr checkout 123' before
the command parser runs. Built-in commands always win over aliases
of the same name.

Use 'gitee alias list' to see what's defined and 'gitee alias
delete <name>' to remove one.`,
}

func init() {
	AliasCmd.AddCommand(setCmd)
	AliasCmd.AddCommand(listCmd)
	AliasCmd.AddCommand(deleteCmd)
}
