package release

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDownloadToSuccessWritesFullBody verifies the happy path: a 200
// response body is written to destPath in full and the file is closed
// (flushed) so the bytes are actually on disk.
func TestDownloadToSuccessWritesFullBody(t *testing.T) {
	body := "hello-release-asset"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer ts.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "asset.bin")
	if err := downloadTo(ts.URL, dest); err != nil {
		t.Fatalf("downloadTo: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(got) != body {
		t.Errorf("dest content = %q, want %q", got, body)
	}
}

// TestDownloadToHTTPErrorReturnsErrorAndNoFile verifies that a 4xx from
// the download URL returns an error AND does not create a dest file
// (Create hasn't been reached yet in that path, but we assert it to
// lock the contract).
func TestDownloadToHTTPErrorReturnsErrorAndNoFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "asset.bin")
	err := downloadTo(ts.URL, dest)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention status 404, got: %v", err)
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Errorf("dest file should not exist after a 404, but it does")
	}
}

// TestDownloadToServerHangupRemovesPartialFile verifies the core fix:
// when the server closes the connection mid-stream (so io.Copy fails
// after a partial write), the partial dest file is removed rather than
// left behind as a corrupt artifact that a later script might use.
func TestDownloadToServerHangupRemovesPartialFile(t *testing.T) {
	// Hijack the connection: write a Content-Length header claiming 100
	// bytes, send only 10, then close. io.Copy will get an
	// "unexpected EOF" and return an error — the fix must delete dest.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatalf("test server does not support hijacking")
		}
		conn, bufrw, _ := hj.Hijack()
		defer conn.Close()
		_, _ = bufrw.WriteString("HTTP/1.1 200 OK\r\n")
		_, _ = bufrw.WriteString("Content-Length: 100\r\n")
		_, _ = bufrw.WriteString("Connection: close\r\n\r\n")
		_, _ = bufrw.WriteString("0123456789") // only 10 of 100 bytes
		_ = bufrw.Flush()
		// Close mid-stream → io.Copy fails with unexpected EOF.
	}))
	defer ts.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "asset.bin")
	err := downloadTo(ts.URL, dest)
	if err == nil {
		t.Fatal("expected downloadTo to error on a truncated stream, got nil")
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Errorf("partial dest file should have been removed after io.Copy failure, but it exists")
	}
}
