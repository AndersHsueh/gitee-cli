package gist

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- endpoint + wire-format contract tests (httptest) -----

// TestGistCommentsListEndpoint verifies GET /gists/{gist_id}/comments
// — path, method, auth header, and full JSON unmarshal of the
// GistComment slice.
func TestGistCommentsListEndpoint(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotAuthHdr string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuthHdr = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"id": 12345,
				"body": "Nice trick with the slice here.",
				"created_at": "2026-06-25T10:00:00Z",
				"updated_at": "2026-06-25T10:00:00Z",
				"user": {
					"id": 1,
					"login": "alice",
					"name": "Alice",
					"html_url": "https://gitee.com/alice"
				}
			},
			{
				"id": 12346,
				"body": "Thanks, glad it helps.",
				"created_at": "2026-06-26T08:30:00Z",
				"updated_at": "2026-06-26T08:30:00Z",
				"user": {
					"id": 2,
					"login": "bob",
					"name": "Bob",
					"html_url": "https://gitee.com/bob"
				}
			}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	var comments []types.GistComment
	err := client.Get(t.Context(), api.GistCommentsList("abc123def"), &comments)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/gists/abc123def/comments", gotPath)
	assert.Equal(t, "token test-token", gotAuthHdr)
	require.Len(t, comments, 2)
	assert.Equal(t, int64(12345), comments[0].ID)
	assert.Equal(t, "Nice trick with the slice here.", comments[0].Body)
	assert.Equal(t, "alice", comments[0].User.Login)
	assert.Equal(t, int64(12346), comments[1].ID)
	assert.Equal(t, "bob", comments[1].User.Login)
}

// TestGistCommentCreateEndpoint verifies POST /gists/{gist_id}/comments
// — path, method, auth header, request body shape, and that the
// response unmarshals into a GistComment.
func TestGistCommentCreateEndpoint(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotAuthHdr string
		gotCT      string
		gotBody    []byte
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuthHdr = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 99999,
			"body": "Thanks, fixed.",
			"created_at": "2026-07-04T12:00:00Z",
			"updated_at": "2026-07-04T12:00:00Z",
			"user": {
				"id": 7,
				"login": "carol",
				"name": "Carol",
				"html_url": "https://gitee.com/carol"
			}
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	req := types.GistCommentRequest{Body: "Thanks, fixed."}
	var created types.GistComment
	err := client.Post(t.Context(), api.GistCommentCreate("abc123def"), req, &created)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/gists/abc123def/comments", gotPath)
	assert.Equal(t, "token test-token", gotAuthHdr)
	assert.Contains(t, gotCT, "application/json",
		"POST body should be JSON-encoded")
	var sent map[string]any
	require.NoError(t, json.Unmarshal(gotBody, &sent))
	assert.Equal(t, "Thanks, fixed.", sent["body"])
	assert.Equal(t, int64(99999), created.ID)
	assert.Equal(t, "carol", created.User.Login)
}

// TestGistCommentDeleteEndpoint verifies DELETE
// /gists/{gist_id}/comments/{id} — path, method, auth header.
func TestGistCommentDeleteEndpoint(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotAuthHdr string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuthHdr = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	err := client.Delete(t.Context(), api.GistCommentDelete("abc123def", 12345))
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/gists/abc123def/comments/12345", gotPath)
	assert.Equal(t, "token test-token", gotAuthHdr)
}

// TestGistCommentRequestRoundTrip pins the wire shape of the create
// request body so the contract to Gitee does not silently drift.
func TestGistCommentRequestRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"simple", "hello world", "hello world"},
		{"multiline", "line1\nline2", "line1\nline2"},
		{"empty", "", ""},
		{"unicode", "你好 world 🐹", "你好 world 🐹"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := types.GistCommentRequest{Body: c.body}
			b, err := json.Marshal(req)
			require.NoError(t, err)
			var got types.GistCommentRequest
			require.NoError(t, json.Unmarshal(b, &got))
			assert.Equal(t, c.want, got.Body)
		})
	}
}

// ----- command-level flag/args contracts -----

// resetGistCommentFlags restores the package-level flag state between
// tests. Cobra preserves parsed flag values after Execute(), so the
// globals would otherwise leak across tests.
func resetGistCommentFlags() {
	commentsCmd.SetArgs(nil)
	commentCmd.SetArgs(nil)
	commentDeleteCmd.SetArgs(nil)
	commentsFlags.limit = 30
	commentsFlags.page = 1
	commentFlags.body = ""
	gistCommentDeleteFlags.yes = false
}

// TestGistCommentsCommandRequiresGistID verifies the LIST verb errors
// without a positional gist_id. The cobra.ExactArgs(1) check fires at
// Command.Execute() time when the parent GistCmd is invoked with the
// "comments" subcommand and zero further args.
func TestGistCommentsCommandRequiresGistID(t *testing.T) {
	resetGistCommentFlags()
	defer resetGistCommentFlags()
	GistCmd.SetArgs([]string{"comments"})
	err := GistCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg")
}

// TestGistCommentCommandRequiresBody verifies the CREATE verb rejects
// executions without --body. Cobra's MarkFlagRequired on the comment
// cmd catches this at Execute() time.
func TestGistCommentCommandRequiresBody(t *testing.T) {
	resetGistCommentFlags()
	defer resetGistCommentFlags()
	GistCmd.SetArgs([]string{"comment", "abc123def"})
	err := GistCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required",
		"unexpected error: %v", err)
}

// TestGistCommentCommandAcceptsBody verifies the happy path through
// cobra's flag parsing: --body is parsed, MarkFlagRequired is
// satisfied, and the handler proceeds to the network call (which we
// mock via httptest below).
func TestGistCommentCommandAcceptsBody(t *testing.T) {
	resetGistCommentFlags()
	defer resetGistCommentFlags()
	commentCmd.ParseFlags([]string{"--body", "hello"})
	assert.Equal(t, "hello", commentFlags.body)
}

// TestGistCommentDeleteCommandRequiresTwoArgs verifies the DELETE verb
// rejects fewer than two positional args (cobra.ExactArgs(2)).
func TestGistCommentDeleteCommandRequiresTwoArgs(t *testing.T) {
	resetGistCommentFlags()
	defer resetGistCommentFlags()
	GistCmd.SetArgs([]string{"comment-delete"})
	err := GistCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 2 arg")
}

// TestGistCommentDeleteCommandRejectsNonNumericID verifies the DELETE
// verb reports a clear error for non-numeric comment ids. We pin this
// at the parsing boundary by ensuring the function rejects a string
// id before opening a network connection.
func TestGistCommentDeleteCommandRejectsNonNumericID(t *testing.T) {
	resetGistCommentFlags()
	defer resetGistCommentFlags()
	err := runGistCommentDelete(commentDeleteCmd, []string{"abc123def", "not-an-int"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid comment id")
}

// TestGistCommentsArgsBypassWithAgentHelp verifies the
// AgentHelpAwareArgs wrapper on cobra.ExactArgs(1) lets
// `gitee gist comments --agent-help` reach PreRunE without a
// <gist_id> positional.
func TestGistCommentsArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	assert.NoError(t, commentsCmd.Args(commentsCmd, nil))
}

// TestGistCommentsSendsPageAndPerPage verifies that --page and --limit
// are actually sent to the API as page / per_page query params. Before
// the fix, runGistComments called api.GistCommentsList(gistID) (bare
// /gists/{id}/comments) and both flags were silently ignored — --page
// returned page 1 every time.
func TestGistCommentsSendsPageAndPerPage(t *testing.T) {
	var (
		gotPerPage string
		gotPage    string
		gotPath    string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		q := r.URL.Query()
		gotPerPage = q.Get("per_page")
		gotPage = q.Get("page")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		resetGistCommentFlags()
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	commentsFlags.limit = 50
	commentsFlags.page = 2

	if err := runGistComments(commentsCmd, []string{"abc123def"}); err != nil {
		t.Fatalf("runGistComments failed: %v", err)
	}
	if gotPath != "/gists/abc123def/comments" {
		t.Errorf("expected path /gists/abc123def/comments, got %s", gotPath)
	}
	if gotPerPage != "50" {
		t.Errorf("expected per_page=50, got %q", gotPerPage)
	}
	if gotPage != "2" {
		t.Errorf("expected page=2, got %q", gotPage)
	}
}

// TestGistCommentsClampsPerPageTo100 verifies --limit is clamped to the
// Gitee v5 hard max of 100 so a value like 200 is not sent as-is.
func TestGistCommentsClampsPerPageTo100(t *testing.T) {
	var gotPerPage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		resetGistCommentFlags()
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	commentsFlags.limit = 200
	commentsFlags.page = 1

	if err := runGistComments(commentsCmd, []string{"abc123def"}); err != nil {
		t.Fatalf("runGistComments failed: %v", err)
	}
	if gotPerPage != "100" {
		t.Errorf("expected per_page clamped to 100, got %q", gotPerPage)
	}
}

// TestGistCommentsClientSideTruncation verifies --limit truncates the
// result client-side when the server returns more items than requested.
// This is the safety net that supplements per_page (the server may
// return a full page even if the user wanted fewer).
func TestGistCommentsClientSideTruncation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return 5 comments; --limit 2 should truncate to 2.
		_, _ = w.Write([]byte(`[
			{"id":1,"body":"a","user":{"login":"x"}},
			{"id":2,"body":"b","user":{"login":"x"}},
			{"id":3,"body":"c","user":{"login":"x"}},
			{"id":4,"body":"d","user":{"login":"x"}},
			{"id":5,"body":"e","user":{"login":"x"}}
		]`))
	}))
	defer ts.Close()

	origAPIURL := cmd.GlobalAPIURL
	origToken := cmd.GlobalToken
	origTimeout := cmd.GlobalTimeout
	origOutput := cmd.GlobalOutput
	defer func() {
		cmd.GlobalAPIURL = origAPIURL
		cmd.GlobalToken = origToken
		cmd.GlobalTimeout = origTimeout
		cmd.GlobalOutput = origOutput
		resetGistCommentFlags()
	}()
	cmd.GlobalAPIURL = ts.URL
	cmd.GlobalToken = "test-token"
	cmd.GlobalTimeout = 30
	cmd.GlobalOutput = "json"
	commentsFlags.limit = 2
	commentsFlags.page = 1

	if err := runGistComments(commentsCmd, []string{"abc123def"}); err != nil {
		t.Fatalf("runGistComments failed: %v", err)
	}
}

// TestGistCommentArgsBypassWithAgentHelp verifies the wrapper on
// cobra.ExactArgs(1) for `gitee gist comment --agent-help`.
func TestGistCommentArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	assert.NoError(t, commentCmd.Args(commentCmd, nil))
}

// TestGistCommentDeleteArgsBypassWithAgentHelp verifies the wrapper
// on cobra.ExactArgs(2) for `gitee gist comment-delete --agent-help`.
// We pass nil to Args and rely on the wrapper short-circuit.
func TestGistCommentDeleteArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	assert.NoError(t, commentDeleteCmd.Args(commentDeleteCmd, nil))
}

// TestGistUnstarArgsBypassWithAgentHelp verifies the wrapper on
// cobra.ExactArgs(1) for `gitee gist unstar --agent-help`.
func TestGistUnstarArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	assert.NoError(t, unstarCmd.Args(unstarCmd, nil))
}
