package repo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- agent help content (per-verb) -----

func TestCollaboratorAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo collaborator",
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
		if !strings.Contains(repoCollaboratorAgentHelp, want) {
			t.Errorf("collaborator agent help missing required section: %q", want)
		}
	}
}

func TestCollaboratorListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo collaborator list",
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
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoCollaboratorListAgentHelp, want) {
			t.Errorf("list agent help missing required section: %q", want)
		}
	}
}

func TestCollaboratorAddAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo collaborator add",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoCollaboratorAddAgentHelp, want) {
			t.Errorf("add agent help missing required section: %q", want)
		}
	}
}

func TestCollaboratorRemoveAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo collaborator remove",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoCollaboratorRemoveAgentHelp, want) {
			t.Errorf("remove agent help missing required section: %q", want)
		}
	}
}

// ----- flag coverage -----

// TestCollaboratorFlagsCoveredInAgentHelp ensures every flag exposed by
// the collaborator subcommands is also documented in its agent help
// string.
func TestCollaboratorFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// list
		{"--repo", repoCollaboratorListAgentHelp},
		{"-R", repoCollaboratorListAgentHelp},
		{"--limit", repoCollaboratorListAgentHelp},
		{"-L", repoCollaboratorListAgentHelp},

		// add
		{"--repo", repoCollaboratorAddAgentHelp},
		{"-R", repoCollaboratorAddAgentHelp},
		{"--username", repoCollaboratorAddAgentHelp},
		{"-u", repoCollaboratorAddAgentHelp},
		{"--permission", repoCollaboratorAddAgentHelp},
		{"-p", repoCollaboratorAddAgentHelp},

		// remove
		{"--repo", repoCollaboratorRemoveAgentHelp},
		{"-R", repoCollaboratorRemoveAgentHelp},
		{"--username", repoCollaboratorRemoveAgentHelp},
		{"-u", repoCollaboratorRemoveAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}

// ----- wire-format + endpoint contract tests (httptest) -----

// TestCollaboratorsListEndpoint verifies GET /repos/{owner}/{repo}/collaborators
// is hit and parses cleanly into []types.Collaborator.
func TestCollaboratorsListEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1,"login":"alice","name":"Alice","permission":"push","html_url":"https://gitee.com/alice"},
			{"id":2,"login":"bob","name":"Bob","permission":"admin","html_url":"https://gitee.com/bob"}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	var collabs []types.Collaborator
	err := client.Get(t.Context(), api.CollaboratorsList("owner", "repo"), &collabs)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/repos/owner/repo/collaborators", gotPath)
	require.Len(t, collabs, 2)
	assert.Equal(t, "alice", collabs[0].Login)
	assert.Equal(t, "Alice", collabs[0].Name)
	assert.Equal(t, "push", collabs[0].Permission)
	assert.Equal(t, "bob", collabs[1].Login)
	assert.Equal(t, "admin", collabs[1].Permission)
}

// TestCollaboratorAddEndpoint verifies the PUT body shape and path for
// adding a collaborator with a permission level.
func TestCollaboratorAddEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   types.CollaboratorAddRequest
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		// Gitee returns 204 No Content on success; we simulate the
		// optional response-body shape (some endpoints echo the user).
		_, _ = w.Write([]byte(`{"id":42,"login":"alice","name":"Alice","permission":"push"}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	req := types.CollaboratorAddRequest{Permission: "push"}
	var created types.Collaborator
	err := client.Put(t.Context(), api.CollaboratorAdd("owner", "repo", "alice"), req, &created)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/repos/owner/repo/collaborators/alice", gotPath)
	assert.Equal(t, "push", gotBody.Permission)
	assert.Equal(t, "alice", created.Login)
	assert.Equal(t, "push", created.Permission)
}

// TestCollaboratorAddRequestWireFormat verifies the JSON wire format
// of the request body matches what Gitee expects: a single optional
// `permission` field. The omitempty ensures absent fields don't leak.
func TestCollaboratorAddRequestWireFormat(t *testing.T) {
	// With permission set.
	with := types.CollaboratorAddRequest{Permission: "admin"}
	data, err := json.Marshal(with)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"permission":"admin"`)

	// Without permission — should omit the field entirely (omitempty).
	empty := types.CollaboratorAddRequest{}
	data2, err := json.Marshal(empty)
	require.NoError(t, err)
	assert.NotContains(t, string(data2), "permission",
		"empty permission should be omitted from wire format")
}

// TestCollaboratorRemoveEndpoint verifies the DELETE method + path
// shape for removing a collaborator.
func TestCollaboratorRemoveEndpoint(t *testing.T) {
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
	err := client.Delete(t.Context(), api.CollaboratorRemove("owner", "repo", "alice"))
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/repos/owner/repo/collaborators/alice", gotPath)
}

// TestResolveCollaboratorUsername validates the helper that picks the
// username from either the positional arg or the --username flag, and
// rejects ambiguous inputs.
func TestResolveCollaboratorUsername(t *testing.T) {
	// Reset shared flag before each sub-case (state is shared between
	// subtests; this is acceptable because the function only reads it).
	reset := func() { collaboratorFlags.username = "" }

	cases := []struct {
		name      string
		flagVal   string
		args      []string
		wantUser  string
		wantErr   bool
	}{
		{"positional arg only", "", []string{"alice"}, "alice", false},
		{"flag only", "bob", nil, "bob", false},
		{"flag only, no args", "carol", nil, "carol", false},
		{"both set, agree", "alice", []string{"alice"}, "", true},
		{"both set, disagree", "alice", []string{"bob"}, "", true},
		{"neither", "", nil, "", true},
		{"empty positional", "", []string{""}, "", true},
		{"empty flag, no args", "", nil, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reset()
			collaboratorFlags.username = c.flagVal
			got, err := resolveCollaboratorUsername(c.args)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error, got user=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.wantUser {
				t.Errorf("resolveCollaboratorUsername = %q, want %q", got, c.wantUser)
			}
		})
	}
}