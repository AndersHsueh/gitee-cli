package utils

import (
	"os/exec"
	"testing"
)

func TestParseGiteeRemoteURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantOwner   string
		wantRepo    string
		wantErr     bool
		errContains string
	}{
		// --- HTTPS form, gitee.com ---
		{
			name:      "https gitee.com with .git",
			input:     "https://gitee.com/andershsueh/gitee-cli.git",
			wantOwner: "andershsueh",
			wantRepo:  "gitee-cli",
		},
		{
			name:      "https gitee.com without .git",
			input:     "https://gitee.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "https gitee.com with trailing slash",
			input:     "https://gitee.com/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		// --- HTTPS form, enterprise ---
		{
			name:      "https enterprise gitee",
			input:     "https://gitee.example.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "https enterprise gitee with subdomain",
			input:     "https://git.gitee.company.io/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		// --- SSH form ---
		{
			name:      "ssh gitee.com with .git",
			input:     "git@gitee.com:andershsueh/gitee-cli.git",
			wantOwner: "andershsueh",
			wantRepo:  "gitee-cli",
		},
		{
			name:      "ssh gitee.com without .git",
			input:     "git@gitee.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "ssh enterprise gitee",
			input:     "git@gitee.example.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		// --- Edge cases: names with dashes/underscores/dots ---
		{
			name:      "repo name with dashes",
			input:     "https://gitee.com/owner/my-cool-repo.git",
			wantOwner: "owner",
			wantRepo:  "my-cool-repo",
		},
		{
			name:      "repo name with underscores",
			input:     "https://gitee.com/owner/my_cool_repo.git",
			wantOwner: "owner",
			wantRepo:  "my_cool_repo",
		},
		{
			name:      "repo name with dots",
			input:     "https://gitee.com/owner/repo.v2.git",
			wantOwner: "owner",
			wantRepo:  "repo.v2",
		},
		// --- Whitespace handling ---
		{
			name:      "leading and trailing whitespace",
			input:     "  https://gitee.com/owner/repo.git  ",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		// --- Errors ---
		{
			name:        "empty",
			input:       "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "non-gitee host https",
			input:       "https://github.com/owner/repo.git",
			wantErr:     true,
			errContains: "not a Gitee host",
		},
		{
			name:        "non-gitee host ssh",
			input:       "git@github.com:owner/repo.git",
			wantErr:     true,
			errContains: "not a Gitee host",
		},
		{
			name:        "garbage input",
			input:       "not a url at all",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "https missing path components",
			input:       "https://gitee.com/owner",
			wantErr:     true,
			errContains: "unrecognized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo, err := ParseGiteeRemoteURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGiteeRemoteURL(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if gotOwner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", gotOwner, tt.wantOwner)
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", gotRepo, tt.wantRepo)
			}
		})
	}
}

// TestIsInsideWorkTree is best-effort: it only asserts that the function
// returns without panicking and that the return value is consistent
// across two calls (i.e. it doesn't read random memory). The "is the
// test running inside a git repo?" answer depends on the test runner's
// CWD, so we don't make a fixed assertion either way.
func TestIsInsideWorkTree(t *testing.T) {
	a := IsInsideWorkTree()
	b := IsInsideWorkTree()
	if a != b {
		t.Errorf("IsInsideWorkTree returned inconsistent results: %v vs %v", a, b)
	}
	// Sanity: when not in a repo (which we can't reliably simulate without
	// chdir), at least the function should not return true if git is missing.
	if _, err := exec.LookPath("git"); err != nil {
		// git not on PATH — function should return false, not error out
		if a {
			t.Errorf("IsInsideWorkTree() = true but git is not installed")
		}
	}
}

// contains is a small helper to keep test file stdlib-free of strings.Contains
// in the failure path (and to give better error messages than assert.Contains).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
