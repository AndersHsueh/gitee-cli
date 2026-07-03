package repo

import (
	"encoding/base64"
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

func TestFileAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo file",
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
		if !strings.Contains(repoFileAgentHelp, want) {
			t.Errorf("umbrella file agent help missing required section: %q", want)
		}
	}
}

func TestFileViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo file view",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
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
		if !strings.Contains(repoFileViewAgentHelp, want) {
			t.Errorf("view verb agent help missing required section: %q", want)
		}
	}
}

func TestFilePushAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo file push",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
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
		if !strings.Contains(repoFilePushAgentHelp, want) {
			t.Errorf("push verb agent help missing required section: %q", want)
		}
	}
}

func TestFileDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo file delete",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
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
		if !strings.Contains(repoFileDeleteAgentHelp, want) {
			t.Errorf("delete verb agent help missing required section: %q", want)
		}
	}
}

// ----- flag coverage -----

// TestFileFlagsCoveredInAgentHelp ensures every flag exposed by the
// file subcommands is also documented in its agent help string.
func TestFileFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// view
		{"--repo", repoFileViewAgentHelp},
		{"-R", repoFileViewAgentHelp},

		// push
		{"--repo", repoFilePushAgentHelp},
		{"-R", repoFilePushAgentHelp},
		{"--message", repoFilePushAgentHelp},
		{"-m", repoFilePushAgentHelp},
		{"--branch", repoFilePushAgentHelp},
		{"-b", repoFilePushAgentHelp},

		// delete
		{"--repo", repoFileDeleteAgentHelp},
		{"-R", repoFileDeleteAgentHelp},
		{"--message", repoFileDeleteAgentHelp},
		{"-m", repoFileDeleteAgentHelp},
		{"--branch", repoFileDeleteAgentHelp},
		{"-b", repoFileDeleteAgentHelp},
		{"--sha", repoFileDeleteAgentHelp},
		{"--force", repoFileDeleteAgentHelp},
		{"-f", repoFileDeleteAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}

// ----- wire-format + endpoint contract tests (httptest) -----

// TestRepoContentsGetEndpoint verifies GET /repos/{owner}/{repo}/contents/{path}
// returns a single ContentEntry JSON object for files.
func TestRepoContentsGetEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		// base64 of "hello\n"
		_, _ = w.Write([]byte(`{
			"name":"README.md",
			"path":"README.md",
			"sha":"abc123",
			"size":6,
			"type":"file",
			"encoding":"base64",
			"content":"aGVsbG8K",
			"html_url":"https://gitee.com/o/r/blob/main/README.md",
			"download_url":"https://gitee.com/o/r/raw/main/README.md"
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	body, _, err := client.GetRaw(t.Context(), api.RepoContentsGet("o", "r", "README.md"))
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/repos/o/r/contents/README.md", gotPath)

	var entry types.ContentEntry
	require.NoError(t, json.Unmarshal(body, &entry))
	assert.Equal(t, "README.md", entry.Name)
	assert.Equal(t, "file", entry.Type)
	assert.Equal(t, "base64", entry.Encoding)
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(entry.Content))
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(decoded))
}

// TestRepoContentsListEndpoint verifies the same path returns a JSON
// array of ContentEntry when the path is a directory.
func TestRepoContentsListEndpoint(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"name":"a.md","path":"docs/a.md","type":"file","size":10},
			{"name":"b.md","path":"docs/b.md","type":"file","size":20}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	body, _, err := client.GetRaw(t.Context(), api.RepoContentsGet("o", "r", "docs"))
	require.NoError(t, err)
	assert.Equal(t, "/repos/o/r/contents/docs", gotPath)

	var entries []types.ContentEntry
	require.NoError(t, json.Unmarshal(body, &entries))
	require.Len(t, entries, 2)
	assert.Equal(t, "a.md", entries[0].Name)
	assert.Equal(t, "file", entries[0].Type)
}

// TestRepoContentsCreateEndpoint verifies the POST body shape (no sha)
// for creating a file.
func TestRepoContentsCreateEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   types.ContentCreateRequest
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"content":{"name":"x.md","path":"x.md","sha":"new-sha","type":"file"},
			"commit":{"sha":"commitsha","message":"create x"}
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	req := types.ContentCreateRequest{
		Content: base64.StdEncoding.EncodeToString([]byte("# x\n")),
		Message: "create x",
	}
	var resp types.ContentResponse
	err := client.Post(t.Context(), api.RepoContentsCreate("o", "r", "x.md"), req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/repos/o/r/contents/x.md", gotPath)
	assert.Equal(t, "create x", gotBody.Message)
	assert.Equal(t, "new-sha", resp.Content.SHA)
	assert.Equal(t, "commitsha", resp.Commit.SHA)
}

// TestRepoContentsUpdateEndpoint verifies the PUT body shape (with sha)
// for updating a file.
func TestRepoContentsUpdateEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   types.ContentUpdateRequest
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"content":{"name":"x.md","path":"x.md","sha":"updated-sha","type":"file"},
			"commit":{"sha":"commitsha2","message":"update x"}
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	req := types.ContentUpdateRequest{
		Content: base64.StdEncoding.EncodeToString([]byte("# x updated\n")),
		Message: "update x",
		SHA:     "old-sha",
	}
	var resp types.ContentResponse
	err := client.Put(t.Context(), api.RepoContentsUpdate("o", "r", "x.md"), req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/repos/o/r/contents/x.md", gotPath)
	assert.Equal(t, "old-sha", gotBody.SHA)
	assert.Equal(t, "update x", gotBody.Message)
	assert.Equal(t, "updated-sha", resp.Content.SHA)
}

// TestRepoContentsDeleteEndpoint verifies that DeleteWithBody sends a
// DELETE with the JSON body containing sha + message.
func TestRepoContentsDeleteEndpoint(t *testing.T) {
	var (
		gotMethod      string
		gotPath        string
		gotContentType string
		gotBody        types.ContentDeleteRequest
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	req := types.ContentDeleteRequest{
		SHA:     "old-sha",
		Message: "delete x",
	}
	err := client.DeleteWithBody(t.Context(), api.RepoContentsDelete("o", "r", "x.md"), req)
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/repos/o/r/contents/x.md", gotPath)
	assert.Contains(t, gotContentType, "application/json",
		"DELETE-with-body must carry a JSON content-type so the server parses the body")
	assert.Equal(t, "old-sha", gotBody.SHA)
	assert.Equal(t, "delete x", gotBody.Message)
}

// TestContentCreateRequestWireFormat verifies the JSON wire format of
// the create payload: required `content` and `message` plus optional
// branch/author fields, with omitempty on the optional ones.
func TestContentCreateRequestWireFormat(t *testing.T) {
	full := types.ContentCreateRequest{
		Content: "Y29udGVudA==",
		Message: "msg",
		Branch:  "feat/x",
		Author:  "Alice",
	}
	data, err := json.Marshal(full)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"content":"Y29udGVudA=="`)
	assert.Contains(t, string(data), `"message":"msg"`)
	assert.Contains(t, string(data), `"branch":"feat/x"`)
	assert.Contains(t, string(data), `"author":"Alice"`)

	minimal := types.ContentCreateRequest{Content: "YQ==", Message: "m"}
	data2, err := json.Marshal(minimal)
	require.NoError(t, err)
	assert.NotContains(t, string(data2), "branch",
		"empty branch should be omitted from wire format")
	assert.NotContains(t, string(data2), "author",
		"empty author should be omitted from wire format")
}

// TestIsNotFoundRecognizes404 verifies that api.IsNotFound matches the
// 404 *api.APIError so probeFileSHA can disambiguate "no such file"
// from "transport failure".
func TestIsNotFoundRecognizes404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"404 Not Found","status":404}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	_, _, err := client.GetRaw(t.Context(), api.RepoContentsGet("o", "r", "missing.md"))
	require.Error(t, err)
	assert.True(t, api.IsNotFound(err), "api.IsNotFound must recognize a 404 as a not-found error")
}
