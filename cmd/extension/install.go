package extension

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/extension"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var installFlags struct {
	force bool
	name  string // optional override; default derived from source
}

var installCmd = &cobra.Command{
	Use:   "install <owner/repo | /local/path>",
	Short: "Install an extension from Gitee or a local path",
	Args:  cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	Example: `  gitee extension install andershsueh/my-tool
  gitee extension install https://gitee.com/owner/repo.git
  gitee extension install ./my-local-tool
  gitee extension install .                              # install from current dir (scaffolds gitee-<name> from create)
  gitee extension install owner/repo --name custom-name  # override the name
  gitee extension install owner/repo --force             # overwrite existing`,
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(extensionInstallAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		src := strings.TrimSpace(args[0])
		destName, err := resolveInstallName(src, installFlags.name)
		if err != nil {
			return err
		}
		dest := extension.ExtPath(destName)

		// Refuse to clobber unless --force.
		if _, err := os.Stat(dest); err == nil {
			if !installFlags.force {
				return fmt.Errorf("%q is already installed; use --force to overwrite", destName)
			}
			if err := os.RemoveAll(dest); err != nil {
				return fmt.Errorf("removing existing %q: %w", destName, err)
			}
		}

		now := time.Now().UTC()
		var pinned string
		switch {
		case isLocalPath(src):
			if err := installLocal(src, dest); err != nil {
				return err
			}
			pinned = "local" // no git ref for symlink/copy
		case looksLikeURL(src):
			if err := installClone(src, dest); err != nil {
				return err
			}
			pinned, _ = gitHead(dest)
		default: // owner/repo
			cloneURL := buildGiteeCloneURL(src)
			if err := installClone(cloneURL, dest); err != nil {
				return err
			}
			pinned, _ = gitHead(dest)
		}

		// Write manifest. Use the original src as Source so the user
		// can re-install with the same command.
		if err := extension.WriteManifest(&extension.Manifest{
			Name:        destName,
			Path:        dest,
			Source:      src,
			InstalledAt: now,
			PinnedRef:   pinned,
		}); err != nil {
			return fmt.Errorf("writing manifest: %w", err)
		}

		// Ensure the <name> executable bit is set (clone may have lost it).
		exe := filepath.Join(dest, destName)
		if info, err := os.Stat(exe); err == nil && !info.IsDir() {
			_ = os.Chmod(exe, 0755)
		}

		green := colorGreen()
		bold := colorBold()
		fmt.Printf("%s installed %s from %s\n", green("✓"), bold(destName), src)
		fmt.Printf("  invoke as: %s %s <args>\n", bold("gitee"), destName)
		return nil
	},
}

func init() {
	f := installCmd.Flags()
	f.BoolVarP(&installFlags.force, "force", "f", false, "Overwrite an existing install")
	f.StringVar(&installFlags.name, "name", "", "Override the extension name (default: derived from source)")
}

// resolveInstallName returns the final name to install under. Priority:
//  1. explicit --name (if set)
//  2. for "owner/repo"      -> strip "gitee-" / "gh-" prefix
//  3. for "gitee-foo.git"   -> "foo"
//  4. for "/local/path"     -> basename of path, with "gitee-" / "gh-" stripped
//  5. for "https://.../foo" -> "foo" (basename without .git)
func resolveInstallName(src, explicit string) (string, error) {
	if explicit != "" {
		return extension.ResolveName(explicit)
	}
	raw := baseNameOfSource(src)
	return extension.ResolveName(raw)
}

// installClone runs `git clone` into dest.
func installClone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s -> %s failed: %w", url, dest, err)
	}
	return nil
}

// installLocal symlinks the gitee-<name> subdir of src into dest.
//
// The convention is: src contains a single `gitee-<name>/` directory
// (created by `gitee extension create`). The extension executable
// lives at src/gitee-<name>/<name>. The install layout requires
// the executable at <dest>/<name>, so we symlink src/gitee-<name>
// to <dest>. This way, edits in the source dir are immediately
// visible as edits in the installed extension.
//
// We use absolute paths so the symlink survives cwd changes.
func installLocal(src, dest string) error {
	abs, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("resolving %s: %w", src, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("source %s: %w", abs, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source %s is not a directory", abs)
	}

	// If src itself is a "gitee-<x>" directory, symlink it directly.
	// Otherwise, find the single gitee-<x> child and symlink that.
	target := abs
	base := filepath.Base(abs)
	if !strings.HasPrefix(base, "gitee-") {
		entries, err := os.ReadDir(abs)
		if err != nil {
			return fmt.Errorf("reading %s: %w", abs, err)
		}
		var found string
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), "gitee-") {
				found = filepath.Join(abs, e.Name())
				break
			}
		}
		if found == "" {
			return fmt.Errorf("source %s contains no gitee-<name>/ subdirectory; run 'gitee extension create <name>' first, or pass the path to the gitee-<name> dir directly", abs)
		}
		target = found
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return err
	}
	if err := os.Symlink(target, dest); err != nil {
		return fmt.Errorf("symlink %s -> %s failed (try copying manually): %w", dest, target, err)
	}
	return nil
}

// gitHead returns the short commit SHA at HEAD in dir, or "" on
// any error. Used to populate Manifest.PinnedRef.
func gitHead(dir string) (string, error) {
	c := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD")
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// isLocalPath returns true if src is a filesystem path (absolute or
// relative starting with . or /).
func isLocalPath(src string) bool {
	return strings.HasPrefix(src, "/") || strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../") || src == "."
}

// looksLikeURL is a cheap heuristic for http(s) URLs.
func looksLikeURL(src string) bool {
	return strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")
}

// buildGiteeCloneURL converts "owner/repo" into the canonical
// https clone URL. Adds .git if missing.
func buildGiteeCloneURL(ownerRepo string) string {
	if !strings.HasSuffix(ownerRepo, ".git") {
		ownerRepo += ".git"
	}
	return "https://gitee.com/" + ownerRepo
}

// baseNameOfSource extracts the extension name from any of the
// supported source forms.
func baseNameOfSource(src string) string {
	switch {
	case isLocalPath(src):
		// Resolve to absolute so we can inspect the directory.
		abs := src
		if a, err := filepath.Abs(src); err == nil {
			abs = a
		}
		// If the path itself is a "gitee-<x>" dir, use "x".
		// (User ran `gitee extension install ./gitee-foo`.)
		base := filepath.Base(abs)
		if strings.HasPrefix(base, "gitee-") {
			return base[len("gitee-"):]
		}
		// Otherwise, the user pointed at a parent dir. Look INSIDE
		// for a single "gitee-<x>" subdir. This is the common
		// post-`create` flow: `gitee extension create foo` makes
		// ./gitee-foo, then `gitee extension install .` should name
		// the install "foo" (not "." or "fakeext").
		if entries, err := os.ReadDir(abs); err == nil {
			var found string
			count := 0
			for _, e := range entries {
				if e.IsDir() && strings.HasPrefix(e.Name(), "gitee-") {
					found = e.Name()[len("gitee-"):]
					count++
				}
			}
			if count == 1 {
				return found
			}
		}
		// Fallback: use the dir's own name. May include the
		// "gitee-" prefix; ResolveName strips it.
		return base
	case looksLikeURL(src):
		// e.g. https://gitee.com/owner/gitee-foo.git -> gitee-foo
		base := filepath.Base(src)
		base = strings.TrimSuffix(base, ".git")
		return base
	default:
		// owner/repo -> repo
		if i := strings.LastIndex(src, "/"); i >= 0 {
			return src[i+1:]
		}
		return src
	}
}

// colour helpers. Kept local to this file so we don't pull
// fatih/color into every verb.
func colorGreen() func(...any) string { return color.New(color.FgGreen).SprintFunc() }
func colorBold() func(...any) string  { return color.New(color.Bold).SprintFunc() }

const extensionInstallAgentHelp = `# gitee extension install — Agent Help

## Purpose
Install a gitee-cli extension from a Gitee repository or a local
path. The source repository is cloned (or symlinked) into
~/.gitee-cli/extensions/<name>/, a manifest.json is written, and
the extension becomes immediately available as ` + "`" + `gitee <name>` + "`" + `.

## Pre-conditions
- git on PATH (for remote sources)
- For local sources: the path must exist and contain a ` + "`" + `<name>` + "`" + ` executable
- Write access to ~/.gitee-cli/extensions/

## Required arguments
  <source>   One of:
               owner/repo              (https://gitee.com/owner/repo.git)
               https://.../repo.git    (full URL)
               /abs/path               (local path; will be symlinked)
               ./relative/path         (local path; will be symlinked)
               .                       (current dir; expects gitee-<name> layout)

## Optional flags
  --force / -f   Overwrite an existing extension of the same name
  --name         Override the extension name (default: derived from source)

## Idempotency
NOT idempotent. Re-running with the same source creates a fresh
clone (or symlink) and updates the manifest timestamp. With
--force, the previous install is wiped first.

## Atomicity
Non-atomic. A failed clone leaves a half-written destination dir
(but no manifest is written, so the install is "rolled back"
from the loader's point of view — the broken dir is just left
behind for the user to inspect / remove).

## Side effects
- New directory under ~/.gitee-cli/extensions/<name>/
- ~/.gitee-cli/extensions/<name>/manifest.json (atomic write)
- Network: one ` + "`" + `git clone` + "`" + ` (for remote sources)

## Post-conditions (on success)
- ` + "`" + `gitee <name>` + "`" + ` resolves to the installed executable
- ` + "`" + `gitee extension list` + "`" + ` shows the new entry
- Manifest records Source, InstalledAt, PinnedRef (HEAD commit)

## Common errors
- "already installed"          -> use --force
- "git clone ... failed"       -> check network/repo URL
- "source ... is not a dir"    -> local path missing
- "symlink failed"             -> no symlink permission; copy manually

## Race condition warning ⚠️
The shared mutable state is the install dir
~/.gitee-cli/extensions/<name>/ plus its manifest.json. The
"already installed" stat check and the clone/symlink are NOT atomic:
two concurrent installs of the same name both pass the check, then
the second ` + "`" + `git clone` + "`" + ` fails with "already exists and is not an empty
directory" — or, with --force, one run's RemoveAll wipes the other's
fresh clone mid-write. Concurrent manifest writes are
last-writer-wins (each write is atomic, but there is no
cross-process lock). Also note the clone pins whatever HEAD was at
fetch time: if the upstream repo receives a push mid-install,
PinnedRef records the older commit and ` + "`" + `gitee extension upgrade` + "`" + `
will pull the newer one later. Serialize same-name installs across
agents; retry after a failure is safe with --force, which wipes the
partial destination first.

## OAuth scope required
None (no Gitee API call; uses git over HTTPS).

## Rate limit
git clone is not rate-limited by Gitee the way the API is.

## Example
  gitee extension install andershsueh/gitee-pr-stats
  # → installs to ~/.gitee-cli/extensions/pr-stats/
  # → available as: gitee pr-stats

## See also
- ` + "`" + `gitee extension list` + "`" + `   — see what's installed
- ` + "`" + `gitee extension upgrade` + "`" + ` — pull the latest for an existing one
- ` + "`" + `gitee extension create` + "`" + `  — scaffold a brand new one
`
