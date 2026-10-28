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

func TestWatchAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo watch",
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
		if !strings.Contains(repoWatchAgentHelp, want) {
			t.Errorf("umbrella watch agent help missing required section: %q", want)
		}
	}
}

func TestWatchWatchAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo watch watch",
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
		if !strings.Contains(repoWatchWatchAgentHelp, want) {
			t.Errorf("watch verb agent help missing required section: %q", want)
		}
	}
}

func TestWatchUnwatchAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo watch unwatch",
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
		if !strings.Contains(repoWatchUnwatchAgentHelp, want) {
			t.Errorf("unwatch verb agent help missing required section: %q", want)
		}
	}
}

// ----- flag coverage -----

// TestWatchFlagsCoveredInAgentHelp ensures every flag exposed by the
// watch subcommands is also documented in its agent help string.
func TestWatchFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// watch verb
		{"--repo", repoWatchWatchAgentHelp},
		{"-R", repoWatchWatchAgentHelp},
		{"--type", repoWatchWatchAgentHelp},
		{"-t", repoWatchWatchAgentHelp},

		// unwatch verb
		{"--repo", repoWatchUnwatchAgentHelp},
		{"-R", repoWatchUnwatchAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}

// ----- wire-format + endpoint contract tests (httptest) -----

// TestRepoWatchEndpoint verifies PUT /user/subscriptions/{owner}/{repo}
// is hit with watch_type in the query string (per Gitee v5 swagger
// convention; this endpoint does not take a JSON body).
func TestRepoWatchEndpoint(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotWatchTp string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotWatchTp = r.URL.Query().Get("watch_type")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	// Build the path the same way runRepoWatch does.
	path := api.RepoWatch("owner", "repo") + "?watch_type=watching"
	err := client.Put(t.Context(), path, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/user/subscriptions/owner/repo", gotPath)
	assert.Equal(t, "watching", gotWatchTp)
}

// TestRepoWatchEndpointReleasesOnly verifies the watch_type query
// parameter round-trips for each of the three documented values.
func TestRepoWatchEndpointReleasesOnly(t *testing.T) {
	cases := []string{"watching", "releases_only", "ignoring"}
	for _, typ := range cases {
		t.Run(typ, func(t *testing.T) {
			var gotWatchTp string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotWatchTp = r.URL.Query().Get("watch_type")
				w.WriteHeader(http.StatusNoContent)
			}))
			defer ts.Close()

			client := api.NewClient("test-token", ts.URL, false, 30)
			path := api.RepoWatch("o", "r") + "?watch_type=" + typ
			require.NoError(t, client.Put(t.Context(), path, nil, nil))
			assert.Equal(t, typ, gotWatchTp)
		})
	}
}

// TestRepoUnwatchEndpoint verifies DELETE /user/subscriptions/{owner}/{repo}
// is hit with no body and accepts a 204 No Content response.
func TestRepoUnwatchEndpoint(t *testing.T) {
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
	err := client.Delete(t.Context(), api.RepoUnwatch("owner", "repo"))
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/user/subscriptions/owner/repo", gotPath)
}

// TestResolveWatchRepoArg validates the positional-arg-first helper used
// by watch/unwatch to pick the repo target.
func TestResolveWatchRepoArg(t *testing.T) {
	// Reset shared flags before each sub-case.
	reset := func() { watchFlags.repo = "" }

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
			watchFlags.repo = c.flagVal
			got, err := resolveWatchRepoArg(c.args)
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
				t.Errorf("resolveWatchRepoArg = (%q, %q), want (%q, %q)",
					got[0], got[1], c.wantOwner, c.wantRepo)
			}
		})
	}
}
