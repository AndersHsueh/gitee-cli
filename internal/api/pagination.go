package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"gitee-cli/pkg/types"
)

// ParseLinkHeader parses HTTP Link header and returns PageLinks.
func ParseLinkHeader(header string) *types.PageLinks {
	if header == "" {
		return nil
	}

	links := &types.PageLinks{}
	parts := strings.Split(header, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "<") {
			continue
		}

		// Extract URL between < and >
		idx := strings.Index(part, ">")
		if idx == -1 {
			continue
		}
		url := part[1:idx]

		// Extract rel value
		relIdx := strings.Index(part, "rel=")
		if relIdx == -1 {
			continue
		}
		rel := part[relIdx+4:]
		if strings.HasPrefix(rel, "\"") {
			rel = rel[1:]
		}
		if idx := strings.Index(rel, "\""); idx != -1 {
			rel = rel[:idx]
		}

		switch rel {
		case "next":
			links.Next = url
		case "prev":
			links.Prev = url
		case "last":
			links.Last = url
		case "first":
			links.First = url
		}
	}

	return links
}

// GetTotalCount extracts X-Total-Count header value.
func GetTotalCount(header http.Header) int {
	v := header.Get("X-Total-Count")
	if v == "" {
		return 0
	}
	total, _ := strconv.Atoi(v)
	return total
}

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

