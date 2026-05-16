package utils

import "strings"

// Truncate truncates a string to maxLen runes, adding "..." if truncated.
// Uses rune count (not byte count) so it never splits multi-byte UTF-8.
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// TruncateDate truncates a date string to 10 characters (YYYY-MM-DD).
func TruncateDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// ParseOwnerRepo parses "owner/repo" string into owner and repo parts.
func ParseOwnerRepo(input string) (owner, repo string) {
	if input == "" {
		return "", ""
	}
	idx := strings.Index(input, "/")
	if idx == -1 {
		return "", input
	}
	return input[:idx], input[idx+1:]
}
