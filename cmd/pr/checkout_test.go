package pr

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

// TestCheckoutNilHeadNoPanic verifies that runCheckout does not panic
// when the API returns a PR with a nil Head pointer. Before the fix,
// checkout.go line 144 dereferenced pr.Head.Ref directly, which
// panicked in the --no-fetch + API-failure path (pr stays zero-value,
// Head == nil).
//
// The test uses --no-fetch (so no git fetch is attempted) and a mock
// API server that returns a PR with head=null. To reach the
// Head-printing line (line 144) where the panic used to occur, we need
// `git checkout <branch>` to succeed. We use --force + a branch that
// is the current HEAD: `git branch -D <current>` fails harmlessly
// (cannot delete the current branch) and is ignored, then `git
// checkout <current>` succeeds as a no-op.
func TestCheckoutNilHeadNoPanic(t *testing.T) {
	// We need a real git working tree for the checkout step to succeed.
	// Skip if not inside a git repo.
	if _, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Output(); err != nil {
		t.Skip("not inside a git repo; skipping checkout nil-guard test")
	}

	// Record the current branch so we can restore it after the test.
	origBranchBytes, _ := exec.Command("git", "branch", "--show-current").Output()
	origBranch := strings.TrimSpace(string(origBranchBytes))
	if origBranch == "" {
		t.Skip("cannot determine current branch; skipping checkout nil-guard test")
	}

	// Mock API: return a PR with Head=null (nil *PRBranch).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pr := map[string]interface{}{
			"number":   99999,
			"title":    "Test PR nil head",
			"state":    "open",
			"html_url": "https://gitee.com/owner/repo/pulls/99999",
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
	origRepo := checkoutFlags.repo
	origBranchFlag := checkoutFlags.branch
	origNoFetch := checkoutFlags.noFetch
	origForce := checkoutFlags.force
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		checkoutFlags.repo = origRepo
		checkoutFlags.branch = origBranchFlag
		checkoutFlags.noFetch = origNoFetch
		checkoutFlags.force = origForce
		// Restore the original branch.
		_ = exec.Command("git", "checkout", origBranch).Run()
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	checkoutFlags.repo = "owner/repo"
	// Use the current branch as the checkout target. With --force,
	// `git branch -D <current>` fails harmlessly (ignored), and
	// `git checkout <current>` is a no-op that succeeds.
	checkoutFlags.branch = origBranch
	checkoutFlags.noFetch = true
	checkoutFlags.force = true

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// If the nil guard is missing, runCheckout panics at line 144.
	// Use a recover wrapper to turn a panic into a test failure rather
	// than crashing the test binary.
	ran := false
	func() {
		defer func() {
			if rcv := recover(); rcv != nil {
				// Restore stdout before failing so test output is visible.
				os.Stdout = oldStdout
				_ = w.Close()
				t.Fatalf("runCheckout panicked with nil Head: %v", rcv)
			}
		}()
		if err := runCheckout(checkoutCmd, []string{"99999"}); err != nil {
			// The git checkout step should succeed; any error here is
			// unexpected but the key assertion is "no panic".
			t.Logf("runCheckout returned error: %v", err)
		}
		ran = true
	}()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !ran {
		t.Fatal("runCheckout did not complete")
	}

	// The Head line must NOT appear because pr.Head is nil. If the old
	// buggy code ran, it would have panicked here — so reaching this
	// assertion at all is the success condition.
	if strings.Contains(output, "Head:") {
		t.Errorf("expected no Head line for nil pr.Head, but got:\n%s", output)
	}
}
