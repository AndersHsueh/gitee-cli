// Package extension manages the gitee-cli extension subsystem: storage of
// installed extensions on disk, their per-extension manifests, and the
// loader that synthesizes cobra commands from them.
//
// Extension layout (mirrors gh):
//
//	~/.gitee-cli/extensions/<name>/<name>     // executable, the extension entry point
//	~/.gitee-cli/extensions/<name>/manifest.json
//
// The <name> directory is the clone of a `gitee-<name>` repository. The
// bare <name> executable inside is what `gitee <name> ...` ultimately
// invokes.
package extension

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitee-cli/internal/atomicfile"
	"gitee-cli/internal/utils"
)

// ExtDirOverride lets tests redirect ExtDir() to a temp directory
// without going through the user's home. Set to a non-empty path
// before any ExtDir() call; pass "" to use the default.
var ExtDirOverride string

// ExtDir returns the root directory where extensions are installed.
// Priority:
//
//  1. ExtDirOverride (test hook)
//  2. $GITEE_EXT_DIR (user override)
//  3. ~/.gitee-cli/extensions (default)
//
// The directory is *not* created here — callers that need it to exist
// (e.g. Install) should MkdirAll explicitly. Read-only callers
// (e.g. ListInstalled) tolerate a missing directory.
func ExtDir() string {
	if ExtDirOverride != "" {
		return ExtDirOverride
	}
	if v := os.Getenv("GITEE_EXT_DIR"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".gitee-cli", "extensions")
}

// ExtPath returns the directory for a single named extension, e.g.
// ~/.gitee-cli/extensions/foo. The directory is not validated or
// created — caller decides.
func ExtPath(name string) string {
	return filepath.Join(ExtDir(), name)
}

// Manifest is the per-extension metadata stored at
// <ext-dir>/<name>/manifest.json. It records the source (so the user
// can re-install with `gitee extension install <source>`), the
// installation timestamp, and the current HEAD commit (so upgrade
// can show a clean diff vs. the previous state).
type Manifest struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Source      string    `json:"source"` // "owner/repo" or "/local/path"
	InstalledAt time.Time `json:"installed_at"`
	PinnedRef   string    `json:"pinned_ref,omitempty"` // HEAD commit SHA at install/upgrade
}

// ReadManifest loads the manifest for an installed extension.
// Returns os.ErrNotExist-compatible errors (wrapped) if the file
// or its parent directory is missing.
func ReadManifest(name string) (*Manifest, error) {
	if err := utils.ValidateName("extension", name); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(manifestPath(name))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest for %q: %w", name, err)
	}
	return &m, nil
}

// WriteManifest atomically writes m to <ext-dir>/<name>/manifest.json.
// Atomic via temp + rename in the same directory (POSIX & Windows).
func WriteManifest(m *Manifest) error {
	if err := utils.ValidateName("extension", m.Name); err != nil {
		return err
	}
	dir := ExtPath(m.Name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	final := manifestPath(m.Name)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(final, data, 0600)
}

// ListInstalled scans ExtDir for extension subdirectories and returns
// the manifests it can read. Directories without a manifest.json are
// silently skipped (they are not valid extensions). Returns an empty
// slice (not an error) if ExtDir itself does not exist.
//
// We stat each entry (not just e.IsDir()) so symlinks to directories
// (the format produced by `gitee extension install /local/path`)
// are recognised as valid extensions.
func ListInstalled() ([]*Manifest, error) {
	entries, err := os.ReadDir(ExtDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", ExtDir(), err)
	}
	var out []*Manifest
	for _, e := range entries {
		// Skip hidden dirs (e.g. ".tmp" from a crashed install).
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// Stat (not DirEntry.IsDir) so symlinks to directories pass.
		info, err := os.Stat(filepath.Join(ExtDir(), e.Name()))
		if err != nil || !info.IsDir() {
			continue
		}
		m, err := ReadManifest(e.Name())
		if err != nil {
			// Bad manifest is a real problem — surface it. But a
			// missing manifest (extension was half-installed) is fine.
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("loading manifest for %q: %w", e.Name(), err)
			}
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// Remove deletes the extension's directory (and its manifest with it).
// Returns os.ErrNotExist-compatible errors (wrapped) if the extension
// is not installed.
func Remove(name string) error {
	if err := utils.ValidateName("extension", name); err != nil {
		return err
	}
	dir := ExtPath(name)
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// resolveName canonicalizes a user-supplied extension name.
//
//   - "foo"            -> "foo"
//   - "gitee-foo"      -> "foo"   (strip our canonical prefix)
//   - "gh-foo"         -> "foo"   (tolerate the gh-style prefix too, in
//     case the user copy-pastes from gh docs)
//
// The result is what the user types as `gitee <name>`.
//
// Exported as ResolveName so the install verb can normalise the
// user-supplied --name override without duplicating the prefix logic.
func ResolveName(raw string) (string, error) {
	if err := utils.ValidateName("extension", raw); err != nil {
		return "", err
	}
	n := raw
	n = strings.TrimPrefix(n, "gitee-")
	n = strings.TrimPrefix(n, "gh-") // tolerate cross-pollination
	if err := utils.ValidateName("extension", n); err != nil {
		return "", err
	}
	return n, nil
}

// manifestPath is the on-disk location of an extension's manifest.
// Kept private — callers go through ReadManifest / WriteManifest.
func manifestPath(name string) string {
	return filepath.Join(ExtPath(name), "manifest.json")
}
