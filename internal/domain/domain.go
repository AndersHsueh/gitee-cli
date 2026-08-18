// Package domain holds the business logic for repo-scoped resources
// (labels, milestones, webhooks, notifications) with explicit dependencies.
//
// Historically this logic lived in internal/handler and imported gitee-cli/cmd
// to read cmd.GlobalOutput / cmd.GlobalTimeout / cmd.NewAPIClient — an
// internal → cmd layering inversion. Moving it here with a Client that
// carries its own *api.Client, timeout, output format, and repo fallback
// makes the internal layer a one-way bottom again and lets each operation
// be unit-tested without driving the whole command tree.
package domain

import (
	"context"
	"time"

	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
	"gitee-cli/internal/utils"
)

// Client carries the explicit dependencies a domain operation needs. The
// cmd layer builds one from its flags/globals and hands it over.
type Client struct {
	// API is the authenticated Gitee API client.
	API *api.Client
	// Timeout is the per-request context timeout.
	Timeout time.Duration
	// Output is the output format: "table" (default) or "json".
	Output string
	// Repo is the global --repo fallback used when a local flag is empty.
	Repo string
}

// ctx returns a context with the client's timeout applied.
func (c *Client) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.Timeout)
}

// IsJSON reports whether the output format is json.
func (c *Client) IsJSON() bool { return c.Output == "json" }

// ResolveRepo resolves a local repo flag with the global fallback.
func (c *Client) ResolveRepo(repo string) (string, string, error) {
	if repo == "" {
		repo = c.Repo
	}
	return utils.ResolveRepo("", repo, nil)
}

// opts builds the cmdkit.Options for a list operation from the client's
// explicit dependencies. Pagination fields are left zero because the
// domain list operations use a custom single-GET Fetch (per_page cap).
func (c *Client) opts() cmdkit.Options {
	return cmdkit.Options{
		Client:  c.API,
		Timeout: c.Timeout,
		Output:  c.Output,
		Repo:    c.Repo,
	}
}
