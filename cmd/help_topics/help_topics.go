package help_topics

import (
	"fmt"

	"github.com/spf13/cobra"
)

// currentHelpCmd is set by RegisterHelpTopics so tests can
// inspect the help command (cobra v1.10 does not export a
// HelpCommand() getter — helpCommand is unexported).
var currentHelpCmd *cobra.Command

// RegisterHelpTopics attaches every topic in this package as a
// subcommand of a help command on root. The implementation
// replaces cobra's default help command with one that has the
// same "show help for a named command" behavior PLUS our four
// topics as explicit subcommands (so `gitee help environment`
// works and tab-completion shows the topics).
//
// Idempotent: calling it twice will be a no-op the second time
// because cobra refuses to add a command with a duplicate Use.
//
// Safe to call before or after the user types a command — topics
// are only invoked explicitly via 'gitee help <topic>'.
func RegisterHelpTopics(root *cobra.Command) {
	// Build a fresh help command that mimics cobra's default but
	// also has our 4 topics. We can't easily graft onto the
	// auto-created one (helpCommand is unexported in cobra v1.10),
	// so we replace it.
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
			// No match — fall through to root help with a hint
			// that topics are available.
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
	currentHelpCmd = helpCmd
}

// HelpCommandForTest returns the help command created by the most
// recent RegisterHelpTopics call. Used by tests; cobra v1.10 has
// no public getter for the help command. Returns nil if
// RegisterHelpTopics has not been called yet.
func HelpCommandForTest() *cobra.Command { return currentHelpCmd }

// AllTopicNames returns the list of topic names in the order they
// are registered. Used by the content_test to assert wiring
// independent of cobra internals.
func AllTopicNames() []string {
	return []string{
		environmentTopic.Name(),
		exitCodesTopic.Name(),
		formattingTopic.Name(),
		referenceTopic.Name(),
	}
}
