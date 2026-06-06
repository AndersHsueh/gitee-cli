package help_topics

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RegisterHelpTopics attaches every topic in this package as a
// subcommand of a help command on root and returns that help
// command. The implementation replaces cobra's default help command
// with one that has the same "show help for a named command"
// behavior PLUS our four topics as explicit subcommands (so
// `gitee help environment` works and tab-completion shows the
// topics).
//
// Why we replace rather than graft: cobra v1.10 does not expose
// helpCommand (it's unexported), so we cannot attach topics to
// the auto-created help command. Replacing it wholesale with a
// custom command that mimics cobra's default dispatch is the
// simplest way to add topics.
//
// Safe to call before or after the user types a command — topics
// are only invoked explicitly via 'gitee help <topic>'.
func RegisterHelpTopics(root *cobra.Command) *cobra.Command {
	helpCmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Help provides help for any command in the application.

Simply type 'gitee help [path to command]' for full details, or
'gitee help <topic>' for cross-cutting documentation:

  gitee help environment    — env vars recognized by gitee-cli
  gitee help exit-codes     — exit codes & their meaning
  gitee help formatting     — output formats and jq recipes
  gitee help reference      — auto-generated command index`,
		Args: cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			// Replicate cobra's default help dispatch:
			//   1. No args                       -> show root help
			//   2. Args match a command/topic    -> show that command's help
			//   3. Args don't match anything     -> root help + hint
			if len(args) == 0 {
				return c.Root().Help()
			}
			cmd, _, err := c.Root().Find(args)
			if err == nil && cmd != nil && cmd != c.Root() {
				return cmd.Help()
			}
			root := c.Root()
			_, _ = fmt.Fprintf(root.OutOrStderr(),
				"unknown help topic %q; showing root help\n", args)
			return root.Help()
		},
	}
	for _, topic := range []*cobra.Command{
		environmentTopic,
		exitCodesTopic,
		formattingTopic,
		referenceTopic,
	} {
		helpCmd.AddCommand(topic)
	}
	root.SetHelpCommand(helpCmd)
	return helpCmd
}
