package repo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitee-cli/cmd"
)

// resetDeleteFlags restores deleteCmd flag state between tests.
func resetDeleteFlags() {
	deleteFlags.owner = ""
	deleteFlags.repo = ""
	deleteFlags.yes = false
}

// TestRepoDeleteNonInteractiveWithoutYesCancels verifies that in a
// non-interactive context (no TTY — which is the case under `go test`,
// since os.Stdin is not a character device) the confirmation prompt
// degrades to "no" and the command cancels WITHOUT hitting the API.
// This is the correct, safe behavior for CI/agents and the reason the
// command was switched from bare fmt.Scanln (which ignored its error)
// to utils.ConfirmYN (which handles EOF/non-TTY explicitly).
func TestRepoDeleteNonInteractiveWithoutYesCancels(t *testing.T) {
	var hitAPI bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitAPI = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		resetDeleteFlags()
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	deleteFlags.yes = false // force the prompt path

	err := runDelete(deleteCmd, []string{"owner/repo-to-delete"})
	if err != nil {
		t.Fatalf("runDelete returned error in non-interactive cancel: %v", err)
	}
	if hitAPI {
		t.Errorf("API was hit even though the confirmation should have cancelled (non-interactive stdin → no)")
	}
}

// TestRepoDeleteWithYesSkipsPromptAndDeletes verifies the --yes path
// skips the prompt and performs the DELETE against the API.
func TestRepoDeleteWithYesSkipsPromptAndDeletes(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		resetDeleteFlags()
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	deleteFlags.yes = true

	err := runDelete(deleteCmd, []string{"owner/repo-to-delete"})
	if err != nil {
		t.Fatalf("runDelete returned error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", gotMethod)
	}
	if gotPath != "/repos/owner/repo-to-delete" {
		t.Errorf("expected path /repos/owner/repo-to-delete, got %s", gotPath)
	}
}
