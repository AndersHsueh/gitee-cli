package gist

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

// resetViewFlags restores viewCmd flag state between tests so package-level
// viewFlags doesn't leak across cases.
func resetViewFlags() {
	viewFlags.raw = false
}

// TestGistViewRawFetchFailureEmitsStderrWarning verifies that when the
// raw_url fetch for a file fails (here: the test server returns 404 for
// the raw path), the failure is NOT silently swallowed — a per-file
// warning is printed to stderr and the command still completes. This
// locks in the fix for the silent-error-swallow bug where a failed
// raw_url fetch made files look empty.
func TestGistViewRawFetchFailureEmitsStderrWarning(t *testing.T) {
	var rawHit bool
	var rawURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gists/abc123def":
			w.Header().Set("Content-Type", "application/json")
			// A gist with one file whose inline content is empty but
			// raw_url is set — the case that triggers a raw fetch in
			// table (non-raw) mode.
			_, _ = w.Write([]byte(`{
				"id":"abc123def",
				"description":"d",
				"public":true,
				"html_url":"https://gitee.com/x/abc123def",
				"files":{
					"main.go":{"filename":"main.go","language":"Go","size":10,"raw_url":"` + rawURL + `","content":""}
				}
			}`))
		case "/raw/main.go":
			rawHit = true
			// Simulate a transient failure fetching the raw body.
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	rawURL = ts.URL + "/raw/main.go"

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	origOutput := cmd.GlobalOutput
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		cmd.GlobalOutput = origOutput
		resetViewFlags()
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	cmd.GlobalOutput = "table"
	viewFlags.raw = false // table mode → empty content triggers raw fetch

	// Capture stderr to assert the warning is emitted there (not stdout,
	// which would corrupt machine-parseable output).
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	errC := make(chan error, 1)
	go func() { errC <- runView(viewCmd, []string{"abc123def"}) }()

	// Drain stderr in a goroutine so the pipe doesn't block the writer.
	stderrC := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(rErr)
		stderrC <- string(b)
	}()

	runErr := <-errC
	wErr.Close()
	stderrOut := <-stderrC
	os.Stderr = oldStderr

	if runErr != nil {
		t.Fatalf("runView returned error: %v (expected nil — raw fetch failure must not abort the whole view)", runErr)
	}
	if !rawHit {
		t.Fatalf("expected the test server's /raw/main.go path to be hit, but it was not")
	}
	if !strings.Contains(stderrOut, "warning: fetching raw content for main.go") {
		t.Errorf("expected per-file stderr warning for main.go, got stderr:\n%s", stderrOut)
	}
}

// TestGistViewRawFlagBinding sanity-checks that the --raw flag is bound to
// the (renamed) viewFlags.raw field, not a misleadingly-named field.
func TestGistViewRawFlagBinding(t *testing.T) {
	resetViewFlags()
	// Reset to false first, then parse the flag on.
	viewFlags.raw = false
	if err := viewCmd.Flags().Parse([]string{"--raw"}); err != nil {
		t.Fatalf("parsing --raw: %v", err)
	}
	if !viewFlags.raw {
		t.Errorf("expected viewFlags.raw to be true after --raw")
	}
}
