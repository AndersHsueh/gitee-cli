package api

import (
	"net/http"
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
