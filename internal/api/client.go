package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
		AddRetryCondition(func(resp *resty.Response, err error) bool {
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
	req := c.r.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json")
	if body != nil {
		req = req.SetBody(body)
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

// Patch issues a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, body, result interface{}) error {
	req := c.r.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json")
	if body != nil {
		req = req.SetBody(body)
	}
	if result != nil {
		req = req.SetResult(result)
	}
	resp, err := req.Patch(path)
	if err != nil {
		return err
	}
	return checkStatus(resp)
}

// Delete issues a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.r.R().SetContext(ctx).Delete(path)
	if err != nil {
		return err
	}
	return checkStatus(resp)
}

// GetRaw returns the raw response body for a GET request.
func (c *Client) GetRaw(ctx context.Context, path string) ([]byte, *http.Header, error) {
	resp, err := c.r.R().SetContext(ctx).Get(path)
	if err != nil {
		return nil, nil, err
	}
	if err := checkStatus(resp); err != nil {
		return nil, nil, err
	}
	h := resp.Header()
	return resp.Body(), &h, nil
}
