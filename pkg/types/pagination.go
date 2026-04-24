package types

import (
	"net/url"
	"strconv"
)

// PageLinks represents pagination links from HTTP Link header.
type PageLinks struct {
	Next  string
	Prev  string
	Last  string
	First string
}

// PaginatedResponse wraps a slice with pagination info.
type PaginatedResponse[T any] struct {
	Items   []T
	Total   int
	Page    int
	PerPage int
	HasMore bool
}

// PageOptions contains pagination parameters.
type PageOptions struct {
	Page    int
	PerPage int
}

// DefaultPageOptions returns sensible defaults.
func DefaultPageOptions() PageOptions {
	return PageOptions{
		Page:    1,
		PerPage: 30,
	}
}

// Query returns URL query string.
func (p PageOptions) Query() string {
	q := url.Values{}
	q.Set("page", strconv.Itoa(p.Page))
	q.Set("per_page", strconv.Itoa(p.PerPage))
	return q.Encode()
}
