package pr

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

// TestViewNilHeadBaseNoPanic verifies that runView does not panic when
// the API returns a PR with nil Head/Base pointers. Before the fix,
// view.go line 89-90 dereferenced pr.Base.Ref / pr.Head.Ref directly,
// causing a nil pointer panic when the head branch was deleted or the
// API response was missing those fields.
func TestViewNilHeadBaseNoPanic(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a PR with Head and Base as JSON null — they will
		// unmarshal to nil *PRBranch pointers.
		pr := map[string]interface{}{
			"number":   42,
			"title":    "Test PR",
			"state":    "open",
			"html_url": "https://gitee.com/owner/repo/pulls/42",
			"user":     map[string]interface{}{"login": "alice"},
			"head":     nil,
			"base":     nil,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pr)
	}))
	defer ts.Close()

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	origRepo := viewFlags.repo
	origOwner := viewFlags.owner
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		viewFlags.repo = origRepo
		viewFlags.owner = origOwner
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	viewFlags.repo = "owner/repo"
	viewFlags.owner = ""

	// Capture stdout to verify the fallback "-" is printed instead of
	// crashing.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	// If the nil guard is missing, this will panic (fatal).
	if err := runView(viewCmd, []string{"42"}); err != nil {
		t.Fatalf("runView failed: %v", err)
	}

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// tabwriter pads with spaces, so the literal tab is gone — match
	// on the label + fallback value on the same line.
	lines := strings.Split(output, "\n")
	baseLine := ""
	headLine := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Base:") {
			baseLine = trimmed
		}
		if strings.HasPrefix(trimmed, "Head:") {
			headLine = trimmed
		}
	}
	if !strings.HasSuffix(baseLine, "-") {
		t.Errorf("expected Base line to end with fallback '-', got %q", baseLine)
	}
	if !strings.HasSuffix(headLine, "-") {
		t.Errorf("expected Head line to end with fallback '-', got %q", headLine)
	}
}
