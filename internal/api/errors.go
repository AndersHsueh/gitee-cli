package api

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-resty/resty/v2"
)

// APIError represents an error returned by the Gitee API.
type APIError struct {
	StatusCode int
	Message    string      `json:"message"`
	Errors     interface{} `json:"errors"`
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

// Unwrap 返回底层错误，支持错误链提取。
func (e *APIError) Unwrap() error {
	return fmt.Errorf("gitee api error: status=%d message=%s", e.StatusCode, e.Message)
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
