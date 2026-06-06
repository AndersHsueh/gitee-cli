// Package alias manages user-defined shell-style command aliases for
// gitee-cli. An alias maps a short name (e.g. "co") to a gitee-cli
// command expansion (e.g. "pr checkout"). After a user runs
// `gitee alias set co pr checkout`, typing `gitee co 123` at the shell
// is rewritten to `gitee pr checkout 123` before cobra parses it.
//
// Storage layout (mirrors gh's aliases):
//
//	~/.gitee-cli/aliases.json
//
// A flat JSON object: {"co": "pr checkout", "issues": "issue list"}.
// The file is created on first write; it is *not* created on read —
// Load returns an empty map when the file is missing.
//
// Aliases are resolved in cmd/gitee/main.go's Execute() wrapper: before
// rootCmd.Execute(), we inspect os.Args[1]. If it is not a recognized
// cobra command name and is a key in the alias map, we splice the
// expansion in. Built-in command names (rootCmd.Commands()) take
// precedence over alias entries — same convention as gh.
package alias

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileOverride lets tests redirect File() to a temp directory without
// touching the user's home. Set to a non-empty path before any File()
// call; pass "" to use the default. Tests should use t.TempDir().
//
// Mirrors the ExtDirOverride pattern in internal/extension/store.go.
var FileOverride string

// File returns the absolute path to the aliases JSON file.
//
// Priority:
//
//	1. FileOverride (test hook)
//	2. $GITEE_ALIASES_FILE (user override — useful for portable dotfiles)
//	3. ~/.gitee-cli/aliases.json (default)
//
// The directory is *not* created here. Save creates the parent dir
// on first write. Load tolerates a missing file.
func File() string {
	if FileOverride != "" {
		return FileOverride
	}
	if v := os.Getenv("GITEE_ALIASES_FILE"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".gitee-cli", "aliases.json")
}

// Load reads the aliases file and returns the parsed map. An empty
// map (not an error) is returned when the file does not exist.
//
// Parse errors *are* errors — they mean the file is corrupt and the
// user should know (so they can fix or delete it). The error is
// wrapped with a hint about the file path.
func Load() (map[string]string, error) {
	data, err := os.ReadFile(File())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading aliases file %s: %w", File(), err)
	}
	out := map[string]string{}
	if len(data) == 0 {
		// Treat empty file as empty map (some editors save an
		// empty file on first write). json.Unmarshal of "" would
		// fail with "unexpected end of JSON input".
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parsing aliases file %s: %w (delete it to reset)", File(), err)
	}
	// Defensive: reject any entries that fail current validation so
	// the rest of the system can assume a loaded alias is valid.
	cleaned := make(map[string]string, len(out))
	for k, v := range out {
		if err := validateName(k); err != nil {
			return nil, fmt.Errorf("invalid alias name %q in %s: %w", k, File(), err)
		}
		if strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("alias %q in %s has empty expansion", k, File())
		}
		cleaned[k] = v
	}
	return cleaned, nil
}

// Save atomically writes m to the aliases file. The file is replaced
// via temp-file + rename in the same directory (POSIX & Windows).
// The final file is chmod 0600 — same convention as the credentials
// file (cmd/auth) and extension manifests.
//
// The parent directory is created on demand with 0700.
func Save(m map[string]string) error {
	// Validate every entry before touching disk. Refuse to clobber
	// a good file with a bad one.
	for k, v := range m {
		if err := validateName(k); err != nil {
			return err
		}
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("alias %q has empty expansion", k)
		}
	}
	path := File()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling aliases: %w", err)
	}
	// Write to a temp file in the same dir (so os.Rename is atomic
	// on POSIX; on Windows the rename may fail if dest exists — that
	// is acceptable; we re-replace via os.Rename which on Win
	// requires the dest not to exist. We Delete it first as a fallback.)
	tmp, err := os.CreateTemp(filepath.Dir(path), ".aliases-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	// Windows requires the target to be absent for os.Rename.
	// Since we just created tmpPath, no other goroutine should
	// have written `path` between MkdirAll and now. Be defensive
	// anyway — silently swallow ENOENT on the unlink.
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("renaming %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

// Set adds or overwrites a single alias. It loads the current file,
// applies the change, and saves atomically.
func Set(name, expansion string) error {
	if err := validateName(name); err != nil {
		return err
	}
	expansion = strings.TrimSpace(expansion)
	if expansion == "" {
		return fmt.Errorf("alias expansion is empty")
	}
	m, err := Load()
	if err != nil {
		return err
	}
	m[name] = expansion
	return Save(m)
}

// Delete removes a single alias. Returns an error if the alias does
// not exist — callers can check with Get first if they want a
// silent miss.
func Delete(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	m, err := Load()
	if err != nil {
		return err
	}
	if _, ok := m[name]; !ok {
		return fmt.Errorf("alias %q is not defined", name)
	}
	delete(m, name)
	return Save(m)
}

// Get returns the expansion for name, and whether it exists.
func Get(name string) (string, bool) {
	m, err := Load()
	if err != nil {
		return "", false
	}
	v, ok := m[name]
	return v, ok
}

// Sorted returns the alias map keys in sorted order, paired with
// their expansions. Convenient for table output and tests.
func Sorted() ([]string, map[string]string, error) {
	m, err := Load()
	if err != nil {
		return nil, nil, err
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, m, nil
}

// validateName rejects alias names that would shadow a built-in
// command path or that contain characters unusable in `gitee <name>`.
// The same rules apply as for extensions — keep it simple.
//
// Why these rules:
//   - Reject empty names (cobra would error anyway, but fail early).
//   - Reject anything starting with `.` or `-` to avoid surprise flags.
//   - Restrict to [a-z0-9_-] so the alias name is a valid cobra Use
//     segment, a valid POSIX-style short name, and safe to splice
//     into os.Args.
//   - 40-char cap mirrors the extension-name limit; if you need
//     something longer, write a real command.
func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("alias name is empty")
	}
	if len(name) > 40 {
		return fmt.Errorf("alias name %q too long (max 40 chars)", name)
	}
	if name[0] == '.' || name[0] == '-' {
		return fmt.Errorf("alias name %q must not start with '.' or '-'", name)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return fmt.Errorf("alias name %q contains invalid character %q (allowed: a-z, 0-9, -, _)", name, r)
		}
	}
	return nil
}
