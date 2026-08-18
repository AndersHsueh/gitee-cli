package repo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee-cli/cmd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEditAgentHelpDocumentsPublicFlag ensures the agent help mentions
// --public and the mutual exclusivity rule (issue #IK5FYX). Without
// this, agents reading the help can't figure out how to flip a repo
// from private back to public.
func TestEditAgentHelpDocumentsPublicFlag(t *testing.T) {
	mustContain := []string{
		"--public",
		"Mutually exclusive",
	}
	for _, want := range mustContain {
		if !strings.Contains(repoEditAgentHelp, want) {
			t.Errorf("repo edit agent help should mention %q", want)
		}
	}
}

// ----- wire-format integration tests (httptest) -----

// resetEditFlags restores package-level state between tests. We do
// two things:
//   - ResetFlags() rebuilds the FlagSet, which clears pflag's internal
//     Changed map (the only way to reset it short of discarding the
//     process).
//   - We then re-zero the bound struct fields. pflag's StringVar/BoolVar
//     stores a Go pointer to the field (e.g. &editFlags.name); the
//     pointer survives FlagSet teardown, so we must zero it ourselves.
//     registerEditFlags() also re-initializes the field to its default
//     when it re-binds, but doing it explicitly keeps the intent
//     obvious and protects against a future refactor that drops the
//     default values from the flag registration.
func resetEditFlags(t *testing.T) {
	t.Helper()
	editCmd.SetArgs(nil)
	editFlags.owner = ""
	editFlags.repo = ""
	editFlags.name = ""
	editFlags.description = ""
	editFlags.privateValue = false
	editFlags.publicValue = false
	editFlags.defaultBranch = ""
	editCmd.ResetFlags()
	registerEditFlags(editCmd)
}

// captureEditRequest runs `gitee repo edit <args>` against an
// httptest server and returns the parsed PATCH body the server
// received. It also registers cleanup that closes the server and
// restores the cmd package's API globals.
func captureEditRequest(t *testing.T, args []string) map[string]interface{} {
	t.Helper()

	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":1,
			"name":"test-repo",
			"full_name":"owner/test-repo",
			"private":false,
			"html_url":"https://gitee.com/owner/test-repo"
		}`))
	}))
	t.Cleanup(ts.Close)

	editCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{
		APIURL:  ts.URL,
		Token:   "test-token",
		Timeout: 30,
	}))
	require.NoError(t, editCmd.Flags().Parse(args))
	require.NoError(t, runEdit(editCmd, args))

	require.NotEmpty(t, gotBody, "PATCH body should not be empty")
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(gotBody, &body))
	return body
}

// expectNoEditRequest stands up an httptest server that fails the
// test if any request is made. Used to assert that cobra's mutual-
// exclusivity check short-circuits before any HTTP call.
func expectNoEditRequest(t *testing.T) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be called; got %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(ts.Close)
}

// TestRunEditIncludesRepoNameInBody — issue #IK5FYW.
//
// Gitee API v5 rejects PATCH /repos/{owner}/{repo} with HTTP 400
// "name is missing" when the body omits `name`. The fix is to always
// populate req.Name with the current repo name; --name overrides it
// for rename.
func TestRunEditIncludesRepoNameInBody(t *testing.T) {
	t.Cleanup(func() { resetEditFlags(t) })
	resetEditFlags(t)

	body := captureEditRequest(t, []string{
		"--repo", "owner/test-repo",
		"--description", "New description",
	})
	assert.Equal(t, "test-repo", body["name"], "name must be present and equal the current repo name")
	assert.Equal(t, "New description", body["description"])
}

// TestRunEditPublicFlag — issue #IK5FYX.
func TestRunEditPublicFlag(t *testing.T) {
	t.Cleanup(func() { resetEditFlags(t) })
	resetEditFlags(t)

	body := captureEditRequest(t, []string{
		"--repo", "owner/test-repo",
		"--public",
	})
	assert.Equal(t, "test-repo", body["name"])
	assert.Equal(t, false, body["private"], "private must be false when --public is set")
}

// TestRunEditPrivateFlag — guards against regression in --private.
func TestRunEditPrivateFlag(t *testing.T) {
	t.Cleanup(func() { resetEditFlags(t) })
	resetEditFlags(t)

	body := captureEditRequest(t, []string{
		"--repo", "owner/test-repo",
		"--private",
	})
	assert.Equal(t, "test-repo", body["name"])
	assert.Equal(t, true, body["private"], "private must be true when --private is set")
}

// TestRunEditPublicPrivateMutuallyExclusive — passing --public and
// --private together must be rejected by cobra before reaching the
// API.
func TestRunEditPublicPrivateMutuallyExclusive(t *testing.T) {
	t.Cleanup(func() { resetEditFlags(t) })
	resetEditFlags(t)

	expectNoEditRequest(t)

	RepoCmd.SetArgs([]string{"edit", "--repo", "owner/test-repo", "--public", "--private"})
	execErr := RepoCmd.Execute()
	require.Error(t, execErr, "passing --public and --private together must error")
	assert.Contains(t, strings.ToLower(execErr.Error()), "public",
		"error should mention --public; got: %v", execErr)
}

// TestRunEditNameOverride — --name should override the current repo
// name in the body (the rename case).
func TestRunEditNameOverride(t *testing.T) {
	t.Cleanup(func() { resetEditFlags(t) })
	resetEditFlags(t)

	body := captureEditRequest(t, []string{
		"--repo", "owner/old-name",
		"--name", "new-name",
	})
	assert.Equal(t, "new-name", body["name"], "--name should override the current repo name in the body")
}

// TestRunEditNoMutableFlagsEmptyBody — when only --repo is given,
// the body should still contain `name` (per #IK5FYW fix) but no other
// fields (PATCH treats this as a no-op).
func TestRunEditNoMutableFlagsEmptyBody(t *testing.T) {
	t.Cleanup(func() { resetEditFlags(t) })
	resetEditFlags(t)

	body := captureEditRequest(t, []string{"--repo", "owner/foo"})
	assert.Equal(t, "foo", body["name"], "name must be present even with no mutable flags")
	_, hasDescription := body["description"]
	assert.False(t, hasDescription, "description should be omitted when empty")
	_, hasPrivate := body["private"]
	assert.False(t, hasPrivate, "private should be omitted when --private/--public not set")
	_, hasDefaultBranch := body["default_branch"]
	assert.False(t, hasDefaultBranch, "default_branch should be omitted when empty")
}
