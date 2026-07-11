package extension

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitee-cli/internal/utils"
)

// withTempExtDir points ExtDirOverride at a fresh temp dir for the
// duration of the test. Restores the previous value on cleanup.
func withTempExtDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := ExtDirOverride
	ExtDirOverride = dir
	t.Cleanup(func() { ExtDirOverride = prev })
	return dir
}

func TestExtDirOverride(t *testing.T) {
	dir := withTempExtDir(t)
	if got := ExtDir(); got != dir {
		t.Errorf("ExtDir() = %q, want %q", got, dir)
	}
}

func TestExtPath(t *testing.T) {
	withTempExtDir(t)
	want := filepath.Join(ExtDir(), "mytool")
	if got := ExtPath("mytool"); got != want {
		t.Errorf("ExtPath() = %q, want %q", got, want)
	}
}

func TestResolveName(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOk bool
	}{
		{"foo", "foo", true},
		{"gitee-foo", "foo", true},
		{"gh-foo", "foo", true},
		{"my-cool_tool", "my-cool_tool", true},
		{"a1b2c3", "a1b2c3", true},
		{"", "", false},
		{".hidden", "", false},
		{"-dash", "", false},
		{"with space", "", false},
		{"with/slash", "", false},
		{"with.dot", "", false},
		{strings.Repeat("a", 41), "", false},
		{"emoji-🎉", "", false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := ResolveName(c.in)
			ok := err == nil
			if ok != c.wantOk {
				t.Fatalf("resolveName(%q) ok=%v wantOk=%v err=%v", c.in, ok, c.wantOk, err)
			}
			if ok && got != c.want {
				t.Errorf("resolveName(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestManifestRoundTrip(t *testing.T) {
	withTempExtDir(t)
	original := &Manifest{
		Name:        "myext",
		Path:        ExtPath("myext"),
		Source:      "owner/repo",
		InstalledAt: time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC),
		PinnedRef:   "abc123def",
	}
	if err := WriteManifest(original); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	got, err := ReadManifest("myext")
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got.Name != original.Name {
		t.Errorf("Name: got %q, want %q", got.Name, original.Name)
	}
	if got.Source != original.Source {
		t.Errorf("Source: got %q, want %q", got.Source, original.Source)
	}
	if !got.InstalledAt.Equal(original.InstalledAt) {
		t.Errorf("InstalledAt: got %v, want %v", got.InstalledAt, original.InstalledAt)
	}
	if got.PinnedRef != original.PinnedRef {
		t.Errorf("PinnedRef: got %q, want %q", got.PinnedRef, original.PinnedRef)
	}
}

func TestManifestFileMode(t *testing.T) {
	withTempExtDir(t)
	if err := WriteManifest(&Manifest{Name: "x", Source: "y"}); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	info, err := os.Stat(manifestPath("x"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// On macOS /tmp is a symlink so umask may not apply, but we
	// explicitly Chmod 0600 in WriteManifest. Check the file mode
	// masked to permission bits.
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("manifest mode = %o, want 0600", mode)
	}
}

func TestListInstalledEmpty(t *testing.T) {
	withTempExtDir(t)
	got, err := ListInstalled()
	if err != nil {
		t.Fatalf("ListInstalled: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ListInstalled on empty dir = %d items, want 0", len(got))
	}
}

func TestListInstalledMissingDir(t *testing.T) {
	// Override to a path that doesn't exist yet.
	ExtDirOverride = filepath.Join(t.TempDir(), "nonexistent")
	t.Cleanup(func() { ExtDirOverride = "" })

	got, err := ListInstalled()
	if err != nil {
		t.Errorf("ListInstalled on missing dir should NOT error, got: %v", err)
	}
	if got != nil {
		t.Errorf("ListInstalled on missing dir = %v, want nil", got)
	}
}

func TestListInstalledSkipsBadDirs(t *testing.T) {
	dir := withTempExtDir(t)
	// A valid extension
	if err := WriteManifest(&Manifest{Name: "good", Source: "x"}); err != nil {
		t.Fatalf("WriteManifest good: %v", err)
	}
	// A directory with no manifest (half-installed / junk)
	if err := os.MkdirAll(filepath.Join(dir, "half-baked"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// A hidden directory (left-over from a crashed install)
	if err := os.MkdirAll(filepath.Join(dir, ".tmp-junk"), 0755); err != nil {
		t.Fatalf("mkdir hidden: %v", err)
	}
	// A regular file at the ExtDir root (should be ignored)
	if err := os.WriteFile(filepath.Join(dir, "stray.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("writefile: %v", err)
	}

	got, err := ListInstalled()
	if err != nil {
		t.Fatalf("ListInstalled: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("ListInstalled = %d items, want 1 (good)", len(got))
	}
	if len(got) == 1 && got[0].Name != "good" {
		t.Errorf("ListInstalled[0].Name = %q, want %q", got[0].Name, "good")
	}
}

func TestRemove(t *testing.T) {
	withTempExtDir(t)
	if err := WriteManifest(&Manifest{Name: "rm-me", Source: "x"}); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	// Touch the executable too so we can confirm RemoveAll cleans it
	exe := filepath.Join(ExtPath("rm-me"), "rm-me")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write exe: %v", err)
	}
	if err := Remove("rm-me"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(ExtPath("rm-me")); !os.IsNotExist(err) {
		t.Errorf("ext dir still exists after Remove: stat err = %v", err)
	}
	// Re-remove is a no-op error (not a panic).
	if err := Remove("rm-me"); err == nil {
		t.Error("Remove of missing extension should error")
	}
}

func TestValidateNameRejectsReserved(t *testing.T) {
	// Reserved names that would conflict with built-in cobra commands.
	// We don't hard-fail on these in utils.ValidateName (too brittle — list
	// could drift), but resolveName + the loader check the live
	// cobra command set. This test just documents the current policy
	// for utils.ValidateName: it allows these (loader catches them).
	for _, n := range []string{"auth", "issue", "pr", "repo", "extension"} {
		if err := utils.ValidateName("extension", n); err != nil {
			t.Errorf("utils.ValidateName(%q) unexpectedly rejected: %v", n, err)
		}
	}
}
