// Package alias implements the `gitee alias` command group: set / list
// / delete. The runtime resolver that splices user-defined aliases
// into os.Args lives in internal/alias.
package alias

import (
	"fmt"
	"os"

	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

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

// preRunAgentHelp returns a PreRunE that prints helpText and exits
// 0 when --agent-help is set, otherwise returns nil. Each verb in
// this package supplies its own agent-help const.
func preRunAgentHelp(helpText string) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, _ []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(helpText)
			os.Exit(0)
		}
		return nil
	}
}
