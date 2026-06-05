// Package extension implements the `gitee extension` command group:
// list / install / remove / upgrade / create. It also exposes a
// dynamic loader (LoadCommands) that synthesizes cobra commands from
// whatever is in the user's extension directory at startup.
//
// The extension model mirrors gh's:
//   - An extension is a git repo named `gitee-<name>`, with an
//     executable of the same `<name>` at its root.
//   - Installed extensions live at ~/.gitee-cli/extensions/<name>/.
//   - `gitee <name> ...` is forwarded to <ext>/<name>.
//   - `gitee extension install owner/repo` clones the repo into
//     the extension directory.
//   - `gitee extension create <name>` scaffolds a new extension.
package extension

import "github.com/spf13/cobra"

// ExtensionCmd is the top-level command for the gitee extension
// subcommand group.
var ExtensionCmd = &cobra.Command{
	Use:   "extension <subcommand>",
	Short: "Manage gitee-cli extensions",
	Long: `Install, manage, and create external commands that extend gitee-cli.

An extension is a git repository (named 'gitee-<name>') with an
executable of the same <name> at its root. Once installed, you can
invoke it directly as 'gitee <name> ...' just like a built-in command.

Use 'gitee extension list' to see what's installed, 'gitee extension
install <owner/repo>' to grab one from Gitee, and 'gitee extension
create <name>' to scaffold a new one.`,
}

func init() {
	ExtensionCmd.AddCommand(listCmd)
	ExtensionCmd.AddCommand(installCmd)
	ExtensionCmd.AddCommand(removeCmd)
	ExtensionCmd.AddCommand(upgradeCmd)
	ExtensionCmd.AddCommand(createCmd)
}
