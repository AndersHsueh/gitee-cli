package utils

import "strings"

// Truncate truncates a string to maxLen characters, adding "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
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
