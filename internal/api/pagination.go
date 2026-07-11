package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// FetchAllPages iterates paginated GET requests via the provided client and
// appends all items into the slice pointed to by `result`.
//
// Stops when:
//   - a page returns 0 items (no more data)
//   - a page returns fewer than perPage items (last page detected)
//   - limit (>= 1) items have been collected
//
// Parameters:
//   - perPage: items per request, must be 1-100 (Gitee API hard max)
//   - limit:   total cap on returned items; 0 means no cap
//
// basePath may already contain a query string; FetchAllPages preserves it
// and adds/overrides page + per_page. The path is used as-is — no scheme/host
// rewriting is performed (the client already knows its base URL).
//
// result must be a non-nil pointer to a slice (*[]T). If pagination fails
// partway, the slice contains the items fetched so far (best-effort),
// and the returned error describes what went wrong.
//
// This function is the canonical way to "list everything" in gitee-cli.
// It was added in v0.4.1 to fix a long-standing bug where list commands
// (issue list, pr list) silently dropped everything past page 1.
func FetchAllPages[T any](ctx context.Context, client *Client, basePath string, perPage, limit int, result *[]T) error {
	if client == nil {
		return fmt.Errorf("FetchAllPages: client is nil")
	}
	if result == nil {
		return fmt.Errorf("FetchAllPages: result must be a non-nil pointer to a slice")
	}
	if perPage < 1 || perPage > 100 {
		return fmt.Errorf("FetchAllPages: perPage must be 1-100, got %d", perPage)
	}
	if limit < 0 {
		return fmt.Errorf("FetchAllPages: limit must be >= 0, got %d", limit)
	}
	if *result == nil {
		*result = []T{}
	}

	// Parse the base path so we can preserve its existing query params
	// and override page/per_page on each iteration.
	parsed, err := url.Parse(basePath)
	if err != nil {
		return fmt.Errorf("FetchAllPages: invalid basePath %q: %w", basePath, err)
	}
	existingQuery := parsed.Query()

	for page := 1; ; page++ {
		// Build the query for this page.
		q := url.Values{}
		for k, v := range existingQuery {
			q[k] = v
		}
		q.Set("page", strconv.Itoa(page))
		q.Set("per_page", strconv.Itoa(perPage))
		parsed.RawQuery = q.Encode()
		pagePath := parsed.String()

		// Fetch this page into a fresh slice.
		var pageItems []T
		if err := client.Get(ctx, pagePath, &pageItems); err != nil {
			return fmt.Errorf("FetchAllPages: page %d: %w", page, err)
		}

		// Empty page = we've reached the end.
		if len(pageItems) == 0 {
			return nil
		}

		*result = append(*result, pageItems...)

		// Total cap reached.
		if limit > 0 && len(*result) >= limit {
			*result = (*result)[:limit]
			return nil
		}

		// Short page = this was the last one (server didn't fill the page).
		// No need to fetch another empty page just to discover that.
		if len(pageItems) < perPage {
			return nil
		}
	}
}

// FetchListOrAll is the "do I want a single page or all pages?" dispatcher
// used by the status command (and any other future command that needs
// both modes controlled by the user's --paginate flag).
//
// Behavior:
//   - paginate=true  → call FetchAllPages; result is bounded by limit (0 = no cap)
//   - paginate=false → call client.Get once, honoring cmd.GlobalPage/--page
//     via WithPageParams; a 404 is treated as an empty result (not an error)
//     because "no matching items" is a valid answer, not a failure.
//
// perPage is clamped to 1-100 (Gitee's hard max). This matches what the
// per-list command's inline perPage clamp has been doing; centralizing it
// here lets us delete the duplicated blocks.
//
// errCtx is a short prefix prepended to any non-404 error (e.g.
// "listing issues assigned to you"). It is purely for log message clarity.
func FetchListOrAll[T any](ctx context.Context, client *Client, basePath string, paginate bool, perPage, limit, page int, errCtx string) ([]T, error) {
	perPage = clampPerPage(perPage)
	var out []T
	if paginate {
		if err := FetchAllPages(ctx, client, basePath, perPage, limit, &out); err != nil {
			return nil, fmt.Errorf("%s: %w", errCtx, err)
		}
		return out, nil
	}
	// Single page. Apply the user's --page so the request matches what they
	// asked for (without this, --page was silently ignored on single-page paths).
	path, err := WithPageParams(basePath, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errCtx, err)
	}
	if err := client.Get(ctx, path, &out); err != nil {
		if IsNotFound(err) {
			return nil, nil // empty list is a valid answer
		}
		return nil, fmt.Errorf("%s: %w", errCtx, err)
	}
	return out, nil
}

// clampPerPage enforces Gitee v5's per_page range of 1-100. Values out of
// range are normalized rather than rejected: --per-page 0 (default state
// before cobra binds a value) becomes 30; --per-page 500 becomes 100.
// This is the same behavior the per-list commands have had inline.
func clampPerPage(perPage int) int {
	if perPage < 1 {
		return 30
	}
	if perPage > 100 {
		return 100
	}
	return perPage
}

// WithPageParams appends/overrides ?page= and ?per_page= on a base path that
// may already contain a query string. Extracted from cmd/issue/list.go so
// that single-page fetches everywhere can use the same query-building logic.
func WithPageParams(basePath string, page, perPage int) (string, error) {
	u, err := url.Parse(basePath)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

