package api

import (
	"encoding/json"
	"fmt"
	"strings"

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

func checkStatus(resp *resty.Response) error {
	code := resp.StatusCode()
	if code >= 200 && code < 300 {
		return nil
	}

	// Try to parse Gitee error body
	var apiErr APIError
	apiErr.StatusCode = code
	_ = json.Unmarshal(resp.Body(), &apiErr)
	if apiErr.Message == "" {
		apiErr.Message = http_status_message(code)
	}

	switch code {
	case 401:
		return fmt.Errorf("authentication failed (HTTP 401). Run 'gitee auth login' or set GITEE_TOKEN")
	case 403:
		return fmt.Errorf("permission denied (HTTP 403): %s", apiErr.Message)
	case 404:
		return fmt.Errorf("not found (HTTP 404): resource does not exist or you lack read access")
	case 422:
		return fmt.Errorf("validation failed (HTTP 422): %s", apiErr.Message)
	default:
		return &apiErr
	}
}

func http_status_message(code int) string {
	switch code {
	case 400:
		return "bad request"
	case 500:
		return "internal server error"
	case 503:
		return "service unavailable"
	default:
		return fmt.Sprintf("unexpected status %d", code)
	}
}

// IsNotFound reports whether err indicates an HTTP 404 response.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "404") && strings.Contains(msg, "not found")
}
