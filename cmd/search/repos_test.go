package search

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

// isolateSearchGlobals points the search client at a test server and
// restores the globals it touches afterwards.
func isolateSearchGlobals(t *testing.T, apiURL string) {
	t.Helper()
	oldURL, oldOut, oldTimeout := cmd.GlobalAPIURL, cmd.GlobalOutput, cmd.GlobalTimeout
	t.Cleanup(func() {
		cmd.GlobalAPIURL = oldURL
		cmd.GlobalOutput = oldOut
		cmd.GlobalTimeout = oldTimeout
	})
	cmd.GlobalAPIURL = apiURL
	cmd.GlobalOutput = "table"
	cmd.GlobalTimeout = 5
	t.Setenv("GITEE_TOKEN", "test-token")
	t.Setenv("HOME", t.TempDir())
}

// captureStderr runs f with os.Stderr redirected to a pipe and returns
// everything written to it.
func captureStderr(t *testing.T, f func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()

	runErr := f()
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return string(out), runErr
}

// TestSearchReposEmptyResultPrintsStderrHint locks the P1 fix: an empty
// result from Gitee's (upstream-broken) repository search must explain
// itself on stderr and point at `gitee repo list --owner`, while leaving
// the exit code at 0 and stdout machine-clean.
func TestSearchReposEmptyResultPrintsStderrHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/search/repositories" {
			t.Errorf("unexpected request path %q", got)
		}
		if got := r.URL.Query().Get("q"); got != "cobra" {
			t.Errorf("query q = %q, want cobra", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	isolateSearchGlobals(t, srv.URL)

	stderr, err := captureStderr(t, func() error {
		return runSearchRepos(&cobra.Command{}, []string{"cobra"})
	})
	if err != nil {
		t.Fatalf("runSearchRepos on empty result should not fail (exit code stays 0): %v", err)
	}
	for _, want := range []string{"no repositories matched", "gitee repo list --owner"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr hint missing %q; got:\n%s", want, stderr)
		}
	}
}

// TestSearchReposEmptyResultJSONModeAlsoHints pins the contract for the
// most common agent invocation: `gitee search repos q -o json` returns
// `[]` on stdout (clean, parseable) AND explains itself on stderr. The
// hint is not table-only — an agent that parses stdout must still learn
// from stderr that the endpoint is broken upstream.
func TestSearchReposEmptyResultJSONModeAlsoHints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	isolateSearchGlobals(t, srv.URL)
	cmd.GlobalOutput = "json"

	stderr, err := captureStderr(t, func() error {
		return runSearchRepos(&cobra.Command{}, []string{"cobra"})
	})
	if err != nil {
		t.Fatalf("runSearchRepos: %v", err)
	}
	for _, want := range []string{"no repositories matched", "gitee repo list --owner"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("JSON mode stderr hint missing %q; got:\n%s", want, stderr)
		}
	}
}

// TestSearchReposNonEmptyHasNoHint is the control case: a populated
// response must not produce the empty-result hint.
func TestSearchReposNonEmptyHasNoHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"full_name":"aa/bb","language":"Go","stargazers_count":1,"forks_count":2,"description":"d"}]`))
	}))
	defer srv.Close()

	isolateSearchGlobals(t, srv.URL)

	stderr, err := captureStderr(t, func() error {
		return runSearchRepos(&cobra.Command{}, []string{"cobra"})
	})
	if err != nil {
		t.Fatalf("runSearchRepos: %v", err)
	}
	if strings.Contains(stderr, "no repositories matched") {
		t.Errorf("non-empty result should not print the empty-result hint; got:\n%s", stderr)
	}
}
