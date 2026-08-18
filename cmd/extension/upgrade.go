package extension

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gitee-cli/cmd"
	"gitee-cli/internal/extension"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var upgradeFlags struct {
	all    bool
	force  bool
	dryRun bool
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [<name> | --all]",
	Short: "Upgrade installed extensions by pulling the latest",
	Args:  cobra.MaximumNArgs(1),
	Example: `  gitee extension upgrade myext
  gitee extension upgrade --all
  gitee extension upgrade --all --dry-run`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(extensionUpgradeAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		// Decide which extensions to upgrade.
		var targets []string
		if upgradeFlags.all {
			manifests, err := extension.ListInstalled()
			if err != nil {
				return err
			}
			if len(manifests) == 0 {
				fmt.Println("No extensions installed.")
				return nil
			}
			for _, m := range manifests {
				targets = append(targets, m.Name)
			}
		} else {
			if len(args) == 0 {
				return fmt.Errorf("specify an extension name or --all")
			}
			name, err := extension.ResolveName(args[0])
			if err != nil {
				return err
			}
			targets = []string{name}
		}

		// Upgrade each. Continue on error (--all shouldn't stop at
		// the first failure); the user can re-run with the specific
		// name to see what went wrong.
		anyErr := false
		for _, name := range targets {
			if err := upgradeOne(name); err != nil {
				fmt.Fprintf(os.Stderr, "%s upgrade %s: %v\n",
					color.RedString("Error:"), name, err)
				anyErr = true
			}
		}
		if anyErr {
			return fmt.Errorf("one or more upgrades failed")
		}
		return nil
	},
}

func init() {
	f := upgradeCmd.Flags()
	f.BoolVar(&upgradeFlags.all, "all", false, "Upgrade all installed extensions")
	f.BoolVar(&upgradeFlags.force, "force", false, "Force upgrade (allow dirty / diverged working tree)")
	f.BoolVar(&upgradeFlags.dryRun, "dry-run", false, "Only display what would be upgraded")
}

// upgradeOne pulls the latest for a single extension and updates
// its manifest's PinnedRef to the new HEAD.
//
// For local-path installs (symlinked), there's nothing to "pull" —
// the user just edits their source dir. We skip these with a notice.
func upgradeOne(name string) error {
	extPath := extension.ExtPath(name)
	m, err := extension.ReadManifest(name)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	// Local symlink: no network op available.
	if m.Source != "" && (m.Source[0] == '/' || m.Source[0] == '.' || m.Source == "local") {
		fmt.Printf("%s %s is a local install (source=%s); nothing to pull\n",
			color.YellowString("skip:"), name, m.Source)
		return nil
	}

	green := color.New(color.FgGreen).SprintFunc()
	if upgradeFlags.dryRun {
		fmt.Printf("would pull %s in %s\n", name, extPath)
		return nil
	}

	cmd := exec.Command("git", "-C", extPath, "pull")
	if upgradeFlags.force {
		cmd.Args = append(cmd.Args, "--force")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull in %s: %w", extPath, err)
	}

	// Update PinnedRef to the new HEAD.
	if out, err := exec.Command("git", "-C", extPath, "rev-parse", "--short", "HEAD").Output(); err == nil {
		m.PinnedRef = trim(string(out))
		if err := extension.WriteManifest(m); err != nil {
			// Non-fatal: the upgrade succeeded, we just couldn't
			// update the manifest. Print a warning but don't fail.
			fmt.Fprintf(os.Stderr, "warning: updated but couldn't rewrite manifest: %v\n", err)
		}
	}
	fmt.Printf("%s upgraded %s -> %s\n", green("✓"), name, filepath.Base(extPath))
	return nil
}

// trim is a tiny helper to keep the upgrade flow tight; the result
// of exec.Command.Output has a trailing newline.
func trim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

const extensionUpgradeAgentHelp = `# gitee extension upgrade — Agent Help

## Purpose
Pull the latest code for one (or all) installed extensions. After
this, ` + "`" + `gitee <name>` + "`" + ` runs the freshly-pulled executable.

Local-symlink installs are skipped (no network op available — the
user edits the source dir in place).

## Pre-conditions
- git on PATH
- For remote installs: network access to the source repo

## Optional flags
  --all         Upgrade every installed extension
  --force       Pass --force to ` + "`" + `git pull` + "`" + ` (allow non-ff)
  --dry-run     Only print what would be pulled; don't actually pull

## Idempotency
YES (mostly). Pulling when already up-to-date is a no-op. Pulling
when the working dir is dirty produces a merge conflict; the
manifest is NOT updated in that case (it stays pinned to the
pre-pull HEAD).

## Atomicity
Non-atomic at the git level. A failed pull leaves the working
dir as-is; the previous manifest + executable are still in place.

## Side effects
- Network: one ` + "`" + `git pull` + "`" + ` per remote extension
- ~/.gitee-cli/extensions/<name>/manifest.json (PinnedRef updated)
- The executable's bytes are replaced with the pulled version

## Post-conditions (on success)
- Source repo HEAD is reflected in the local clone
- Manifest's PinnedRef matches the new HEAD
- ` + "`" + `gitee <name>` + "`" + ` next call uses the new code

## Common errors
- "git pull failed"  -> network, conflict, or diverged branch
- "skip: local"      -> extension was installed from a local path
- "read manifest"    -> manifest.json missing or corrupt

## Race condition warning ⚠️
Two layers of shared mutable state. (1) The upstream Gitee repo:
` + "`" + `git pull` + "`" + ` fetches whatever HEAD is at that moment, so a push that
lands between your --dry-run and the real run changes what "latest"
means. (2) The local clone + manifest.json under
~/.gitee-cli/extensions/<name>/: two concurrent upgrades of the same
extension race inside one git working dir — the second typically
fails on git's index.lock. That failure is safe to retry (consistent
with Idempotency: re-pulling once up-to-date is a no-op). With
--all, the target list is snapshotted at start; an extension
installed or removed concurrently is simply absent from (or drops
out of) this run's set. A pull that hits a merge conflict leaves the
working tree dirty and the manifest pinned to the pre-pull HEAD —
resolve or re-run with --force before assuming the upgrade happened.

## OAuth scope required
None (no Gitee API call; uses ` + "`" + `git pull` + "`" + ` over HTTPS). Local-symlink
installs are skipped entirely with no network op.

## Rate limit
git pull is not rate-limited by Gitee the way the API is.

## Example
  gitee extension upgrade --all --dry-run
  # would pull myext in /Users/.../extensions/myext
  # would pull pr-stats in /Users/.../extensions/pr-stats

## See also
- ` + "`" + `gitee extension list` + "`" + `    — see which are installed
- ` + "`" + `gitee extension install` + "`" + ` — fresh install (also re-pulls)
`
