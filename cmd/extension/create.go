package extension

import (
	"fmt"
	"os"
	"path/filepath"

	"gitee-cli/cmd"
	"gitee-cli/internal/extension"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Scaffold a new extension in the current directory",
	Args:  cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	Example: `  gitee extension create myext
  # creates ./gitee-myext/ with myext/ executable + README

  cd gitee-myext
  ./myext foo                            # try it locally
  gitee extension install .              # install it for the current user`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(extensionCreateAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		name := args[0]
		// Sanity: name follows the same rules as install.
		// Use the same ResolveName path so a "gitee-myext" arg
		// normalises to "myext".
		canonical, err := extension.ResolveName(name)
		if err != nil {
			return err
		}
		dirName := "gitee-" + canonical
		dir := dirName
		if _, err := os.Stat(dir); err == nil {
			return fmt.Errorf("directory %q already exists in CWD", dir)
		}
		// Create the parent (gitee-<name>/) only. The <name> subdir
		// is created by WriteFile below — MkdirAll-ing a nested
		// dir would make the executable path a directory.
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}

		// Scaffold a minimal shell-script extension. (Go-template
		// option is tracked as v0.9.0-c work; the shell version
		// covers the common case and zero deps.)
		exe := filepath.Join(dir, canonical)
		if err := os.WriteFile(exe, []byte(scaffoldScript(canonical)), 0755); err != nil {
			return fmt.Errorf("write %s: %w", exe, err)
		}

		// README with install + invoke instructions.
		readme := filepath.Join(dir, "README.md")
		if err := os.WriteFile(readme, []byte(scaffoldREADME(canonical)), 0644); err != nil {
			return fmt.Errorf("write %s: %w", readme, err)
		}

		green := color.New(color.FgGreen).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()
		fmt.Printf("%s scaffolded %s\n", green("✓"), bold(dirName))
		fmt.Println("  next steps:")
		fmt.Printf("    cd %s && edit %s\n", bold(dirName), bold(canonical))
		fmt.Printf("    %s\n", bold("gitee extension install ."))
		fmt.Printf("    %s %s <args>\n", bold("gitee"), canonical)
		return nil
	},
}

// scaffoldScript returns the body of the entry-point shell script.
// It echoes the args and demonstrates a subcommand pattern.
func scaffoldScript(name string) string {
	return fmt.Sprintf(`#!/bin/sh
# gitee extension: %s
# This file is the entry point. Anything you `+"`"+`echo`+"`"+` goes to the user's
# terminal. Args are forwarded from `+"`"+`gitee %s ...`+"`"+`.

set -e

case "${1:-}" in
  --agent-help)
    cat <<AGENT_HELP
# gitee %s — Agent Help

## Purpose
Demonstration extension scaffolded by `+"`"+`gitee extension create`+"`"+`.

## Pre-conditions
- None

## Optional flags
  (none — accepts any args)

## Idempotency
YES (read-only).

## Atomicity
Atomic.

## Side effects
None.

## Post-conditions
- Stdout prints the args list

## Example
  gitee %s foo bar
  # got: foo bar

## See also
- `+"`"+`gitee extension list`+"`"+`   — see installed extensions
- `+"`"+`gitee extension create`+"`"+`  — scaffold a new one
AGENT_HELP
    ;;
  "")
    echo "usage: gitee %s <args...>"
    echo "try: gitee %s --agent-help"
    ;;
  *)
    echo "got: $*"
    ;;
esac
`, name, name, name, name, name, name)
}

// scaffoldREADME returns the README contents for the new extension.
func scaffoldREADME(name string) string {
	return fmt.Sprintf(`# gitee-%s

A gitee-cli extension. Invoked as:

    gitee %s <args>

## Local development

    cd gitee-%s
    ./%s foo
    # got: foo

## Install (current user)

    gitee extension install .

After install, the extension is available globally as `+"`"+`gitee %s`+"`"+`.

## Upgrade (after pushing changes to a Gitee repo)

    gitee extension upgrade %s

## Uninstall

    gitee extension remove %s

## Publish (optional)

    # Create a Gitee repo named "gitee-%s" and push:
    cd gitee-%s
    git init && git add . && git commit -m "initial"
    git remote add origin https://gitee.com/<your-username>/gitee-%s.git
    git push -u origin master
`, name, name, name, name, name, name, name, name, name, name)
}

const extensionCreateAgentHelp = `# gitee extension create — Agent Help

## Purpose
Scaffold a brand-new extension in the current working directory.
Creates ` + "`" + `gitee-<name>/<name>` + "`" + ` (an executable shell script) and a
README with install/upgrade/uninstall instructions. The user can
edit the script to do anything and then ` + "`" + `gitee extension install .` + "`" + ` to
install it.

This is the shell-script scaffold. A ` + "`" + `--go` + "`" + ` flag for a Go
project (go.mod + main.go template) is on the roadmap.

## Pre-conditions
- CWD is writable
- CWD does not already contain a ` + "`" + `gitee-<name>/` + "`" + ` directory

## Required arguments
  <name>   Extension name (with or without "gitee-" prefix; the prefix is auto-added to the directory)

## Optional flags
  (none)

## Idempotency
NOT idempotent. Re-running errors with "directory already exists".

## Atomicity
Atomic. Files are written in order; on failure, the partial dir is
left for the user to inspect.

## Side effects
- Creates ` + "`" + `./gitee-<name>/<name>` + "`" + ` (chmod 0755) and ` + "`" + `./gitee-<name>/README.md` + "`" + `

## Post-conditions (on success)
- A ready-to-edit extension at ` + "`" + `gitee-<name>/<name>` + "`" + `
- README with install / upgrade / uninstall instructions
- Run ` + "`" + `gitee extension install .` + "`" + ` to install

## Common errors
- "directory already exists"   -> choose a different name or rm the old dir
- "name contains invalid char" -> use [a-z0-9_-]

## Race condition warning ⚠️
The shared mutable state is the CWD filesystem — the ` + "`" + `gitee-<name>/` + "`" + `
directory this command creates. The "already exists" stat check and
the mkdir are NOT atomic: two concurrent ` + "`" + `gitee extension create <name>` + "`" + `
runs can both pass the check, after which one run's writes silently
overwrite the other's (same executable path, last-writer-wins). In
multi-agent setups, serialize creates that share a name or give each
agent a unique name. Retry is safe: rm the partial ` + "`" + `gitee-<name>/` + "`" + ` dir
and re-run — a second run on an intact existing dir errors out
instead of merging (see Idempotency above).

## OAuth scope required
N/A (local command) — no Gitee API request is made; works fully offline.

## Rate limit
N/A (local command) — no Gitee API request is made.

## Example
  cd ~/projects
  gitee extension create myext
  # created ./gitee-myext/ with myext/ + README.md
  cd gitee-myext
  vim myext                          # edit the script
  gitee extension install .
  gitee myext hello                  # -> got: hello

## See also
- ` + "`" + `gitee extension list` + "`" + `   — see installed extensions
- ` + "`" + `gitee extension install` + "`" + ` — install the scaffolded extension
`
