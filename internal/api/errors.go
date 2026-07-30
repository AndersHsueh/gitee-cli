package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

// projectOrEnterpriseSignature is the lowercase substring Gitee v5 uses
// in the message of PATCH /repos/{owner}/{repo}/issues/{number} 404
// responses when the issue lives in an enterprise/project container that
// the public API cannot route to. Exported as the matching happens in
// two layers (predicate + agent-help docs).
const projectOrEnterpriseSignature = "project or enterprise"

// APIError represents an error returned by the Gitee API.
type APIError struct {
	StatusCode int
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// Is 支持按状态码匹配 API 错误，便于 errors.Is 调用。
func (e *APIError) Is(target error) bool {
	var t *APIError
	if errors.As(target, &t) {
		return e.StatusCode == t.StatusCode
	}
	return false
}

func checkStatus(resp *resty.Response) error {
	code := resp.StatusCode()
	if code >= 200 && code < 300 {
		return nil
	}

	var apiErr APIError
	apiErr.StatusCode = code
	_ = json.Unmarshal(resp.Body(), &apiErr)
	if apiErr.Message == "" {
		apiErr.Message = httpStatusMessage(code)
	}

	return &apiErr
}

func httpStatusMessage(code int) string {
	switch code {
	case 400:
		return "bad request"
	case 401:
		return "authentication failed: run 'gitee auth login' or set GITEE_TOKEN"
	case 403:
		return "permission denied"
	case 404:
		return "resource does not exist or you lack read access"
	case 422:
		return "validation failed"
	case 500:
		return "internal server error"
	case 503:
		return "service unavailable"
	default:
		return fmt.Sprintf("unexpected status %d", code)
	}
}

// IsNotFound 通过 errors.As 检查 StatusCode 而非字符串匹配。
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}
	return false
}

// IsUnauthorized reports whether err indicates an HTTP 401 response.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401
	}
	return false
}

// IsForbidden reports whether err indicates an HTTP 403 response.
func IsForbidden(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 403
	}
	return false
}

// IsUnprocessable reports whether err indicates an HTTP 422 response
// (validation failed / unprocessable entity). Use this instead of
// strings.Contains(err.Error(), "422"), which would false-positive on
// any error message that happens to contain the substring "422"
// (e.g. a commit SHA or URL).
func IsUnprocessable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 422
	}
	return false
}

// IsProjectOrEnterpriseNotFound reports whether err is a Gitee v5 404
// carrying the "project or enterprise" routing message. This is the
// signature of the issue state-transition regression: when an issue
// lives in an enterprise/project container, PATCH on
// /repos/{owner}/{repo}/issues/{number} cannot find the container and
// returns 404 with a body containing the signature string. Callers can
// use this to surface a Web-UI fallback URL in the error message.
//
// We match on the APIError.Message field via errors.As rather than
// strings.Contains(err.Error(), ...) so that outer wrapping text
// (e.g. "closing issue: HTTP 404: project or enterprise") does not
// re-trigger the predicate and so we don't allocate a lowercased
// copy of the full Error() string on every check.
func IsProjectOrEnterpriseNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode != 404 {
		return false
	}
	return strings.Contains(strings.ToLower(apiErr.Message), projectOrEnterpriseSignature)
}
