package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string unchanged", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate with dots", "hello world", 8, "hello..."},
		{"maxLen 3 no dots", "hello", 3, "hel"},
		{"maxLen 2 no dots", "hello", 2, "he"},
		{"empty string", "", 5, ""},
		{"unicode aware", "你好世界测试", 3, "你好世"},
		{"unicode exact", "你好世界", 4, "你好世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"full datetime", "2024-01-15T12:30:00Z", "2024-01-15"},
		{"already date", "2024-01-15", "2024-01-15"},
		{"short string", "abc", "abc"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateDate(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
	}{
		{"standard format", "owner/repo", "owner", "repo"},
		{"no slash", "repoonly", "", "repoonly"},
		{"empty", "", "", ""},
		{"multiple slashes", "owner/repo/sub", "owner", "repo/sub"},
		{"org/repo", "gitee-cli/gitee", "gitee-cli", "gitee"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := ParseOwnerRepo(tt.input)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantRepo, repo)
		})
	}
}

func TestResolveRepo(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		repo      string
		args      []string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"from args", "", "", []string{"owner/repo"}, "owner", "repo", false},
		{"from owner+repo", "owner", "repo", nil, "owner", "repo", false},
		{"from combined repo", "", "owner/repo", nil, "owner", "repo", false},
		{"args override others", "other", "repo", []string{"owner/repo"}, "owner", "repo", false},
		{"empty all", "", "", nil, "", "", true},
		{"invalid format", "", "no-slash", nil, "", "", true},
		{"owner only", "owner", "", nil, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ResolveRepo(tt.owner, tt.repo, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOwner, owner)
				assert.Equal(t, tt.wantRepo, repo)
			}
		})
	}
}

func TestParseGistFileFlag(t *testing.T) {
	// Create a temp file for the @path cases.
	tmpFile := filepath.Join(t.TempDir(), "content.txt")
	const tmpContent = "file body line 1\nline 2"
	if err := os.WriteFile(tmpFile, []byte(tmpContent), 0600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	tests := []struct {
		name        string
		input       string
		wantName    string
		wantContent string
		wantErr     bool
	}{
		{"inline content", "readme.md=hello world", "readme.md", "hello world", false},
		{"content with equals", "f.txt=a=b=c", "f.txt", "a=b=c", false},
		{"from file path", "readme.md=@" + tmpFile, "readme.md", tmpContent, false},
		{"missing equals", "noequals", "", "", true},
		{"empty filename", "=content", "", "content", false},
		{"nonexistent file", "x=@/does/not/exist.xyz", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, content, err := ParseGistFileFlag(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantContent, content)
		})
	}
}
