package label

import (
	"github.com/spf13/cobra"
)

// LabelCmd is the top-level command for managing Gitee labels.
//
// NOTE: This coexists with `gitee issue label` (see cmd/issue/label.go).
// Both commands share the same underlying handler (internal/handler) so
// behavior is identical; the top-level form is the canonical entrypoint
// going forward and is the one recommended for AI agents.
var LabelCmd = &cobra.Command{
	Use:   "label <subcommand>",
	Short: "Manage repository labels",
	Long: `Manage labels for a Gitee repository.

Labels are used to categorize and organize issues and pull requests.
You can create, list, view, edit, delete, and clone labels between
repositories.

This is the top-level form of the command; the legacy path
"gitee issue label" is preserved as an alias and behaves identically.`,
}

// Subcommands returns a copy of the label subcommand pointers.
// Exposed so the legacy `gitee issue label` path can register
// aliases that point to the same Cobra commands.
func Subcommands() []*cobra.Command {
	return []*cobra.Command{
		listCmd,
		viewCmd,
		createCmd,
		editCmd,
		deleteCmd,
		cloneCmd,
	}
}

func init() {
	LabelCmd.AddCommand(listCmd)
	LabelCmd.AddCommand(viewCmd)
	LabelCmd.AddCommand(createCmd)
	LabelCmd.AddCommand(editCmd)
	LabelCmd.AddCommand(deleteCmd)
	LabelCmd.AddCommand(cloneCmd)
}
