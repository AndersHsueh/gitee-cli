package label

import (
	"gitee-cli/cmd"

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
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
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

// AgentHelpMarkdown returns the --agent-help markdown for the label verb
// named verb (list/view/create/edit/delete/clone), or "" when unknown.
//
// The legacy `gitee issue label <verb>` aliases (cmd/issue/label.go) reuse
// the same doc text as the top-level `gitee label <verb>` commands but must
// present their own command path in the heading; they call this to get the
// body and then retitle it. Keeping the mapping here (next to the consts
// it indexes) means adding a new label verb without updating it is a
// test-visible omission rather than a silently mis-titled doc.
func AgentHelpMarkdown(verb string) string {
	switch verb {
	case "list":
		return labelListAgentHelp
	case "view":
		return labelViewAgentHelp
	case "create":
		return labelCreateAgentHelp
	case "edit":
		return labelEditAgentHelp
	case "delete":
		return labelDeleteAgentHelp
	case "clone":
		return labelCloneAgentHelp
	}
	return ""
}

func init() {
	LabelCmd.AddCommand(listCmd)
	LabelCmd.AddCommand(viewCmd)
	LabelCmd.AddCommand(createCmd)
	LabelCmd.AddCommand(editCmd)
	LabelCmd.AddCommand(deleteCmd)
	LabelCmd.AddCommand(cloneCmd)
}
