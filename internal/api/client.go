package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

const DefaultBaseURL = "https://gitee.com/api/v5"

// Client wraps resty with Gitee-specific defaults.
type Client struct {
	r       *resty.Client
	BaseURL string
	Verbose bool
}

// NewClient creates an API client authenticated with the given token.
func NewClient(token, baseURL string, verbose bool, timeout int) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if timeout == 0 {
		timeout = 30
	}

	r := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(time.Duration(timeout) * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(8 * time.Second).
		// Only retry GET requests. A write (POST/PUT/PATCH/DELETE) that
		// returns 5xx — or fails with a network error — may already have
		// taken effect on the server (Gitee does not guarantee the
		// request was not applied on a 5xx/timeout). Retrying such a
		// call would re-issue the write and risk creating duplicate
		// resources (duplicate issues, PRs, releases, keys, ...).
		// resty's condition func receives the response; for a network
		// error err != nil and resp may still carry the original
		// Request, so resp.Request.Method lets us distinguish the verb
		// in both the 5xx and the err!=nil branches.
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			method := ""
			if resp != nil && resp.Request != nil {
				method = resp.Request.Method
			}
			if method != "GET" {
				return false
			}
			return err != nil || resp.StatusCode() >= 500
		})

	if token != "" {
		r.SetHeader("Authorization", "token "+token)
	}

	if verbose {
		r.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
			fmt.Printf("> %s %s\n", req.Method, req.URL)
			if req.Body != nil {
				b, _ := json.MarshalIndent(req.Body, "", "  ")
				fmt.Printf("> Body: %s\n", b)
			}
			return nil
		})
		r.OnAfterResponse(func(_ *resty.Client, resp *resty.Response) error {
			fmt.Printf("< HTTP %d\n", resp.StatusCode())
			return nil
		})
	}

	return &Client{r: r, BaseURL: baseURL, Verbose: verbose}
}

// Get issues a GET request and decodes the JSON response into result.
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	resp, err := c.r.R().
		SetContext(ctx).
		SetResult(result).
		Get(path)
	if err != nil {
		return err
	}
	return checkStatus(resp)
}

// Post issues a POST request with a JSON body and decodes the response.
func (c *Client) Post(ctx context.Context, path string, body, result interface{}) error {
	return c.do(ctx, resty.MethodPost, path, body, result)
}

// Patch issues a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, body, result interface{}) error {
	return c.do(ctx, resty.MethodPatch, path, body, result)
}

// Delete issues a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.r.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	return checkStatus(resp)
}

// DeleteWithBody issues a DELETE request with a JSON body. Some Gitee v5
// endpoints (notably .../contents/{path} for file removal) require
// parameters in the request body even though the verb is DELETE; resty's
// SetBody + Delete combination handles this transparently. If `result`
// is non-nil, the response body is decoded into it.
func (c *Client) DeleteWithBody(ctx context.Context, path string, body interface{}) error {
	req := c.r.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json")
	if body != nil {
		req = req.SetBody(body)
	}
	resp, err := req.Delete(path)
	if err != nil {
		return err
	}
	return checkStatus(resp)
}

// Put issues a PUT request with a JSON body and decodes the response.
func (c *Client) Put(ctx context.Context, path string, body, result interface{}) error {
	return c.do(ctx, resty.MethodPut, path, body, result)
}

// do issues a JSON request with the given verb, optionally setting body and
// result, and checks the response status. It is the shared implementation
// behind Post/Patch/Put.
func (c *Client) do(ctx context.Context, verb, path string, body, result interface{}) error {
	req := c.r.R().SetContext(ctx).SetHeader("Content-Type", "application/json")
	if body != nil {
		req = req.SetBody(body)
	}
	if result != nil {
		req = req.SetResult(result)
	}
	resp, err := req.Execute(verb, path)
	if err != nil {
		return err
	}
	return checkStatus(resp)
}

// UploadFile performs a multipart/form-data POST with a file attachment.
// fileField is the form field name for the file (e.g. "file").
// filePath is the absolute or relative path to the local file.
// extraParams are optional additional form fields (e.g. "label").
// If result is non-nil, the JSON response body is decoded into it.
//
// This method uses resty's SetFile which handles Content-Type detection
// and multipart encoding automatically. It was added in v0.10.0 to
// support `gitee release upload`.
func (c *Client) UploadFile(ctx context.Context, path, fileField, filePath string, extraParams map[string]string, result interface{}) error {
	req := c.r.R().SetContext(ctx)
	if filePath != "" {
		req = req.SetFile(fileField, filePath)
	}
	if len(extraParams) > 0 {
		req = req.SetFormData(extraParams)
	}
	if result != nil {
		req = req.SetResult(result)
	}
	resp, err := req.Post(path)
	if err != nil {
		return err
	}
	return checkStatus(resp)
}

// GetRaw returns the raw response body for a GET request.
func (c *Client) GetRaw(ctx context.Context, path string) ([]byte, error) {
	resp, err := c.r.R().SetContext(ctx).Get(path)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(resp); err != nil {
		return nil, err
	}
	return resp.Body(), nil
}
