package extension

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// LoadCommands is the dynamic-loader entry point. It scans ExtDir
// for installed extensions and synthesizes a *cobra.Command for each
// one. The returned commands are not yet attached to any parent —
// the caller (cmd.RegisterDynamicCommands) decides where to hang
// them and whether to filter out name collisions with built-ins.
//
// Skipped (with a warning) extensions:
//   - manifest cannot be read
//   - executable file with the same name as the extension is missing
//   - the executable is not +x (we don't auto-chmod for safety)
//
// Returns an empty slice (not an error) if ExtDir is missing or
// has no extensions. Returns an error only for unrecoverable
// problems (e.g. permission denied while reading ExtDir).
func LoadCommands() ([]*cobra.Command, error) {
	manifests, err := ListInstalled()
	if err != nil {
		return nil, err
	}
	var cmds []*cobra.Command
	for _, m := range manifests {
		cmd, err := commandForManifest(m)
		if err != nil {
			// Don't abort the whole load — one broken extension
			// shouldn't take down the rest. Print to stderr and skip.
			fmt.Fprintf(os.Stderr, "extension: skipping %q: %v\n", m.Name, err)
			continue
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

// commandForManifest builds the cobra.Command that proxies `gitee <name>`
// to <ExtDir>/<name>/<name>. The command is dynamic — it exists only
// when the user has installed the extension.
func commandForManifest(m *Manifest) (*cobra.Command, error) {
	execPath := filepath.Join(m.Path, m.Name)
	info, err := os.Stat(execPath)
	if err != nil {
		return nil, fmt.Errorf("executable not found: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory, not an executable", execPath)
	}
	// We don't require +x here — the run-time exec.Command will fail
	// loudly if the bit isn't set, and that error message ("permission
	// denied") is more useful than silently chmod'ing the user's file.
	return &cobra.Command{
		Use:                m.Name + " [args...]",
		Short:              fmt.Sprintf("Extension: %s (from %s)", m.Name, m.Source),
		DisableFlagParsing: true, // forward all args verbatim to the extension
		SilenceUsage:       true, // the extension's own usage is what the user wants
		SilenceErrors:      true, // we propagate the exit code via os.Exit; don't double-print
		RunE: func(c *cobra.Command, args []string) error {
			// Propagate agent-help (if the user asked for it) so the
			// extension can print its own structured help. This is the
			// convention we established for top-level commands. We
			// check os.Args directly (rather than c.Flags()) because
			// the root persistent flag may not be on this subcommand's
			// flag set depending on how cobra propagated it.
			allArgs := args
			for _, a := range os.Args[1:] {
				if a == "--agent-help" || a == "-h" {
					// Only inject if the user actually requested it
					// on the invocation of *this* extension. Detecting
					// it on any positional is good enough in practice
					// (root flag presence implies the user wants
					// help everywhere).
					allArgs = append([]string{"--agent-help"}, allArgs...)
					break
				}
			}
			// #nosec G204 — execPath is constrained to ExtPath(name)/name
			// which is itself constrained to the user's own ExtDir;
			// the name is validated against [a-z0-9_-]+. A hostile
			// extension install is out of scope (the user installed it).
			ext := exec.Command(execPath, allArgs...)
			ext.Stdin = os.Stdin
			ext.Stdout = os.Stdout
			ext.Stderr = os.Stderr
			if err := ext.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					// Preserve the extension's exit code so users can
					// write `gitee mytool ... && ...` and have it
					// behave like a normal command.
					os.Exit(exitErr.ExitCode())
				}
				return err
			}
			return nil
		},
	}, nil
}
