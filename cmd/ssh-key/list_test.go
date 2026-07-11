package sshkey

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee ssh-key list",
		"Purpose",
		"Pre-conditions",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"OAuth scope required",
		"Rate limit",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(sshKeyListAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

// TestListSendsPerPageAndPage verifies that --limit and --page are
// actually sent to the API as per_page / page query params. Before
// the fix, runList used api.SSHKeysList() (bare /user/keys) and the
// flags were silently ignored.
func TestListSendsPerPageAndPage(t *testing.T) {
	var (
		gotPerPage string
		gotPage    string
		gotPath    string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		q := r.URL.Query()
		gotPerPage = q.Get("per_page")
		gotPage = q.Get("page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer ts.Close()

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	origLimit := listFlags.limit
	origPage := listFlags.page
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		listFlags.limit = origLimit
		listFlags.page = origPage
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	listFlags.limit = 50
	listFlags.page = 2

	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList failed: %v", err)
	}
	if gotPath != "/user/keys" {
		t.Errorf("expected path /user/keys, got %s", gotPath)
	}
	if gotPerPage != "50" {
		t.Errorf("expected per_page=50, got %q", gotPerPage)
	}
	if gotPage != "2" {
		t.Errorf("expected page=2, got %q", gotPage)
	}
}

// TestListClampsPerPageTo100 verifies --limit is clamped to the Gitee
// v5 hard max of 100 so a value like 200 is not sent as-is.
func TestListClampsPerPageTo100(t *testing.T) {
	var gotPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer ts.Close()

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	origLimit := listFlags.limit
	origPage := listFlags.page
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		listFlags.limit = origLimit
		listFlags.page = origPage
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	listFlags.limit = 200
	listFlags.page = 1

	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList failed: %v", err)
	}
	if gotPerPage != "100" {
		t.Errorf("expected per_page clamped to 100, got %q", gotPerPage)
	}
}
