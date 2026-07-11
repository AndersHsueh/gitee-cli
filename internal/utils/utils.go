package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
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
	if o, r, ok := strings.Cut(input, "/"); ok {
		return o, r
	}
	return "", input
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
		// Fallback: check git local config for gitee.default_repo (v0.10.0)
		if defaultRepo, cfgErr := GitConfigGet("gitee.default_repo"); cfgErr == nil && defaultRepo != "" {
			o, r := ParseOwnerRepo(defaultRepo)
			if o != "" && r != "" {
				return o, r, nil
			}
		}
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

// ResolveRepoFromFlag resolves a single "owner/repo" string flag (with
// fallback to the global --repo) into (owner, name, error). Mirrors the
// common pattern in per-resource handlers (label, release, webhook) of
// substituting the global --repo when the local flag is empty. The caller
// passes `globalRepo` explicitly so this package doesn't need to import
// the cmd package (avoiding an import cycle through cmd → handlers → utils).
func ResolveRepoFromFlag(repo, globalRepo string) (string, string, error) {
	if repo == "" {
		repo = globalRepo
	}
	return ResolveRepo("", repo, nil)
}

// ParseInt64Arg returns the value of `flagVal` if non-zero, otherwise parses
// `args[0]` as an int64. Returns (0, nil) when neither is set so the caller
// can decide whether to error ("id is required") or default. The `name`
// argument is used in error messages ("invalid <name> id %q: ...").
func ParseInt64Arg(flagVal int64, args []string, name string) (int64, error) {
	if flagVal != 0 {
		return flagVal, nil
	}
	if len(args) < 1 {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s id %q: must be an integer", name, args[0])
	}
	return parsed, nil
}

// BoolPtr returns a pointer to b. Useful for optional bool fields in
// PATCH payloads where the API distinguishes "true" / "false" / "unset".
func BoolPtr(b bool) *bool { return &b }

// StringPtr is the string counterpart to BoolPtr. Useful for optional
// string fields in PATCH payloads (e.g. webhook password, PR title).
func StringPtr(s string) *string { return &s }

// BoolPtrIfChanged returns nil if the user did not pass the flag (cobra
// Changed() = false → "leave unchanged"), otherwise returns a pointer to
// the new bool value. This is the cobra-idiomatic way to disambiguate
// "unset" from "set to false" on a BoolVar flag.
func BoolPtrIfChanged(c *cobra.Command, name string, v bool) *bool {
	if !c.Flags().Changed(name) {
		return nil
	}
	return &v
}

// StringPtrIfChanged is the string counterpart to BoolPtrIfChanged.
// Returns nil if the user did not pass the flag, otherwise a pointer to
// the new string value.
func StringPtrIfChanged(c *cobra.Command, name string, v string) *string {
	if !c.Flags().Changed(name) {
		return nil
	}
	return &v
}

// ResolveKeyInput accepts either a path to a public key file or the raw
// public key text itself. If raw refers to an existing, readable regular
// file, the file's contents are returned. Otherwise raw is returned
// verbatim. Trailing whitespace and newlines are stripped.
func ResolveKeyInput(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("key is empty")
	}
	if fi, err := os.Stat(trimmed); err == nil && !fi.IsDir() {
		b, err := os.ReadFile(trimmed)
		if err != nil {
			return "", fmt.Errorf("reading key file %q: %w", trimmed, err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	return trimmed, nil
}

// ParseGistFileFlag parses a --file flag value for gist create/edit.
// Accepted formats:
//
//	filename=content      inline content
//	filename=@path        read content from a local file
//
// Returns the filename and resolved content. Shared by cmd/gist/create.go
// and cmd/gist/edit.go so the parsing logic cannot drift between them.
func ParseGistFileFlag(s string) (name, content string, err error) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid --file format: %s (expected filename=content or filename=@filepath)", s)
	}
	name = parts[0]
	contentOrPath := parts[1]
	if strings.HasPrefix(contentOrPath, "@") {
		filePath := contentOrPath[1:]
		data, rErr := os.ReadFile(filePath)
		if rErr != nil {
			return "", "", fmt.Errorf("failed to read file %s: %w", filePath, rErr)
		}
		return name, string(data), nil
	}
	return name, contentOrPath, nil
}

// ConfirmYN prompts the user with a y/N question and returns true if they
// answered affirmatively. If `defaultYes` is true, an empty input is treated
// as yes. If stdin is not a terminal (piped, redirected, or closed), the
// function returns false and a nil error — this prevents scripts/CI from
// hanging on the prompt. Use --force at the call site for non-interactive
// scripts that want to bypass the prompt entirely.
func ConfirmYN(prompt string, defaultYes bool) (bool, error) {
	// Non-TTY detection: stat os.Stdin and look for "character device" mode.
	// Pipes / redirected files report ModeNamedPipe, regular files, etc. and
	// are not character devices. Avoids a hard dependency on x/term.
	if fi, err := os.Stdin.Stat(); err == nil {
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			return false, nil
		}
	}
	fmt.Printf("%s [y/N] ", prompt)
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		// EOF on stdin → treat as "no"
		return false, nil
	}
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return defaultYes, nil
	}
	return input == "y" || input == "yes", nil
}
