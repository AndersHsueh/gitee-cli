package utils

import "fmt"

// MaxNameLen is the longest name allowed for user-supplied,
// shell-callable identifiers (alias names, extension names). Long
// names are an anti-signal: if you need 40+ chars, write a real
// command.
const MaxNameLen = 40

// ValidateName returns an error if name is not a safe, shell-callable
// identifier. Rules:
//   - non-empty
//   - ≤ 40 characters
//   - does not start with '.' or '-'
//   - only [a-z0-9_-]
//
// Used by both the alias and extension stores (and shared so the
// rules cannot drift between them). The noun argument is the
// identifier's kind — "alias", "extension" — for the error string.
func ValidateName(noun, name string) error {
	if name == "" {
		return fmt.Errorf("%s name is empty", noun)
	}
	if len(name) > MaxNameLen {
		return fmt.Errorf("%s name %q too long (max %d chars)", noun, name, MaxNameLen)
	}
	if name[0] == '.' || name[0] == '-' {
		return fmt.Errorf("%s name %q must not start with '.' or '-'", noun, name)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return fmt.Errorf("%s name %q contains invalid character %q (allowed: a-z, 0-9, -, _)", noun, name, r)
		}
	}
	return nil
}
