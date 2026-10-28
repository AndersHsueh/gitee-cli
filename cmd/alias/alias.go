// Package alias implements the `gitee alias` command group: set / list
// / delete. The runtime resolver that splices user-defined aliases
// into os.Args lives in internal/alias.
package alias

import (
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
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
}

func init() {
	AliasCmd.AddCommand(setCmd)
	AliasCmd.AddCommand(listCmd)
	AliasCmd.AddCommand(deleteCmd)
}

// preRunAgentHelp returns a PreRunE that delegates to cmd.PrintAgentHelp
// when --agent-help is set, otherwise returns nil. Each verb in this
// package supplies its own agent-help const. Using cmd.PrintAgentHelp
// (instead of raw fmt.Print+os.Exit) ensures --agent-help --output json
// produces structured JSON like every other command.
func preRunAgentHelp(helpText string) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, _ []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(helpText)
		}
		return nil
	}
}
