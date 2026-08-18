package issue

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

// IMPORTANT: tests in this file mutate package-level globals
// (cmd.GlobalAPIURL / GlobalToken / GlobalTimeout and commentFlags).
// They MUST run sequentially — do NOT add t.Parallel(): the shared
// globals would race.

// withCommentMockServer starts an httptest server pointing at the cmd
// globals, registers t.Cleanup to restore the globals and close the
// server, and returns the server URL. Mirrors withStateTransitionMockServer.
func withCommentMockServer(t *testing.T, h http.HandlerFunc) {
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

// setCommentFlags sets the package-level commentFlags and registers a
// cleanup that restores the previous values.
func setCommentFlags(t *testing.T, owner, repo, body string) {
	t.Helper()
	origOwner, origRepo, origBody := commentFlags.owner, commentFlags.repo, commentFlags.body
	commentFlags.owner = owner
	commentFlags.repo = repo
	commentFlags.body = body
	t.Cleanup(func() {
		commentFlags.owner = origOwner
		commentFlags.repo = origRepo
		commentFlags.body = origBody
	})
}

// TestRunComment_401ErrorBodyPassthrough verifies that when Gitee
// returns HTTP 401 with a body carrying a real error message
// (field name `error`, e.g. "当前issue已关闭评论功能"), the CLI surfaces
// that message instead of the generic "authentication failed: run
// 'gitee auth login'" hint. Gitee v5 uses 401 for a locked-comments
// issue even though the token is perfectly valid; swallowing the body
// misleads the user into chasing a token problem that does not exist.
func TestRunComment_401ErrorBodyPassthrough(t *testing.T) {
	withCommentMockServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"当前issue已关闭评论功能"}`))
	})
	setCommentFlags(t, "owner", "repo", "x")

	err := runComment(commentCmd, []string{"42"})
	if err == nil {
		t.Fatal("expected error from 401, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "已关闭评论功能") {
		t.Errorf("expected the concrete Gitee error body to be surfaced, got: %q", msg)
	}
	if strings.Contains(msg, "auth login") {
		t.Errorf("locked-comments 401 is NOT an auth failure; must not suggest 'auth login', got: %q", msg)
	}
}

// TestRunComment_FormDataBody verifies that the issue comment write
// endpoint is called with an application/x-www-form-urlencoded body
// (NOT JSON). Gitee v5's issue comment endpoint rejects JSON bodies
// with HTTP 401; only form-encoded bodies reach the business layer.
func TestRunComment_FormDataBody(t *testing.T) {
	withCommentMockServer(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/repos/owner/repo/issues/42/comments" {
			t.Errorf("unexpected path: got %s", req.URL.Path)
		}
		if err := req.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := req.FormValue("body"); got != "x" {
			t.Errorf("expected formData body=x, got %q", got)
		}
		if ct := req.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-www-form-urlencoded") {
			t.Errorf("expected formData Content-Type, got %q", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":9001,"body":"x"}`))
	})
	setCommentFlags(t, "owner", "repo", "x")

	if err := runComment(commentCmd, []string{"42"}); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}
