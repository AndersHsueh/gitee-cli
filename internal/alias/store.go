// Package alias stores user-defined command shortcuts in a flat
// JSON object at ~/.gitee-cli/aliases.json. Each entry maps a
// short name (e.g. "co") to a gitee-cli command expansion (e.g.
// "pr checkout"). Resolution into os.Args happens in cmd.Execute();
// this package only owns persistence.
package alias

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gitee-cli/internal/atomicfile"
	"gitee-cli/internal/utils"
)

// File returns the absolute path to the aliases JSON file.
//
// Priority:
//
//  1. $GITEE_ALIASES_FILE (user override — useful for portable
//     dotfiles; tests set this via t.Setenv to point at t.TempDir)
//  2. ~/.gitee-cli/aliases.json (default)
//
// The directory is *not* created here. Save creates the parent dir
// on first write. Load tolerates a missing file.
func File() string {
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
		if err := utils.ValidateName("alias", k); err != nil {
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
		if err := utils.ValidateName("alias", k); err != nil {
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
	return atomicfile.Write(path, data, 0600)
}

// Set adds or overwrites a single alias. It loads the current file,
// applies the change, and saves atomically. The returned previous
// value (and true) lets callers distinguish "created" from
// "updated" without a second read.
func Set(name, expansion string) (prev string, hadPrev bool, err error) {
	if err := utils.ValidateName("alias", name); err != nil {
		return "", false, err
	}
	expansion = strings.TrimSpace(expansion)
	if expansion == "" {
		return "", false, fmt.Errorf("alias expansion is empty")
	}
	m, loadErr := Load()
	if loadErr != nil {
		return "", false, loadErr
	}
	prev, hadPrev = m[name]
	m[name] = expansion
	if err := Save(m); err != nil {
		return "", false, err
	}
	return prev, hadPrev, nil
}

// Delete removes a single alias. Returns an error if the alias does
// not exist — callers can check with Get first if they want a
// silent miss.
func Delete(name string) error {
	if err := utils.ValidateName("alias", name); err != nil {
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
//
// (validateName was extracted to utils.ValidateName; the alias and
// extension stores now share one source of truth. Call
// utils.ValidateName("alias", name).)
