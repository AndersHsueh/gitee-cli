package repo

import (
	"fmt"
	"net/url"
	"strings"

	"gitee-cli/internal/utils"
)

// FullName represents a repository full name (owner/repo).
type FullName struct {
	Owner string
	Repo  string
}

// Parse parses a "owner/repo" string into its components.
func (f *FullName) Parse(input string) error {
	f.Owner, f.Repo = utils.ParseOwnerRepo(input)
	if f.Owner == "" || f.Repo == "" {
		return fmt.Errorf("invalid repository format: %s (expected owner/repo)", input)
	}
	return nil
}

// String returns the string representation.
func (f FullName) String() string {
	return f.Owner + "/" + f.Repo
}

// IsValid checks if the full name is valid.
func (f FullName) IsValid() bool {
	return f.Owner != "" && f.Repo != ""
}

// NormalizeOwnerRepo normalizes and validates an owner/repo string.
func NormalizeOwnerRepo(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("repository identifier is required")
	}

	// Handle URLs
	if strings.HasPrefix(input, "http") {
		u, err := url.Parse(input)
		if err == nil {
			parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
			if len(parts) >= 2 {
				return parts[len(parts)-2] + "/" + parts[len(parts)-1], nil
			}
		}
	}

	// Check format
	if !strings.Contains(input, "/") {
		return "", fmt.Errorf("invalid format: %s (expected owner/repo)", input)
	}

	parts := strings.SplitN(input, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid format: %s (expected owner/repo)", input)
	}

	return input, nil
}

// Visibility represents repository visibility.
type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityPrivate  Visibility = "private"
	VisibilityInternal Visibility = "internal"
	VisibilityAll      Visibility = "all"
)

// ParseVisibility parses a visibility string.
func ParseVisibility(s string) Visibility {
	switch strings.ToLower(s) {
	case "public", "pub":
		return VisibilityPublic
	case "private", "priv":
		return VisibilityPrivate
	case "internal", "int":
		return VisibilityInternal
	default:
		return VisibilityAll
	}
}

// CloneURL returns the appropriate clone URL for a repository.
func CloneURL(repoURL, protocol string) string {
	if repoURL == "" {
		return ""
	}
	if protocol == "ssh" {
		// Convert https URL to ssh URL
		repoURL = strings.Replace(repoURL, "https://", "git@", 1)
		repoURL = strings.Replace(repoURL, "http://", "git@", 1)
		repoURL = strings.Replace(repoURL, "gitee.com/", "gitee.com:", 1)
	}
	return repoURL
}
