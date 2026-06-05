package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// IsInsideWorkTree returns true if the current directory is inside a git
// working tree (i.e. `git rev-parse --is-inside-work-tree` prints "true").
// Any error (not in a repo, git not installed, etc.) returns false.
func IsInsideWorkTree() bool {
	out, err := runGitCapture("rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "true"
}

// CurrentBranch returns the short name of the current branch, or "" if
// HEAD is detached / not in a repo. Used by `gitee status` to enrich
// the header line with the current branch.
func CurrentBranch() string {
	out, err := runGitCapture("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// GitRemoteURL returns the URL configured for the named remote (usually
// "origin"), trimmed of trailing whitespace. Returns an error if the
// remote is not configured or git fails for any reason.
//
// This is a thin wrapper around `git remote get-url <name>`; the
// capturing variant (vs the streaming `gitRun` in cmd/pr/checkout.go)
// exists so callers can parse the URL rather than just printing it.
func GitRemoteURL(remoteName string) (string, error) {
	if remoteName == "" {
		remoteName = "origin"
	}
	out, err := runGitCapture("remote", "get-url", remoteName)
	if err != nil {
		return "", fmt.Errorf("git remote get-url %s: %w", remoteName, err)
	}
	return strings.TrimSpace(out), nil
}

// runGitCapture shells out to `git <args...>` and returns combined
// stdout. stderr is captured for error context but not returned; the
// function returns a non-nil error if git exits non-zero so callers
// can use a single `if err != nil` check.
//
// We deliberately do NOT chdir or set GIT_DIR — the caller's working
// directory is what we want to inspect (so `cd /tmp && gitee status`
// sees /tmp, not the gitee-cli source repo).
//
// The error is returned as a plain string (not wrapped with %w) because
// every caller re-wraps with a higher-level message — a wrapped-wrapped
// error chain would have to be unwrapped twice by the user. Keeping
// this error a flat string makes the final user-facing message clearer.
func runGitCapture(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// giteeRemoteHTTPS matches https://[gitee.com|<enterprise-host>]/owner/repo[.git].
// The host part is anchored to "gitee" or any "gitee.<tld>" so enterprise
// installations (e.g. gitee.example.com) are also accepted. We deliberately
// allow ANY host that contains "gitee" because there's no way to enumerate
// enterprise hostnames statically.
var giteeRemoteHTTPS = regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+?)(?:\.git)?/?$`)

// giteeRemoteSSH matches git@<host>:owner/repo[.git], the SSH form. The
// host part is again any "gitee*" hostname.
var giteeRemoteSSH = regexp.MustCompile(`^git@([^:]+):([^/]+)/([^/]+?)(?:\.git)?/?$`)

// ParseGiteeRemoteURL extracts (owner, repo) from a git remote URL of
// the forms:
//
//   https://gitee.com/owner/repo[.git]
//   https://gitee.example.com/owner/repo[.git]
//   git@gitee.com:owner/repo[.git]
//   git@<enterprise>:owner/repo[.git]
//
// Returns ("", "", error) if the URL is empty, doesn't match a known
// shape, or points to a non-Gitee host (e.g. github.com).
//
// The "is this a Gitee host?" check is intentionally permissive
// (substring match on "gitee") — enterprise deployments vary widely
// and the alternative is an allowlist that breaks on first onboarding.
func ParseGiteeRemoteURL(rawURL string) (string, string, error) {
	url := strings.TrimSpace(rawURL)
	if url == "" {
		return "", "", fmt.Errorf("empty URL")
	}

	// Try SSH form first. The colon in the SSH form (git@host:owner/repo)
	// is incompatible with HTTP URL parsers, so we explicitly check it.
	if m := giteeRemoteSSH.FindStringSubmatch(url); m != nil {
		host, owner, repo := m[1], m[2], m[3]
		if !strings.Contains(strings.ToLower(host), "gitee") {
			return "", "", fmt.Errorf("not a Gitee host: %q (host=%s)", url, host)
		}
		return owner, repo, nil
	}

	// HTTPS form.
	if m := giteeRemoteHTTPS.FindStringSubmatch(url); m != nil {
		host, owner, repo := m[1], m[2], m[3]
		if !strings.Contains(strings.ToLower(host), "gitee") {
			return "", "", fmt.Errorf("not a Gitee host: %q (host=%s)", url, host)
		}
		return owner, repo, nil
	}

	return "", "", fmt.Errorf("unrecognized git remote URL format: %q (expected https://gitee.com/owner/repo[.git] or git@gitee.com:owner/repo[.git])", url)
}
