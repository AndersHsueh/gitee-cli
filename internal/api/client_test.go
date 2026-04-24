package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitee-cli/pkg/types"
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

func TestClient_GetRaw(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.Header().Set("X-Total-Count", "100")
		w.Write([]byte("raw content here"))
	}

	client, ts := newTestClient(t, handler)
	defer ts.Close()

	body, header, err := client.GetRaw(context.Background(), "/some/path")

	require.NoError(t, err)
	assert.Equal(t, "raw content here", string(body))
	assert.Equal(t, "100", header.Get("X-Total-Count"))
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
