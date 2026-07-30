package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 404, Message: "not found"}
	assert.Contains(t, e.Error(), "404")
	assert.Contains(t, e.Error(), "not found")
}

func TestAPIError_Is(t *testing.T) {
	e404 := &APIError{StatusCode: 404}
	e401 := &APIError{StatusCode: 401}

	assert.True(t, e404.Is(&APIError{StatusCode: 404}))
	assert.False(t, e404.Is(&APIError{StatusCode: 401}))
	assert.False(t, e404.Is(e401))
}

func TestIsNotFound(t *testing.T) {
	assert.False(t, IsNotFound(nil))
	assert.False(t, IsNotFound(errors.New("random error")))

	api404 := &APIError{StatusCode: 404, Message: "not found"}
	assert.True(t, IsNotFound(api404))

	api401 := &APIError{StatusCode: 401}
	assert.False(t, IsNotFound(api401))

	wrapped := errors.New("failed: " + api404.Error())
	assert.False(t, IsNotFound(wrapped))
}

func TestIsUnauthorized(t *testing.T) {
	assert.False(t, IsUnauthorized(nil))
	assert.True(t, IsUnauthorized(&APIError{StatusCode: 401}))
	assert.False(t, IsUnauthorized(&APIError{StatusCode: 403}))
}

func TestIsForbidden(t *testing.T) {
	assert.False(t, IsForbidden(nil))
	assert.True(t, IsForbidden(&APIError{StatusCode: 403}))
	assert.False(t, IsForbidden(&APIError{StatusCode: 401}))
}

func TestIsUnprocessable(t *testing.T) {
	assert.False(t, IsUnprocessable(nil))
	assert.False(t, IsUnprocessable(errors.New("random error")))

	assert.True(t, IsUnprocessable(&APIError{StatusCode: 422, Message: "validation failed"}))
	assert.False(t, IsUnprocessable(&APIError{StatusCode: 404}))
	assert.False(t, IsUnprocessable(&APIError{StatusCode: 500}))

	// A wrapped 422 must still be detected (errors.As walks the chain),
	// and a non-422 error whose message happens to contain "422" (e.g.
	// a commit SHA) must NOT be mistaken for a 422 — the whole point of
	// using a typed check instead of strings.Contains.
	wrapped422 := fmt.Errorf("syncing ref 422abc123: %w", &APIError{StatusCode: 422})
	assert.True(t, IsUnprocessable(wrapped422))

	shaLike := fmt.Errorf("ref 422deadbeef not found: %w", &APIError{StatusCode: 404})
	assert.False(t, IsUnprocessable(shaLike))
}

func TestIsProjectOrEnterpriseNotFound(t *testing.T) {
	assert.False(t, IsProjectOrEnterpriseNotFound(nil))
	assert.False(t, IsProjectOrEnterpriseNotFound(errors.New("random error")))

	// Canonical 404 with the project-or-enterprise signature.
	assert.True(t, IsProjectOrEnterpriseNotFound(&APIError{
		StatusCode: 404,
		Message:    "404 Not Found: project or enterprise not found",
	}))

	// Case-insensitive: Gitee wording may shift case.
	assert.True(t, IsProjectOrEnterpriseNotFound(&APIError{
		StatusCode: 404,
		Message:    "Project Or Enterprise",
	}))

	// 404 without the signature must NOT trigger.
	assert.False(t, IsProjectOrEnterpriseNotFound(&APIError{
		StatusCode: 404,
		Message:    "issue not found",
	}))

	// Non-404 with the signature in the message must NOT trigger.
	assert.False(t, IsProjectOrEnterpriseNotFound(&APIError{
		StatusCode: 500,
		Message:    "project or enterprise",
	}))

	// Wrapped chain must be detected via errors.As.
	wrapped := fmt.Errorf("closing issue: %w", &APIError{
		StatusCode: 404,
		Message:    "project or enterprise not found",
	})
	assert.True(t, IsProjectOrEnterpriseNotFound(wrapped))

	// Outer text containing the signature while the inner APIError has
	// no signature (a generic 404 wrapped with an outer error that
	// happens to mention project) must NOT trigger — the predicate
	// inspects only the APIError.Message field.
	misleading := fmt.Errorf("project or enterprise failure: %w", &APIError{
		StatusCode: 404,
		Message:    "issue not found",
	})
	assert.False(t, IsProjectOrEnterpriseNotFound(misleading))
}

func TestCheckStatus_Success_200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	resp, err := resty.New().R().Get(ts.URL)
	require.NoError(t, err)
	assert.NoError(t, checkStatus(resp))
}

func TestCheckStatus_404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"message":"not found"}`))
	}))
	defer ts.Close()

	resp, err := resty.New().R().Get(ts.URL)
	require.NoError(t, err)

	err = checkStatus(resp)
	require.Error(t, err)

	var apiErr *APIError
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 404, apiErr.StatusCode)
}

func TestCheckStatus_401(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer ts.Close()

	resp, err := resty.New().R().Get(ts.URL)
	require.NoError(t, err)

	err = checkStatus(resp)
	require.Error(t, err)

	var apiErr *APIError
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestCheckStatus_500(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"message":"boom"}`))
	}))
	defer ts.Close()

	resp, err := resty.New().R().Get(ts.URL)
	require.NoError(t, err)

	err = checkStatus(resp)
	require.Error(t, err)

	var apiErr *APIError
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 500, apiErr.StatusCode)
	assert.Equal(t, "boom", apiErr.Message)
}

func TestHTTPStatusMessage(t *testing.T) {
	assert.Equal(t, "bad request", httpStatusMessage(400))
	assert.Contains(t, httpStatusMessage(401), "gitee auth login")
	assert.Equal(t, "permission denied", httpStatusMessage(403))
	assert.Contains(t, httpStatusMessage(404), "does not exist")
	assert.Equal(t, "validation failed", httpStatusMessage(422))
	assert.Equal(t, "internal server error", httpStatusMessage(500))
	assert.Equal(t, "service unavailable", httpStatusMessage(503))
	assert.Contains(t, httpStatusMessage(418), "unexpected status")
}
