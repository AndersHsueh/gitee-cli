package utils

import (
	"fmt"
	"strings"
)

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

// ResolveRepo 从 --owner/--repo flag 和全局 --repo flag 中解析 owner 和 repo。
// 优先使用 args（位置参数），其次使用显式 owner+repo，最后使用全局 --repo。
func ResolveRepo(owner, repo string, args []string) (string, string, error) {
	if len(args) > 0 {
		o, r := ParseOwnerRepo(args[0])
		if o != "" && r != "" {
			return o, r, nil
		}
	}
	if owner != "" && repo != "" {
		return owner, repo, nil
	}
	combined := repo
	if combined == "" {
		combined = ""
	}
	if combined == "" {
		return "", "", fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}
	o, r := ParseOwnerRepo(combined)
	if o == "" || r == "" {
		return "", "", fmt.Errorf("invalid --repo format %q, expected owner/repo", combined)
	}
	return o, r, nil
}
