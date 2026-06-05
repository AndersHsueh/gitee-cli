package repo

import (
	"strings"
	"testing"
)

func TestRenameAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo rename",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(repoRenameAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

func TestParseOwnerRepoStrict(t *testing.T) {
	cases := []struct {
		in        string
		wantOwner string
		wantName  string
		wantOK    bool
	}{
		{"owner/repo", "owner", "repo", true},
		{"a/b/c", "a", "b/c", true},
		{"nope", "", "", false},
		{"/leading", "", "", false},
		{"trailing/", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			o, n, ok := parseOwnerRepoStrict(c.in)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !ok {
				return
			}
			if o != c.wantOwner || n != c.wantName {
				t.Errorf("got (%q, %q), want (%q, %q)", o, n, c.wantOwner, c.wantName)
			}
		})
	}
}
