package alias

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// withTempFile points the package at a fresh temp file and registers
// cleanup. All test cases use this so the user's real aliases.json
// is never touched.
func withTempFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "aliases.json")
	FileOverride = path
	t.Cleanup(func() { FileOverride = "" })
	return path
}

func TestLoadMissing(t *testing.T) {
	withTempFile(t) // FileOverride points at a path that does not exist yet
	m, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file should not error, got: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("Load on missing file should return empty map, got %d entries", len(m))
	}
}

func TestLoadEmpty(t *testing.T) {
	path := withTempFile(t)
	if err := os.WriteFile(path, []byte(""), 0600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	m, err := Load()
	if err != nil {
		t.Fatalf("Load on empty file: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("Load on empty file should return empty map, got %d entries", len(m))
	}
}

func TestLoadInvalid(t *testing.T) {
	path := withTempFile(t)
	if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := Load()
	if err == nil {
		t.Fatal("Load on corrupt JSON should error")
	}
}

func TestSetAndGet(t *testing.T) {
	withTempFile(t)
	if err := Set("co", "pr checkout"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok := Get("co")
	if !ok {
		t.Fatal("Get(co) returned ok=false after Set")
	}
	if got != "pr checkout" {
		t.Errorf("Get(co) = %q, want %q", got, "pr checkout")
	}
}

func TestSetOverwrites(t *testing.T) {
	withTempFile(t)
	if err := Set("co", "pr checkout"); err != nil {
		t.Fatalf("Set #1: %v", err)
	}
	if err := Set("co", "pr co"); err != nil {
		t.Fatalf("Set #2: %v", err)
	}
	got, ok := Get("co")
	if !ok {
		t.Fatal("Get(co) returned ok=false after overwrite")
	}
	if got != "pr co" {
		t.Errorf("Get(co) = %q, want %q (overwrite should replace)", got, "pr co")
	}
}

func TestSetRejectsBadName(t *testing.T) {
	withTempFile(t)
	cases := []struct {
		name string
		err  bool
	}{
		{"", true},
		{".hidden", true},
		{"-flag", true},
		{"with space", true},
		{"UPPER", true},
		{"with.dot", true},
		{"valid_name", false},
		{"valid-name", false},
		{"a-1_b", false},
	}
	for _, tc := range cases {
		err := Set(tc.name, "x")
		gotErr := err != nil
		if gotErr != tc.err {
			t.Errorf("Set(%q) err=%v, wantErr=%v", tc.name, err, tc.err)
		}
	}
}

func TestSetRejectsEmptyExpansion(t *testing.T) {
	withTempFile(t)
	if err := Set("co", "   "); err == nil {
		t.Error("Set with whitespace expansion should error")
	}
}

func TestSetRejectsTooLongName(t *testing.T) {
	withTempFile(t)
	long := make([]byte, 41)
	for i := range long {
		long[i] = 'a'
	}
	if err := Set(string(long), "x"); err == nil {
		t.Error("Set with 41-char name should error")
	}
}

func TestDelete(t *testing.T) {
	withTempFile(t)
	if err := Set("co", "pr checkout"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := Delete("co"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := Get("co"); ok {
		t.Error("Get(co) returned ok=true after Delete")
	}
}

func TestDeleteMissing(t *testing.T) {
	withTempFile(t)
	err := Delete("nope")
	if err == nil {
		t.Error("Delete of missing alias should error")
	}
}

func TestSorted(t *testing.T) {
	withTempFile(t)
	_ = Set("zeta", "z")
	_ = Set("alpha", "a")
	_ = Set("mu", "m")
	keys, m, err := Sorted()
	if err != nil {
		t.Fatalf("Sorted: %v", err)
	}
	want := []string{"alpha", "mu", "zeta"}
	if !equalStrings(keys, want) {
		t.Errorf("Sorted keys = %v, want %v", keys, want)
	}
	if m["alpha"] != "a" || m["mu"] != "m" || m["zeta"] != "z" {
		t.Errorf("Sorted values = %v, want alpha=a mu=m zeta=z", m)
	}
}

func TestSaveAtomic(t *testing.T) {
	path := withTempFile(t)
	// Pre-create a "good" file. Save should replace it atomically.
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"old":"value"}`), 0600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	m := map[string]string{"co": "pr checkout"}
	if err := Save(m); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Verify the file is now our map, not "old".
	got, err := Load()
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if got["old"] != "" {
		t.Errorf("Save should have replaced contents; old entry still present: %v", got)
	}
	if got["co"] != "pr checkout" {
		t.Errorf("Save did not write new entry: %v", got)
	}
	// No leftover temp files in the dir.
	dir := filepath.Dir(path)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".aliases-") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestSavePermission(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("file mode bits behave differently on Windows")
	}
	path := withTempFile(t)
	if err := Save(map[string]string{"co": "pr checkout"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := fi.Mode().Perm()
	if perm != 0600 {
		t.Errorf("aliases file perm = %o, want 0600", perm)
	}
}

func TestLoadRejectsBadStoredEntry(t *testing.T) {
	// If a manually-edited file contains an invalid name, Load must
	// fail loudly rather than silently dropping the entry — otherwise
	// the user's "saved" alias would never be visible.
	path := withTempFile(t)
	if err := os.WriteFile(path, []byte(`{"Bad-Name":"x"}`), 0600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := Load(); err == nil {
		t.Error("Load should error when stored file has invalid name")
	}
}

func TestEndToEnd(t *testing.T) {
	// Realistic flow: set, then set another, then delete one, then
	// list. Verifies the file evolves correctly across operations.
	withTempFile(t)
	if err := Set("co", "pr checkout"); err != nil {
		t.Fatalf("Set co: %v", err)
	}
	if err := Set("issues", "issue list"); err != nil {
		t.Fatalf("Set issues: %v", err)
	}
	if err := Set("co", "pr co"); err != nil {
		t.Fatalf("Set co (overwrite): %v", err)
	}
	if err := Delete("issues"); err != nil {
		t.Fatalf("Delete issues: %v", err)
	}
	keys, m, err := Sorted()
	if err != nil {
		t.Fatalf("Sorted: %v", err)
	}
	if !equalStrings(keys, []string{"co"}) {
		t.Errorf("keys = %v, want [co]", keys)
	}
	if m["co"] != "pr co" {
		t.Errorf("m[co] = %q, want %q", m["co"], "pr co")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// silence unused import warning on some build tags
var _ = sort.Strings
