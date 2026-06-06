package atomicfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.json")
	if err := os.WriteFile(path, []byte(`{"old":true}`), 0600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := Write(path, []byte(`{"new":true}`), 0600); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != `{"new":true}` {
		t.Errorf("got %q, want replaced content", got)
	}
}

func TestWritePermission(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("file mode bits behave differently on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.json")
	if err := Write(path, []byte("hi"), 0600); err != nil {
		t.Fatalf("Write: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0600 {
		t.Errorf("perm = %o, want 0600", perm)
	}
}

func TestWriteNoTempLeftover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.json")
	if err := Write(path, []byte("hi"), 0600); err != nil {
		t.Fatalf("Write: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != filepath.Base(path) && strings.HasPrefix(e.Name(), filepath.Base(path)) {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestWritePermissionZero(t *testing.T) {
	// perm=0 means "don't chmod"; umask decides. Just verify no error.
	dir := t.TempDir()
	path := filepath.Join(dir, "x.json")
	if err := Write(path, []byte("hi"), 0); err != nil {
		t.Fatalf("Write(perm=0): %v", err)
	}
	if _, err := os.ReadFile(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}
