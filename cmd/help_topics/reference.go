package help_topics

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// referenceTopic is the implementation of `gitee help reference`.
//
// It walks the live cobra command tree under root and prints a flat
// reference: every verb with its Use path and Short description.
// This is auto-generated; the intent is that running the command
// always shows what's actually wired up, so it can never go stale
// relative to a separately-maintained doc page.
//
// Cobra exposes the command tree via rootCmd.Commands(). We do a
// depth-first walk so the output mirrors what `gitee help <command>`
// would show individually.
var referenceTopic = &cobra.Command{
	Use:   "reference",
	Short: "A flat list of every command gitee-cli ships (auto-generated)",
	Long: `Auto-generated command reference.

Walks the cobra command tree and prints every verb the gitee-cli
binary knows about, with its Use path and Short description. This
page is generated on every run, so it always matches what the
binary actually does — unlike a hand-maintained docs site, it
cannot go stale.

For full docs on a specific command, run 'gitee help <command>'
or 'gitee <command> --agent-help' (when available).`,
	RunE: func(c *cobra.Command, args []string) error {
		root := c.Root()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "COMMAND\tSHORT")
		// Track which full-paths we've printed so the help
		// subcommand (which we recurse into) doesn't show up
		// twice — once as "gitee help" and again when its
		// topics are listed as "gitee help <topic>".
		printed := map[string]bool{}
		walkCommands(w, root, "", printed)
		return w.Flush()
	},
}

// walkCommands recursively prints every command in the tree. We
// recurse into "help" so the help topics are included, but we
// don't print "help" itself (it's a dispatcher; its subcommands
// are the real commands the user would invoke). Version is
// skipped entirely because it's covered by 'gitee --version'.
//
// We print the parent BEFORE the children so the output reads
// top-down (matches `gitee --help` and `gitee help <cmd>`).
func walkCommands(w *tabwriter.Writer, cmd *cobra.Command, prefix string, printed map[string]bool) {
	for _, sub := range cmd.Commands() {
		if sub.Name() == "version" {
			continue
		}
		full := strings.TrimSpace(prefix + " " + sub.Name())
		// Print the parent first (unless it's the help
		// dispatcher, which we'll skip — but its children
		// are still walked below).
		if sub.Name() != "help" && !sub.Hidden && !printed[full] {
			short := sub.Short
			if short == "" {
				short = "(no short description)"
			}
			fmt.Fprintf(w, "gitee %s\t%s\n", full, short)
			printed[full] = true
		}
		// Then recurse into children with the path as prefix.
		if len(sub.Commands()) > 0 {
			walkCommands(w, sub, full, printed)
		}
	}
}
