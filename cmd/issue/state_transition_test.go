package issue

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

// IMPORTANT: tests in this file mutate package-level globals
// (cmd.GlobalAPIURL / Token / Timeout and the per-verb *Flags struct
// fields). They MUST run sequentially — do NOT add t.Parallel(): the
// shared globals would race.

// stateTransitionRunner abstracts the small differences between
// runClose and runReopen so the table-driven tests below can share the
// server / globals / flags boilerplate.
type stateTransitionRunner struct {
	name         string
	cobraCmd     *cobra.Command
	run          func(*cobra.Command, []string) error
	stateValue   string // "closed" or "open"
	ingVerb      string // "closing" or "reopening"
	impVerb      string // "close" or "reopen" (workaround instruction)
	setFlags     func(owner, repo string)
	restoreFlags func()
}

func closeRunner() stateTransitionRunner {
	origOwner, origRepo := closeFlags.owner, closeFlags.repo
	return stateTransitionRunner{
		name:       "close",
		cobraCmd:   closeCmd,
		run:        runClose,
		stateValue: "closed",
		ingVerb:    "closing",
		impVerb:    "close",
		setFlags: func(o, r string) {
			closeFlags.owner = o
			closeFlags.repo = r
		},
		restoreFlags: func() {
			closeFlags.owner = origOwner
			closeFlags.repo = origRepo
		},
	}
}

func reopenRunner() stateTransitionRunner {
	origOwner, origRepo := reopenFlags.owner, reopenFlags.repo
	return stateTransitionRunner{
		name:       "reopen",
		cobraCmd:   reopenCmd,
		run:        runReopen,
		stateValue: "open",
		ingVerb:    "reopening",
		impVerb:    "reopen",
		setFlags: func(o, r string) {
			reopenFlags.owner = o
			reopenFlags.repo = r
		},
		restoreFlags: func() {
			reopenFlags.owner = origOwner
			reopenFlags.repo = origRepo
		},
	}
}

// withStateTransitionMockServer starts an httptest server pointing at
// the cmd globals, registers t.Cleanup to restore the globals and
// close the server, and returns the server URL.
func withStateTransitionMockServer(t *testing.T, h http.HandlerFunc) {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	snap := struct {
		apiURL, token string
		timeout       int
	}{
		apiURL:  cmd.GlobalAPIURL,
		token:   cmd.GlobalToken,
		timeout: cmd.GlobalTimeout,
	}
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	t.Cleanup(func() {
		cmd.GlobalAPIURL = snap.apiURL
		cmd.GlobalToken = snap.token
		cmd.GlobalTimeout = snap.timeout
	})
}

// assertPatchIssuePath verifies the request method is PATCH and the
// path matches /repos/{owner}/issues/{number}. The Gitee v5 PATCH
// write path for issues DROPS {repo} from the URL (the read path still
// includes it) — `repo` is supplied via formData instead. An off-route
// call would indicate a regression in runClose / runReopen.
func assertPatchIssuePath(t *testing.T, r *http.Request, wantPath string) {
	t.Helper()
	if r.Method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", r.Method)
	}
	if r.URL.Path != wantPath {
		t.Errorf("unexpected path: got %s, want %s", r.URL.Path, wantPath)
	}
}

func TestRunStateTransition_ProjectOrEnterprise404(t *testing.T) {
	for _, r := range []stateTransitionRunner{closeRunner(), reopenRunner()} {
		t.Run(r.name, func(t *testing.T) {
			withStateTransitionMockServer(t, func(w http.ResponseWriter, req *http.Request) {
				// Gitee v5 write path drops {repo}; should be /repos/owner/issues/IK5FZX.
				assertPatchIssuePath(t, req, "/repos/owner/issues/IK5FZX")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"404 Not Found: project or enterprise not found"}`))
			})
			r.setFlags("owner", "repo")
			t.Cleanup(r.restoreFlags)

			err := r.run(r.cobraCmd, []string{"IK5FZX"})
			if err == nil {
				t.Fatal("expected error from run on 404, got nil")
			}
			msg := err.Error()
			for _, want := range []string{
				r.ingVerb + " issue",
				"HTTP 404",
				"project or enterprise",
				"Gitee v5 cannot toggle the state",
				"https://gitee.com/owner/repo/issues/IK5FZX",
				r.impVerb + " the issue manually at",
			} {
				if !strings.Contains(msg, want) {
					t.Errorf("error message missing %q\nfull message:\n%s", want, msg)
				}
			}
		})
	}
}

func TestRunStateTransition_SuccessNoWorkaround(t *testing.T) {
	for _, r := range []stateTransitionRunner{closeRunner(), reopenRunner()} {
		t.Run(r.name, func(t *testing.T) {
			withStateTransitionMockServer(t, func(w http.ResponseWriter, req *http.Request) {
				// Gitee v5 write path drops {repo}.
				assertPatchIssuePath(t, req, "/repos/owner/issues/42")
				// Body must be formData (NOT JSON) carrying `repo` and
				// `state`. We use io.ReadAll rather than r.Body.Read
				// into a ContentLength-sized buffer because Read is
				// allowed to short-read, which would leave the assertion
				// silently matching a zero-filled tail.
				if err := req.ParseForm(); err != nil {
					t.Fatalf("ParseForm: %v", err)
				}
				if req.FormValue("repo") != "repo" {
					t.Errorf("expected formData repo=repo, got %q", req.FormValue("repo"))
				}
				if req.FormValue("state") != r.stateValue {
					t.Errorf("expected formData state=%s, got %q", r.stateValue, req.FormValue("state"))
				}
				if ct := req.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-www-form-urlencoded") {
					t.Errorf("expected formData Content-Type, got %q", ct)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"number":"42","state":"` + r.stateValue + `","title":"x"}`))
			})
			r.setFlags("owner", "repo")
			t.Cleanup(r.restoreFlags)

			if err := r.run(r.cobraCmd, []string{"42"}); err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
		})
	}
}

func TestRunStateTransition_Generic404DoesNotShowWorkaround(t *testing.T) {
	for _, r := range []stateTransitionRunner{closeRunner(), reopenRunner()} {
		t.Run(r.name, func(t *testing.T) {
			withStateTransitionMockServer(t, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"404 Not Found: issue not found"}`))
			})
			r.setFlags("owner", "repo")
			t.Cleanup(r.restoreFlags)

			err := r.run(r.cobraCmd, []string{"42"})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			msg := err.Error()
			if !strings.Contains(msg, "HTTP 404") {
				t.Errorf("generic 404 should keep the HTTP 404 marker, got %q", msg)
			}
			if strings.Contains(msg, "Gitee v5 cannot toggle the state") {
				t.Errorf("generic 404 should NOT trigger workaround hint, got %q", msg)
			}
		})
	}
}

// TestStateTransitionAgentHelpDocuments404Workaround verifies that the
// shared agent-help note (projectOrEnterprise404Note) is reachable
// from both issueCloseAgentHelp and issueReopenAgentHelp. Without
// this, agents reading the help have no machine-readable hint that
// the "project or enterprise" 404 is an expected upstream limitation.
func TestStateTransitionAgentHelpDocuments404Workaround(t *testing.T) {
	for _, tc := range []struct {
		name string
		help string
	}{
		{"close", issueCloseAgentHelp},
		{"reopen", issueReopenAgentHelp},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, want := range []string{
				"project or enterprise",
				"https://gitee.com/{owner}/{repo}/issues/{number}",
				"Workaround",
			} {
				if !strings.Contains(tc.help, want) {
					t.Errorf("%s --agent-help missing 404 workaround note: %q", tc.name, want)
				}
			}
		})
	}
}
