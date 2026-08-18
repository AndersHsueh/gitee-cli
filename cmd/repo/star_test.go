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

func TestStarAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo star",
		"## Purpose",
		"## Subcommands",
		"## Pre-conditions",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## OAuth scope required",
		"## Rate limit",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoStarAgentHelp, want) {
			t.Errorf("umbrella star agent help missing required section: %q", want)
		}
	}
}

func TestStarStarAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo star",
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
		if !strings.Contains(repoStarStarAgentHelp, want) {
			t.Errorf("star verb agent help missing required section: %q", want)
		}
	}
}

func TestStarUnstarAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo unstar",
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
		if !strings.Contains(repoStarUnstarAgentHelp, want) {
			t.Errorf("unstar verb agent help missing required section: %q", want)
		}
	}
}

func TestStarListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo star list",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoStarListAgentHelp, want) {
			t.Errorf("star list agent help missing required section: %q", want)
		}
	}
}

// ----- flag coverage -----

// TestStarFlagsCoveredInAgentHelp ensures every flag exposed by the
// star subcommands is also documented in its agent help string.
func TestStarFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// star verb
		{"--repo", repoStarStarAgentHelp},
		{"-R", repoStarStarAgentHelp},

		// unstar verb
		{"--repo", repoStarUnstarAgentHelp},
		{"-R", repoStarUnstarAgentHelp},

		// list verb
		{"--limit", repoStarListAgentHelp},
		{"-L", repoStarListAgentHelp},
		{"--sort", repoStarListAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}

// ----- wire-format + endpoint contract tests (httptest) -----

// TestRepoStarEndpoint verifies PUT /user/starred/{owner}/{repo} is hit
// with no body and accepts a 204 No Content response (the canonical
// Gitee success shape for star/unstar).
func TestRepoStarEndpoint(t *testing.T) {
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
	err := client.Put(t.Context(), api.RepoStar("owner", "repo"), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/user/starred/owner/repo", gotPath)
}

// TestRepoUnstarEndpoint verifies DELETE /user/starred/{owner}/{repo}
// is hit with no body and accepts a 204 No Content response.
func TestRepoUnstarEndpoint(t *testing.T) {
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
	err := client.Delete(t.Context(), api.RepoUnstar("owner", "repo"))
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/user/starred/owner/repo", gotPath)
}

// TestUserStarredListEndpoint verifies GET /user/starred returns the
// expected JSON array shape compatible with []types.Repository.
func TestUserStarredListEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1,"full_name":"o/r1","private":false,"description":"one","language":"Go","stargazers_count":42,"html_url":"https://gitee.com/o/r1"},
			{"id":2,"full_name":"o/r2","private":true,"description":"two","language":"","stargazers_count":3,"html_url":"https://gitee.com/o/r2"}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	// Path is a constant — verify it's /user/starred.
	assert.Equal(t, "/user/starred", api.UserStarredList())

	// Round-trip through the same call site the verb uses (with query
	// params appended). Strip the query string for the path check.
	var repos []struct {
		ID         int64  `json:"id"`
		FullName   string `json:"full_name"`
		Private    bool   `json:"private"`
		Language   string `json:"language"`
		StarsCount int    `json:"stargazers_count"`
	}
	fullPath := api.UserStarredList() + "?sort=created&per_page=30"
	err := client.Get(t.Context(), fullPath, &repos)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/user/starred", gotPath)
	require.Len(t, repos, 2)
	assert.Equal(t, "o/r1", repos[0].FullName)
	assert.Equal(t, 42, repos[0].StarsCount)
	assert.Equal(t, "Go", repos[0].Language)
	assert.Equal(t, "o/r2", repos[1].FullName)
	assert.True(t, repos[1].Private)
}

// TestResolveStarRepoArg validates the positional-arg-first helper used
// by star/unstar to pick the repo target.
func TestResolveStarRepoArg(t *testing.T) {
	// Reset shared flag before each sub-case.
	reset := func() { starFlags.repo = "" }

	cases := []struct {
		name      string
		flagVal   string
		args      []string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"positional arg only", "", []string{"alice/repo"}, "alice", "repo", false},
		{"flag only", "bob/svc", nil, "bob", "svc", false},
		{"flag only, no args", "carol/api", nil, "carol", "api", false},
		// Positional arg wins over flag.
		{"positional wins over flag", "flag/ignored", []string{"pos/wins"}, "pos", "wins", false},
		{"neither", "", nil, "", "", true},
		{"empty positional", "", []string{""}, "", "", true},
		{"invalid positional", "", []string{"noslash"}, "", "", true},
		{"empty flag, no args", "", nil, "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reset()
			starFlags.repo = c.flagVal
			got, err := resolveStarRepoArg(c.args)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error, got owner=%q repo=%q", got[0], got[1])
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got[0] != c.wantOwner || got[1] != c.wantRepo {
				t.Errorf("resolveStarRepoArg = (%q, %q), want (%q, %q)",
					got[0], got[1], c.wantOwner, c.wantRepo)
			}
		})
	}
}