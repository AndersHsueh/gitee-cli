package repo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee-cli/internal/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- agent help content (per-verb) -----

func TestProtectAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo protect",
		"## Purpose",
		"## Subcommands",
		"## Pre-conditions",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"OAuth scope",
		"## Rate limit",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoProtectAgentHelp, want) {
			t.Errorf("umbrella protect agent help missing required section: %q", want)
		}
	}
}

func TestProtectProtectAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo protect protect",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags / arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions (on success)",
		"## Output (stdout, table mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoProtectProtectAgentHelp, want) {
			t.Errorf("protect verb agent help missing required section: %q", want)
		}
	}
}

func TestProtectUnprotectAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo protect unprotect",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags / arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions (on success)",
		"## Output (stdout, table mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoProtectUnprotectAgentHelp, want) {
			t.Errorf("unprotect verb agent help missing required section: %q", want)
		}
	}
}

// ----- flag coverage -----

// TestProtectFlagsCoveredInAgentHelp ensures every flag exposed by
// the protect / unprotect subcommands is also documented in its
// agent help string.
func TestProtectFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// protect verb
		{"--repo", repoProtectProtectAgentHelp},
		{"-R", repoProtectProtectAgentHelp},

		// unprotect verb (under umbrella + top-level alias share the
		// same help string)
		{"--repo", repoProtectUnprotectAgentHelp},
		{"-R", repoProtectUnprotectAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}

// ----- wire-format + endpoint contract tests (httptest) -----

// TestRepoBranchProtectionEndpoint verifies PUT
// /repos/{owner}/{repo}/branches/{branch}/protection is hit with no
// body and accepts a 204 No Content response (the canonical Gitee
// success shape for branch protection toggles).
func TestRepoBranchProtectionEndpoint(t *testing.T) {
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

	client := api.NewClient("test-token", ts.URL, false, 30)
	err := client.Put(t.Context(), api.RepoBranchProtection("owner", "repo", "main"), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/repos/owner/repo/branches/main/protection", gotPath)
}

// TestRepoUnprotectBranchEndpoint verifies DELETE
// /repos/{owner}/{repo}/branches/{branch}/protection is hit with no
// body and accepts a 204 No Content response.
func TestRepoUnprotectBranchEndpoint(t *testing.T) {
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

	client := api.NewClient("test-token", ts.URL, false, 30)
	err := client.Delete(t.Context(), api.RepoUnprotectBranch("owner", "repo", "main"))
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/repos/owner/repo/branches/main/protection", gotPath)
}

// TestRepoBranchProtectionEscapesBranchName verifies branch names
// containing slashes are URL-escaped so the path segment doesn't
// break Gitee's routing. e.g. "release/v1" must become
// release%2Fv1 in the endpoint builder output. Resty's HTTP client
// then percent-decodes the path before sending, so the server sees
// the canonical segment form — we assert against both: the builder
// output (escaped) and the wire path (decoded).
func TestRepoBranchProtectionEscapesBranchName(t *testing.T) {
	// Builder-level: the segment must be escaped.
	assert.Equal(t,
		"/repos/owner/repo/branches/release%2Fv1/protection",
		api.RepoBranchProtection("owner", "repo", "release/v1"),
		"endpoint builder must escape slashes in branch names")
	assert.Equal(t,
		"/repos/owner/repo/branches/release%2Fv1/protection",
		api.RepoUnprotectBranch("owner", "repo", "release/v1"),
		"endpoint builder must escape slashes in branch names")

	// Wire-level: resty decodes %2F to / before sending, so the test
	// server observes the canonical path. This guards against an
	// accidental double-escape that would surface as %252F.
	var gotMethod string
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	require.NoError(t, client.Put(t.Context(), api.RepoBranchProtection("owner", "repo", "release/v1"), nil, nil))
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/repos/owner/repo/branches/release/v1/protection", gotPath)

	gotMethod, gotPath = "", ""
	require.NoError(t, client.Delete(t.Context(), api.RepoUnprotectBranch("owner", "repo", "release/v1")))
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/repos/owner/repo/branches/release/v1/protection", gotPath)
}

// TestResolveProtectBranchArg validates the helper that requires a
// non-empty positional branch argument.
func TestResolveProtectBranchArg(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{"plain branch", []string{"main"}, "main", false},
		{"branch with slash", []string{"release/v1"}, "release/v1", false},
		{"no args", nil, "", true},
		{"empty arg", []string{""}, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolveProtectBranchArg(c.args)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error, got branch=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("resolveProtectBranchArg = %q, want %q", got, c.want)
			}
		})
	}
}

// TestProtectAndUnprotectSharePath pins the invariant that protect
// and unprotect hit the same path — only the HTTP verb differs.
// This guards against an accidental drift in the endpoint builders.
func TestProtectAndUnprotectSharePath(t *testing.T) {
	assert.Equal(t,
		api.RepoBranchProtection("o", "r", "main"),
		api.RepoUnprotectBranch("o", "r", "main"),
		"protect and unprotect must share the same URL path; only the verb differs")
}
