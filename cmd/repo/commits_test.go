package repo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- agent help content (per-verb) -----

func TestCommitsAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo commits",
		"## Purpose",
		"## Subcommands",
		"## Pre-conditions",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## OAuth scope required",
		"## Rate limit",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoCommitsAgentHelp, want) {
			t.Errorf("commits umbrella agent help missing required section: %q", want)
		}
	}
}

func TestCommitsListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo commits list",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoCommitsListAgentHelp, want) {
			t.Errorf("commits list agent help missing required section: %q", want)
		}
	}
}

func TestCommitsViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo commits view",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoCommitsViewAgentHelp, want) {
			t.Errorf("commits view agent help missing required section: %q", want)
		}
	}
}

func TestCompareAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo compare",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoCompareAgentHelp, want) {
			t.Errorf("compare agent help missing required section: %q", want)
		}
	}
}

// ----- flag coverage -----

// TestCommitsAndCompareFlagsCoveredInAgentHelp ensures every flag
// exposed by the commits/compare commands is also documented in its
// agent help string.
func TestCommitsAndCompareFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// commits list
		{"--repo", repoCommitsListAgentHelp},
		{"-R", repoCommitsListAgentHelp},
		{"--sha", repoCommitsListAgentHelp},
		{"--path", repoCommitsListAgentHelp},
		{"--author", repoCommitsListAgentHelp},
		{"--since", repoCommitsListAgentHelp},
		{"--until", repoCommitsListAgentHelp},
		{"--paginate", repoCommitsListAgentHelp},
		{"--limit", repoCommitsListAgentHelp},

		// commits view
		{"--repo", repoCommitsViewAgentHelp},
		{"-R", repoCommitsViewAgentHelp},

		// compare
		{"--repo", repoCompareAgentHelp},
		{"-R", repoCompareAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}

// ----- wire-format + endpoint contract tests (httptest) -----

// TestReposCommitsListEndpoint verifies the basic list path (no query
// params) and parses cleanly into []types.Commit.
func TestReposCommitsListEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"sha":"a1b2c3d4e5f6",
				"url":"https://gitee.com/o/r/commit/a1b2c3d",
				"html_url":"https://gitee.com/o/r/commit/a1b2c3d",
				"commit":{
					"message":"Fix login bug",
					"author":{"name":"Alice","email":"a@e.com","date":"2026-06-01T12:00:00Z"},
					"committer":{"name":"Alice","email":"a@e.com","date":"2026-06-01T12:00:00Z"}
				},
				"author":{"login":"alice"},
				"committer":{"login":"alice"},
				"parents":[{"sha":"b2c3d4e5f6a7"}]
			}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	var commits []types.Commit
	err := client.Get(t.Context(), api.ReposCommitsList("owner", "repo"), &commits)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/repos/owner/repo/commits", gotPath)
	require.Len(t, commits, 1)
	assert.Equal(t, "a1b2c3d4e5f6", commits[0].SHA)
	require.NotNil(t, commits[0].Commit)
	assert.Equal(t, "Fix login bug", commits[0].Commit.Message)
	assert.Equal(t, "Alice", commits[0].Commit.Author.Name)
	require.NotNil(t, commits[0].Author)
	assert.Equal(t, "alice", commits[0].Author.Login)
	require.Len(t, commits[0].Parents, 1)
	assert.Equal(t, "b2c3d4e5f6a7", commits[0].Parents[0].SHA)
}

// TestRepoCommitGetEndpoint verifies the single-commit path is built
// with the SHA URL-escaped and parses the detail endpoint payload
// (which includes stats and files).
func TestRepoCommitGetEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"sha":"abc1234567890",
			"html_url":"https://gitee.com/o/r/commit/abc1234",
			"commit":{
				"message":"Multi-line\ncommit message",
				"author":{"name":"Alice","email":"a@e.com","date":"2026-06-01T12:00:00Z"},
				"committer":{"name":"Alice","email":"a@e.com","date":"2026-06-01T12:00:00Z"}
			},
			"stats":{"additions":12,"deletions":3,"total":15},
			"files":[
				{"filename":"src/auth.go","additions":8,"deletions":2,"status":"modified"},
				{"filename":"test/auth_test.go","additions":4,"deletions":1,"status":"added"}
			]
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	var cm types.Commit
	err := client.Get(t.Context(), api.RepoCommitGet("owner", "repo", "abc1234"), &cm)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/repos/owner/repo/commits/abc1234", gotPath)
	assert.Equal(t, "abc1234567890", cm.SHA)
	require.NotNil(t, cm.Commit)
	assert.Equal(t, "Multi-line\ncommit message", cm.Commit.Message)
	require.NotNil(t, cm.Stats)
	assert.Equal(t, 12, cm.Stats.Additions)
	assert.Equal(t, 3, cm.Stats.Deletions)
	assert.Equal(t, 15, cm.Stats.Total)
	require.Len(t, cm.Files, 2)
	assert.Equal(t, "src/auth.go", cm.Files[0].Filename)
	assert.Equal(t, "added", cm.Files[1].Status)
}

// TestRepoCommitGetBranchRef verifies a branch-style ref (e.g. "main")
// is URL-escaped correctly so the encoded path is parseable.
func TestRepoCommitGetBranchRef(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sha":"branchesha","commit":{"message":"on main"}}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	var cm types.Commit
	err := client.Get(t.Context(), api.RepoCommitGet("owner", "repo", "main"), &cm)
	require.NoError(t, err)
	// Resty decodes the path; just check it ends with the expected segment.
	assert.True(t, strings.HasSuffix(gotPath, "/commits/main"),
		"path %q must end with /commits/main", gotPath)
}

// TestReposCompareEndpoint verifies the compare path shape and that
// the response parses cleanly into types.CompareResult.
func TestReposCompareEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"diverged",
			"ahead_by":2,
			"behind_by":1,
			"total_commits":3,
			"html_url":"https://gitee.com/o/r/compare/main...feature/x",
			"commits":[
				{"sha":"c1","commit":{"author":{"name":"Alice"},"message":"first"}},
				{"sha":"c2","commit":{"author":{"name":"Bob"},"message":"second"}}
			],
			"files":[
				{"filename":"a.go","additions":10,"deletions":2,"status":"modified"}
			]
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	var result types.CompareResult
	err := client.Get(t.Context(), api.ReposCompare("owner", "repo", "main", "feature/x"), &result)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	// Resty will percent-decode the path; assert the canonical segments
	// are present rather than testing the exact escaped form.
	assert.True(t, strings.HasPrefix(gotPath, "/repos/owner/repo/compare/"),
		"path %q must start with the compare prefix", gotPath)
	assert.True(t, strings.Contains(gotPath, "main"), "path must contain base ref")
	assert.True(t, strings.Contains(gotPath, "feature"), "path must contain head ref")
	assert.Equal(t, "diverged", result.Status)
	assert.Equal(t, 2, result.AheadBy)
	assert.Equal(t, 1, result.BehindBy)
	assert.Equal(t, 3, result.TotalCommits)
	require.Len(t, result.Commits, 2)
	assert.Equal(t, "c1", result.Commits[0].SHA)
	require.Len(t, result.Files, 1)
	assert.Equal(t, "a.go", result.Files[0].Filename)
}

// ----- pure helper tests -----

// TestShortSHA verifies the helper that trims long SHAs to 7 chars but
// leaves shorter inputs (short-SHA queries, branch names) untouched.
func TestShortSHA(t *testing.T) {
	cases := map[string]string{
		"":                     "",
		"abc":                  "abc",
		"abcdefg":              "abcdefg",
		"abcdefgh":             "abcdefg", // first 7 chars
		"a1b2c3d4e5f6a7b8c9d0": "a1b2c3d",
	}
	for in, want := range cases {
		if got := shortSHA(in); got != want {
			t.Errorf("shortSHA(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestFirstLine trims the message to its first line for the list table.
func TestFirstLine(t *testing.T) {
	cases := map[string]string{
		"single line":          "single line",
		"first\nsecond":        "first",
		"first\r\nsecond":      "first",
		"first\nsecond\nthird": "first",
		"":                     "",
	}
	for in, want := range cases {
		if got := firstLine(in); got != want {
			t.Errorf("firstLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCommitDate formats the author date to YYYY-MM-DD.
func TestCommitDate(t *testing.T) {
	cm := types.Commit{
		Commit: &types.CommitInner{
			Author: types.CommitAuthor{Date: "2026-06-01T12:34:56Z"},
		},
	}
	if got := commitDate(&cm); got != "2026-06-01" {
		t.Errorf("commitDate = %q, want %q", got, "2026-06-01")
	}
	// No nested commit object → empty string.
	empty := types.Commit{}
	if got := commitDate(&empty); got != "" {
		t.Errorf("commitDate(empty) = %q, want empty", got)
	}
}

// TestAuthorNamePicksBestLabel ensures the helper chooses the most
// useful display label from the available identity fields.
func TestAuthorNamePicksBestLabel(t *testing.T) {
	cases := []struct {
		name string
		in   types.Commit
		want string
	}{
		{
			name: "Gitee user login wins",
			in: types.Commit{
				Author: &types.User{Login: "alice"},
				Commit: &types.CommitInner{Author: types.CommitAuthor{Name: "Alice Q", Username: "alice_q"}},
			},
			want: "alice",
		},
		{
			name: "falls back to commit.author.username",
			in: types.Commit{
				Commit: &types.CommitInner{Author: types.CommitAuthor{Name: "Alice Q", Username: "alice_q"}},
			},
			want: "alice_q",
		},
		{
			name: "falls back to commit.author.name",
			in: types.Commit{
				Commit: &types.CommitInner{Author: types.CommitAuthor{Name: "Alice Q"}},
			},
			want: "Alice Q",
		},
		{
			name: "unknown when nothing is set",
			in:   types.Commit{},
			want: "-",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := c.in
			if got := authorName(&v); got != c.want {
				t.Errorf("authorName = %q, want %q", got, c.want)
			}
		})
	}
}

// TestCommitMessagePrefersTopLevel confirms the helper prefers the
// top-level Message field, falling back to commit.message.
func TestCommitMessagePrefersTopLevel(t *testing.T) {
	full := types.Commit{
		Message: "top",
		Commit:  &types.CommitInner{Message: "nested"},
	}
	if got := commitMessage(&full); got != "top" {
		t.Errorf("commitMessage(top+nested) = %q, want top", got)
	}
	nestedOnly := types.Commit{
		Commit: &types.CommitInner{Message: "nested"},
	}
	if got := commitMessage(&nestedOnly); got != "nested" {
		t.Errorf("commitMessage(nested only) = %q, want nested", got)
	}
	none := types.Commit{}
	if got := commitMessage(&none); got != "" {
		t.Errorf("commitMessage(empty) = %q, want empty", got)
	}
}

// TestSplitCompareArgs validates both the single-argument "<base>...<head>"
// form and the two-argument form, plus the rejection cases.
func TestSplitCompareArgs(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantBase string
		wantHead string
		wantErr  bool
	}{
		{"single arg with dots", []string{"main...feature/x"}, "main", "feature/x", false},
		{"single arg with spaces around dots", []string{"main ... feature/x"}, "main", "feature/x", false},
		{"single arg identical refs", []string{"v1.0.0...v1.0.0"}, "v1.0.0", "v1.0.0", false},
		{"two args", []string{"main", "feature/x"}, "main", "feature/x", false},
		{"two args with whitespace", []string{"  main  ", "  feature/x  "}, "main", "feature/x", false},
		{"missing dots in single arg", []string{"main feature/x"}, "", "", true},
		{"empty base in single arg", []string{"...head"}, "", "", true},
		{"empty head in single arg", []string{"base..."}, "", "", true},
		{"empty base in two args", []string{"", "head"}, "", "", true},
		{"no args", []string{}, "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base, head, err := splitCompareArgs(c.args)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error, got base=%q head=%q", base, head)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if base != c.wantBase || head != c.wantHead {
				t.Errorf("got base=%q head=%q, want base=%q head=%q",
					base, head, c.wantBase, c.wantHead)
			}
		})
	}
}

// TestCommitListAndCompareJSONWireFormat sanity-checks that the
// struct tags produce a JSON payload that mirrors the Gitee swagger
// shape (camelCase keys, no surprises).
func TestCommitListAndCompareJSONWireFormat(t *testing.T) {
	cm := types.Commit{
		SHA:     "abc",
		HTMLURL: "https://example/c/abc",
		Commit: &types.CommitInner{
			Message: "msg",
			Author:  types.CommitAuthor{Name: "Alice", Email: "a@e.com", Date: "2026-06-01"},
		},
		Stats: &types.CommitStats{Additions: 1, Deletions: 2, Total: 3},
	}
	data, err := json.Marshal(cm)
	require.NoError(t, err)
	s := string(data)
	for _, want := range []string{
		`"sha":"abc"`,
		`"html_url":"https://example/c/abc"`,
		`"message":"msg"`,
		`"additions":1`,
		`"total":3`,
	} {
		assert.Contains(t, s, want, "commit JSON must contain %s", want)
	}

	res := types.CompareResult{
		Status:       "diverged",
		AheadBy:      5,
		BehindBy:     1,
		TotalCommits: 6,
		HTMLURL:      "https://example/cmp",
	}
	data2, err := json.Marshal(res)
	require.NoError(t, err)
	s2 := string(data2)
	for _, want := range []string{
		`"status":"diverged"`,
		`"ahead_by":5`,
		`"behind_by":1`,
		`"total_commits":6`,
		`"html_url":"https://example/cmp"`,
	} {
		assert.Contains(t, s2, want, "compare JSON must contain %s", want)
	}
}
