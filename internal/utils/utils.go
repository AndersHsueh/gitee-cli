package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

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

// HumanDuration converts a duration in days to a human-readable string.
func HumanDuration(days int) string {
	if days < 1 {
		return "today"
	}
	if days == 1 {
		return "1 day ago"
	}
	if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	}
	if days < 30 {
		weeks := days / 7
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}
	if days < 365 {
		months := days / 30
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := days / 365
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// FormatNumber formats a number with thousands separators.
func FormatNumber(n int) string {
	return strconv.Itoa(n)
}

// Pluralize returns the singular or plural form of a word based on count.
func Pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// Contains checks if a string slice contains a specific string.
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Unique removes duplicate strings from a slice.
func Unique(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// ParseBool parses a string into a boolean value.
func ParseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "y"
}

// CleanString removes leading/trailing whitespace and collapses multiple spaces.
func CleanString(s string) string {
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return s
}

// ToTitleCase converts a string to Title Case.
func ToTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			if len(runes) > 1 {
				runes[1] = unicode.ToLower(runes[1])
			}
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// ParseTime parses a time string in various formats.
func ParseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

// DaysSince calculates the number of days since a given time.
func DaysSince(t time.Time) int {
	return int(time.Since(t).Hours() / 24)
}

// FormatStars formats star counts with K/M suffixes for large numbers.
func FormatStars(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return strconv.Itoa(n)
}

// NormalizeVisibility normalizes visibility strings.
func NormalizeVisibility(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "pub":
		return "public"
	case "priv":
		return "private"
	case "int":
		return "internal"
	default:
		return v
	}
}

// MaskString masks a string, showing only the first and last characters.
func MaskString(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

// SplitLines splits a string into lines.
func SplitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

// JoinNonEmpty joins strings with a separator, skipping empty strings.
func JoinNonEmpty(sep string, parts ...string) string {
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return strings.Join(result, sep)
}
