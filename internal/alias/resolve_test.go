package alias

import (
	"os"
	"reflect"
	"testing"
)

func TestResolveArgs(t *testing.T) {
	// Build a tiny alias file on disk so the resolver can read it.
	withTempFile(t)
	if _, _, err := Set("co", "pr checkout"); err != nil {
		t.Fatalf("seed Set: %v", err)
	}
	if _, _, err := Set("issues", "issue list --limit 50"); err != nil {
		t.Fatalf("seed Set: %v", err)
	}

	builtins := BuiltinNames([]string{"repo", "pr", "issue", "alias"})

	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "no args",
			in:   []string{"gitee"},
			want: []string{"gitee"},
		},
		{
			name: "flag only",
			in:   []string{"gitee", "--agent-help"},
			want: []string{"gitee", "--agent-help"},
		},
		{
			name: "builtin wins over alias",
			in:   []string{"gitee", "pr", "list"},
			want: []string{"gitee", "pr", "list"},
		},
		{
			name: "alias expands single token",
			in:   []string{"gitee", "co", "123"},
			want: []string{"gitee", "pr", "checkout", "123"},
		},
		{
			name: "alias with flags expands",
			in:   []string{"gitee", "issues", "--output", "json"},
			want: []string{"gitee", "issue", "list", "--limit", "50", "--output", "json"},
		},
		{
			name: "unknown command is not touched",
			in:   []string{"gitee", "nosuch", "arg"},
			want: []string{"gitee", "nosuch", "arg"},
		},
		{
			name: "alias alone",
			in:   []string{"gitee", "co"},
			want: []string{"gitee", "pr", "checkout"},
		},
		{
			name: "alias of self (idempotent in expansion)",
			// `co` -> "pr checkout" — calling it "co 1" is fine,
			// no recursion. We test simple non-recursive case.
			in:   []string{"gitee", "co"},
			want: []string{"gitee", "pr", "checkout"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveArgs(tc.in, builtins)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ResolveArgs(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolveArgsBuiltinPrecedence(t *testing.T) {
	withTempFile(t)
	// Even if a user manages to define an alias named "issue", the
	// built-in `gitee issue` must win.
	if _, _, err := Set("issue", "somethingelse"); err != nil {
		t.Fatalf("seed Set: %v", err)
	}
	builtins := BuiltinNames([]string{"issue", "pr"})
	got := ResolveArgs([]string{"gitee", "issue", "list"}, builtins)
	want := []string{"gitee", "issue", "list"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("builtin should win; ResolveArgs = %v, want %v", got, want)
	}
}

func TestResolveArgsCorruptFileGraceful(t *testing.T) {
	// Point at a corrupt aliases file. ResolveArgs must NOT panic or
	// return an error — it just falls through and lets cobra print
	// "unknown command".
	path := withTempFile(t)
	// Write invalid JSON to the file.
	if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	builtins := BuiltinNames([]string{"pr"})
	got := ResolveArgs([]string{"gitee", "co", "1"}, builtins)
	want := []string{"gitee", "co", "1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("on corrupt file, ResolveArgs should be no-op; got %v, want %v", got, want)
	}
}
