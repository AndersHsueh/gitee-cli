package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"gitee-cli/pkg/types"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	ts := httptest.NewServer(handler)
	client := NewClient("test-token", ts.URL, false, 30)
	return client, ts
}

func TestClient_GetRepos(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/user/repos", r.URL.Path)
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		repos := []types.Repository{{ID: 1, Name: "test-repo", FullName: "owner/test-repo"}}
		json.NewEncoder(w).Encode(repos)
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	var repos []types.Repository
	err := client.Get(context.Background(), "/user/repos", &repos)

	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "test-repo", repos[0].Name)
	assert.Equal(t, "owner/test-repo", repos[0].FullName)
}

func TestClient_PostCreateRepo(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/user/repos", r.URL.Path)
		assert.Contains(t, r.Header.Get("Content-Type"), "application/json")

		body, _ := io.ReadAll(r.Body)
		var req types.RepositoryCreateRequest
		json.Unmarshal(body, &req)
		assert.Equal(t, "test-repo", req.Name)
		assert.Equal(t, "A test repo", req.Description)

		w.Header().Set("Content-Type", "application/json")
		repo := types.Repository{ID: 1, Name: req.Name, FullName: "owner/" + req.Name, Description: req.Description}
		json.NewEncoder(w).Encode(repo)
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	req := types.RepositoryCreateRequest{Name: "test-repo", Description: "A test repo"}
	var repo types.Repository
	err := client.Post(context.Background(), "/user/repos", req, &repo)

	require.NoError(t, err)
	assert.Equal(t, "test-repo", repo.Name)
	assert.Equal(t, "A test repo", repo.Description)
}

func TestClient_PatchEditRepo(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Contains(t, r.URL.Path, "/repos/owner/test-repo")

		body, _ := io.ReadAll(r.Body)
		var req types.RepositoryEditRequest
		json.Unmarshal(body, &req)
		assert.Equal(t, "Updated description", req.Description)

		w.Header().Set("Content-Type", "application/json")
		repo := types.Repository{ID: 1, Name: "test-repo", Description: req.Description}
		json.NewEncoder(w).Encode(repo)
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	req := types.RepositoryEditRequest{Description: "Updated description"}
	var repo types.Repository
	err := client.Patch(context.Background(), "/repos/owner/test-repo", req, &repo)

	require.NoError(t, err)
	assert.Equal(t, "Updated description", repo.Description)
}

func TestClient_Delete(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "/repos/owner/test-repo")
		w.WriteHeader(http.StatusNoContent)
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	err := client.Delete(context.Background(), "/repos/owner/test-repo")

	require.NoError(t, err)
}

func TestClient_Put(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.URL.Path, "/gists/abc123/star")
		w.WriteHeader(http.StatusNoContent)
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	err := client.Put(context.Background(), "/gists/abc123/star", nil, nil)

	require.NoError(t, err)
}

func TestClient_PostNotRetriedOn5xx(t *testing.T) {
	// A POST that receives a 5xx must NOT be retried — otherwise the
	// server may have already applied the write and we'd create a
	// duplicate resource. We count how many times the handler fires;
	// it must be exactly 1.
	var calls int
	handler := func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"internal server error"}`))
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	client := NewClient("test-token", ts.URL, false, 30)
	var result map[string]interface{}
	err := client.Post(context.Background(), "/user/repos", map[string]string{"name": "test"}, &result)

	require.Error(t, err, "POST 500 should return an error")
	assert.Equal(t, 1, calls, "POST should not be retried on 5xx")
}

func TestClient_GetRetriedOn5xx(t *testing.T) {
	// A GET that receives a 5xx SHOULD be retried. We make the first
	// call fail with 500 and the second succeed, then assert 2 total
	// calls (1 initial + 1 retry). Keeping it to one retry keeps the
	// test fast (resty's first retry wait is ~2s).
	var calls int
	handler := func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"internal server error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]types.Repository{{ID: 1, Name: "test-repo"}})
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	client := NewClient("test-token", ts.URL, false, 30)
	var repos []types.Repository
	err := client.Get(context.Background(), "/user/repos", &repos)

	require.NoError(t, err, "GET should succeed after a retry")
	assert.Equal(t, 2, calls, "GET 500 should be retried at least once")
}

func TestClient_GetRaw(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.Header().Set("X-Total-Count", "100")
		w.Write([]byte("raw content here"))
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	body, err := client.GetRaw(context.Background(), "/some/path")

	require.NoError(t, err)
	assert.Equal(t, "raw content here", string(body))
}

func TestCheckStatus_Success(t *testing.T) {
	tests := []struct {
		statusCode int
	}{
		{200}, {201}, {204},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", "http://test", nil)
		resp := &resty.Response{
			RawResponse: &http.Response{
				StatusCode: tt.statusCode,
				Status:     http.StatusText(tt.statusCode),
				Request:    req,
				Body:       io.NopCloser(nil),
			},
		}

		err := checkStatus(resp)
		require.NoError(t, err, "statusCode %d should not return error", tt.statusCode)
	}
}

func TestCheckStatus_Error(t *testing.T) {
	tests := []struct {
		statusCode int
	}{
		{400}, {401}, {403}, {404}, {422}, {500},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", "http://test", nil)
		resp := &resty.Response{
			RawResponse: &http.Response{
				StatusCode: tt.statusCode,
				Status:     http.StatusText(tt.statusCode),
				Request:    req,
				Body:       io.NopCloser(nil),
			},
		}

		err := checkStatus(resp)
		require.Error(t, err, "statusCode %d should return error", tt.statusCode)
	}
}

// TestClient_PatchForm locks in the wire shape that Gitee v5 requires for
// the issue update endpoint. The bug that prompted this test was:
//
//	PATCH /repos/{owner}/{repo}/issues/{number}
//	Content-Type: application/json
//	{"state":"closed"}            <-- rejected with 400 / 404
//
// The correct request shape is:
//
//	PATCH /repos/{owner}/issues/{number}         (no {repo} segment)
//	Content-Type: application/x-www-form-urlencoded
//	repo=<repo>&state=closed                     (formData, with `repo` field)
//
// Without this, the close / reopen / edit commands all 400. The regression
// test asserts every dimension — path, method, content-type, and form
// fields — so a future refactor can't silently break the contract.
func TestClient_PatchForm(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		// Path drops {repo} — the Gitee v5 write quirk.
		assert.Equal(t, "/repos/andershsueh/issues/IK5FZX", r.URL.Path,
			"issue update path must drop {repo}; use formData for `repo`")
		// Content-Type MUST be form-encoded, NOT JSON.
		ct := r.Header.Get("Content-Type")
		assert.Contains(t, ct, "application/x-www-form-urlencoded",
			"Gitee v5 update-issue endpoint requires formData, not JSON body")

		// Verify formData was parsed and carries both `repo` and `state`.
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		assert.Equal(t, "gitee-cli", r.FormValue("repo"),
			"`repo` must be sent as a form field on the issue update endpoint")
		assert.Equal(t, "closed", r.FormValue("state"),
			"`state` must be sent as a form field for the close verb")

		w.Header().Set("Content-Type", "application/json")
		// Mimic the real Gitee response shape for a closed issue.
		w.Write([]byte(`{"number":"IK5FZX","state":"closed","title":"x"}`))
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	form := url.Values{}
	form.Set("repo", "gitee-cli")
	form.Set("state", "closed")

	var issue types.Issue
	err := client.PatchForm(context.Background(), "/repos/andershsueh/issues/IK5FZX", form, &issue)
	require.NoError(t, err)
	assert.Equal(t, "IK5FZX", issue.Number)
	assert.Equal(t, "closed", issue.State)
}

// TestClient_PatchForm_EmptyForm guards the "PATCH with no body" path.
// resty.SetFormData on a nil map used to panic on some versions; we
// explicitly allow nil/empty url.Values to be a no-op body. The
// endpoint that uses this is rare (issue update with no fields set),
// so we just confirm the call returns 2xx against a handler that
// expects no body.
func TestClient_PatchForm_EmptyForm(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		// No body parsing, no fields expected — just confirm verb + 200.
		w.WriteHeader(http.StatusOK)
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	err := client.PatchForm(context.Background(), "/repos/owner/issues/1", nil, nil)
	require.NoError(t, err)
}

// TestClient_PatchForm_400Error mirrors the bug that triggered this whole
// refactor: if the server returns 400, the client must surface a
// structured APIError so the cmd layer can render it. Without this test,
// a future resty version that swallows 4xx would silently break the
// user-facing error message.
func TestClient_PatchForm_400Error(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"state does not have a valid value"}`))
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	form := url.Values{}
	form.Set("repo", "gitee-cli")
	form.Set("state", "close") // typo — server should reject

	err := client.PatchForm(context.Background(), "/repos/owner/issues/1", form, nil)
	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "state")
}
