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

// TestTagAgentHelpContent verifies the umbrella tag help string covers
// the standard --agent-help sections.
func TestTagAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo tag",
		"## Purpose",
		"## Pre-conditions",
		"## Subcommands",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## OAuth scope required",
		"## Rate limit",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoTagAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

// TestTagListAgentHelpContent verifies the list verb's help string.
func TestTagListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo tag list",
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
		if !strings.Contains(repoTagListAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

// TestTagCreateAgentHelpContent verifies the create verb's help string.
func TestTagCreateAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo tag create",
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
		if !strings.Contains(repoTagCreateAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

// TestRepoTagFlagsCoveredInAgentHelp ensures every flag exposed by the
// tag subcommands is also documented in its agent help string.
func TestRepoTagFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// list
		{"--repo", repoTagListAgentHelp},
		{"-R", repoTagListAgentHelp},

		// create
		{"--repo", repoTagCreateAgentHelp},
		{"-R", repoTagCreateAgentHelp},
		{"--name", repoTagCreateAgentHelp},
		{"-n", repoTagCreateAgentHelp},
		{"--refs", repoTagCreateAgentHelp},
		{"-r", repoTagCreateAgentHelp},
		{"--message", repoTagCreateAgentHelp},
		{"-m", repoTagCreateAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}

// TestResolveRepoArg validates the helper that the tag subcommands use
// to turn the local --repo flag (or the global rt.Repo fallback)
// into an owner/repo pair.
func TestResolveRepoArg(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"valid", "owner/repo", "owner", "repo", false},
		{"dots in repo", "my.org/svc.api", "my.org", "svc.api", false},
		{"empty", "", "", "", true},
		{"no slash", "ownerrepo", "", "", true},
		{"trailing slash only", "owner/", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pair, err := resolveRepoArg(c.input)
			if c.wantErr {
				if err == nil {
					t.Errorf("resolveRepoArg(%q) expected error, got %v", c.input, pair)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveRepoArg(%q) unexpected error: %v", c.input, err)
			}
			if pair[0] != c.wantOwner || pair[1] != c.wantRepo {
				t.Errorf("resolveRepoArg(%q) = (%q, %q), want (%q, %q)",
					c.input, pair[0], pair[1], c.wantOwner, c.wantRepo)
			}
		})
	}
}

// TestTagCreateRequestJSON verifies the wire format of the create payload
// matches what the Gitee API expects (refs / tag_name / tag_message).
func TestTagCreateRequestJSON(t *testing.T) {
	req := types.TagCreateRequest{
		Refs:       "main",
		TagName:    "v1.0.0",
		TagMessage: "release notes",
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"refs":"main"`)
	assert.Contains(t, string(data), `"tag_name":"v1.0.0"`)
	assert.Contains(t, string(data), `"tag_message":"release notes"`)

	// tag_message is omitempty — verify it disappears when unset.
	empty := types.TagCreateRequest{Refs: "main", TagName: "v1"}
	data2, err := json.Marshal(empty)
	require.NoError(t, err)
	assert.NotContains(t, string(data2), "tag_message",
		"empty tag_message should be omitted from wire format")
}

// TestReposTagsListEndpoint is an httptest mock verifying that the
// GET /repos/{owner}/{repo}/tags endpoint is hit and its JSON payload
// parses cleanly into []types.Tag.
func TestReposTagsListEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name":"v1.0.0","message":"first","commit":{"sha":"abc1234567890","url":"https://gitee.com/o/r/commit/abc"}},
			{"name":"v1.1.0","message":"second","commit":{"sha":"deadbeef0000","url":"https://gitee.com/o/r/commit/dead"}}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	var tags []types.Tag
	err := client.Get(t.Context(), api.ReposTagsList("o", "r"), &tags)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/repos/o/r/tags", gotPath)
	require.Len(t, tags, 2)
	assert.Equal(t, "v1.0.0", tags[0].Name)
	assert.Equal(t, "abc1234567890", tags[0].Commit.SHA)
	assert.Equal(t, "v1.1.0", tags[1].Name)
}

// TestReposTagsCreateEndpoint verifies the POST body shape and the
// 201-style response decoding into types.Tag.
func TestReposTagsCreateEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   types.TagCreateRequest
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name":"v2.0.0",
			"message":"",
			"commit":{"sha":"1234567890abcdef","url":"https://gitee.com/o/r/commit/1234"}
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	req := types.TagCreateRequest{
		Refs:    "main",
		TagName: "v2.0.0",
	}
	var created types.Tag
	err := client.Post(t.Context(), api.ReposTagsCreate("o", "r"), req, &created)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/repos/o/r/tags", gotPath)
	assert.Equal(t, "main", gotBody.Refs)
	assert.Equal(t, "v2.0.0", gotBody.TagName)
	assert.Equal(t, "v2.0.0", created.Name)
	assert.Equal(t, "1234567890abcdef", created.Commit.SHA)
}