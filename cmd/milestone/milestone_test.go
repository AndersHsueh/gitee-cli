package milestone

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- agent help content (per-verb) -----

func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee milestone list",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
		// race section names the concrete interleavings
		"point-in-time snapshot",
		"indexing lag",
	}
	for _, want := range required {
		if !strings.Contains(milestoneListAgentHelp, want) {
			t.Errorf("list agent help missing required section: %q", want)
		}
	}
}

func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee milestone view",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
		"point-in-time snapshot",
	}
	for _, want := range required {
		if !strings.Contains(milestoneViewAgentHelp, want) {
			t.Errorf("view agent help missing required section: %q", want)
		}
	}
}

func TestCreateAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee milestone create",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
		"does NOT",
		"deduplicate by title",
	}
	for _, want := range required {
		if !strings.Contains(milestoneCreateAgentHelp, want) {
			t.Errorf("create agent help missing required section: %q", want)
		}
	}
}

func TestEditAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee milestone edit",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
		"last-write-wins",
	}
	for _, want := range required {
		if !strings.Contains(milestoneEditAgentHelp, want) {
			t.Errorf("edit agent help missing required section: %q", want)
		}
	}
}

func TestDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee milestone delete",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
		"silently detached",
	}
	for _, want := range required {
		if !strings.Contains(milestoneDeleteAgentHelp, want) {
			t.Errorf("delete agent help missing required section: %q", want)
		}
	}
}

// ----- wire-format + endpoint contract tests (httptest) -----

// TestMilestonesListEndpoint verifies GET /repos/{owner}/{repo}/milestones
// is hit and parses into []types.Milestone cleanly.
func TestMilestonesListEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1,"number":1,"title":"v1.0","state":"open","open_issues":3,"closed_issues":1},
			{"id":2,"number":2,"title":"v1.1","state":"closed","open_issues":0,"closed_issues":7}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	var ms []types.Milestone
	err := client.Get(t.Context(), api.MilestonesList("owner", "repo"), &ms)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/repos/owner/repo/milestones", gotPath)
	require.Len(t, ms, 2)
	assert.Equal(t, 1, ms[0].Number)
	assert.Equal(t, "v1.0", ms[0].Title)
	assert.Equal(t, "open", ms[0].State)
	assert.Equal(t, 3, ms[0].OpenIssues)
	assert.Equal(t, 2, ms[1].Number)
	assert.Equal(t, "closed", ms[1].State)
}

// TestMilestoneGetEndpoint verifies GET /repos/{owner}/{repo}/milestones/{number}.
func TestMilestoneGetEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":1,"number":1,"title":"v1.0","description":"first major","state":"open",
			"open_issues":3,"closed_issues":1,
			"due_on":"2026-09-30T00:00:00+08:00",
			"html_url":"https://gitee.com/owner/repo/milestones/1"
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	var m types.Milestone
	err := client.Get(t.Context(), api.MilestoneGet("owner", "repo", 1), &m)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/repos/owner/repo/milestones/1", gotPath)
	assert.Equal(t, 1, m.Number)
	assert.Equal(t, "v1.0", m.Title)
	assert.Equal(t, "first major", m.Description)
	require.NotNil(t, m.DueOn)
	// just sanity-check that we parsed a real timestamp
	assert.Equal(t, 2026, m.DueOn.Year())
}

// TestMilestoneCreateEndpoint verifies POST body shape and response decode.
func TestMilestoneCreateEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   types.MilestoneCreateRequest
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":7,"number":7,"title":"v2.0","state":"open","open_issues":0,"closed_issues":0
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	req := types.MilestoneCreateRequest{
		Title:       "v2.0",
		Description: "Major rewrite",
		DueOn:       "2027-01-31",
	}
	var created types.Milestone
	err := client.Post(t.Context(), api.MilestoneCreate("owner", "repo"), req, &created)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/repos/owner/repo/milestones", gotPath)
	assert.Equal(t, "v2.0", gotBody.Title)
	assert.Equal(t, "Major rewrite", gotBody.Description)
	assert.Equal(t, "2027-01-31", gotBody.DueOn)
	assert.Equal(t, 7, created.Number)
}

// TestMilestoneUpdateEndpoint verifies PATCH body shape and response decode.
func TestMilestoneUpdateEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   types.MilestoneUpdateRequest
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":1,"number":1,"title":"v1.0 (revised)","state":"open","open_issues":3,"closed_issues":1
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	req := types.MilestoneUpdateRequest{Title: "v1.0 (revised)"}
	var updated types.Milestone
	err := client.Patch(t.Context(), api.MilestoneUpdate("owner", "repo", 1), req, &updated)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, "/repos/owner/repo/milestones/1", gotPath)
	assert.Equal(t, "v1.0 (revised)", gotBody.Title)
	assert.Equal(t, "v1.0 (revised)", updated.Title)
}

// TestMilestoneDeleteEndpoint verifies DELETE is called on the right path.
func TestMilestoneDeleteEndpoint(t *testing.T) {
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
	err := client.Delete(t.Context(), api.MilestoneDelete("owner", "repo", 5))
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/repos/owner/repo/milestones/5", gotPath)
}

// TestMilestoneTypeRoundTrip verifies the wire format of the request bodies
// matches what the Gitee API expects.
func TestMilestoneTypeRoundTrip(t *testing.T) {
	// Create request — required title, optional everything else.
	create := types.MilestoneCreateRequest{Title: "v1.0"}
	data, err := json.Marshal(create)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"title":"v1.0"`)
	// Optional fields should be omitted when empty.
	assert.NotContains(t, string(data), "description")
	assert.NotContains(t, string(data), "due_on")
	assert.NotContains(t, string(data), `"state"`)

	// Update request — all optional.
	update := types.MilestoneUpdateRequest{State: "closed"}
	data2, err := json.Marshal(update)
	require.NoError(t, err)
	assert.Contains(t, string(data2), `"state":"closed"`)
	assert.NotContains(t, string(data2), "title")

	// Full Milestone unmarshal sanity-check (mirrors what Gitee returns).
	due := time.Date(2026, 9, 30, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	raw := `{"id":1,"number":1,"title":"v1.0","description":"d","state":"open","open_issues":3,"closed_issues":1,"due_on":"2026-09-30T00:00:00+08:00","html_url":"https://gitee.com/o/r/milestones/1","created_at":"2026-01-01T00:00:00+08:00","updated_at":"2026-02-01T00:00:00+08:00"}`
	var m types.Milestone
	require.NoError(t, json.Unmarshal([]byte(raw), &m))
	assert.Equal(t, int64(1), m.ID)
	assert.Equal(t, 1, m.Number)
	assert.Equal(t, "v1.0", m.Title)
	assert.Equal(t, "open", m.State)
	assert.Equal(t, 3, m.OpenIssues)
	assert.Equal(t, 1, m.ClosedIssues)
	require.NotNil(t, m.DueOn)
	assert.Equal(t, due.Year(), m.DueOn.Year())
	assert.Equal(t, "https://gitee.com/o/r/milestones/1", m.HTMLURL)
}

// TestMilestoneFlagsCoveredInAgentHelp ensures every flag exposed by the
// milestone subcommands is also documented in its agent help string.
func TestMilestoneFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// list
		{"--repo", milestoneListAgentHelp},
		{"-R", milestoneListAgentHelp},
		{"--limit", milestoneListAgentHelp},
		{"-L", milestoneListAgentHelp},
		{"--state", milestoneListAgentHelp},

		// view
		{"--number", milestoneViewAgentHelp},
		{"-n", milestoneViewAgentHelp},

		// create
		{"--title", milestoneCreateAgentHelp},
		{"-t", milestoneCreateAgentHelp},
		{"--description", milestoneCreateAgentHelp},
		{"-d", milestoneCreateAgentHelp},
		{"--due-on", milestoneCreateAgentHelp},

		// edit
		{"--title", milestoneEditAgentHelp},
		{"-t", milestoneEditAgentHelp},
		{"--due-on", milestoneEditAgentHelp},
		{"--state", milestoneEditAgentHelp},

		// delete
		{"--force", milestoneDeleteAgentHelp},
		{"-f", milestoneDeleteAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}