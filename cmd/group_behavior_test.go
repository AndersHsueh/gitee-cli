package cmd

import (
	"strings"
	"testing"
)

// TestGroupCommandsRejectUnknownVerbs locks C2: every command group must
// fail loudly (non-zero) with an actionable suggestion when it receives a
// positional argument that is not a subcommand. Before this, cobra treated a
// non-runnable parent as "help requested": printing help and exiting 0, so
// `gitee pr creat 1` or `gitee repo unstar owner/repo` looked successful.
//
// The suggestions come from cmd.GroupRunE: exact reparented verb, edit
// distance <= 2, or (for the old flat syntax) `<group> <group> <arg>`.
func TestGroupCommandsRejectUnknownVerbs(t *testing.T) {
	bin := buildThrowawayBinary(t)
	home := t.TempDir()

	cases := []struct {
		name string
		args []string
		want string // substring that must appear in stderr (the suggestion)
	}{
		{"reparented verb", []string{"repo", "unstar", "owner/repo"}, "gitee repo star unstar owner/repo"},
		{"typo in pr", []string{"pr", "creat", "1"}, "gitee pr create 1"},
		{"reparented org verb", []string{"org", "member", "alice"}, "gitee org members alice"},
		{"typo in issue", []string{"issue", "labl", "list"}, "gitee issue label list"},
		{"typo in gist", []string{"gist", "creat"}, "gitee gist create"},
		{"typo in milestone", []string{"milestone", "craete"}, "gitee milestone create"},
		{"old flat syntax star", []string{"repo", "star", "owner/repo"}, "gitee repo star star owner/repo"},
		{"old flat syntax protect", []string{"repo", "protect", "master"}, "gitee repo protect protect master"},
		{"unknown top-level", []string{"auth", "loign"}, "gitee auth login"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runWithHome(t, bin, home, nil, tc.args...)
			if err == nil {
				t.Fatalf("gitee %s exited 0; a non-subcommand argument must be an error\n%s",
					strings.Join(tc.args, " "), out)
			}
			if !strings.Contains(out, tc.want) {
				t.Errorf("gitee %s: stderr should suggest %q; got:\n%s",
					strings.Join(tc.args, " "), tc.want, out)
			}
			if !strings.Contains(out, "command group") {
				t.Errorf("gitee %s: error should explain it is a command group; got:\n%s",
					strings.Join(tc.args, " "), out)
			}
		})
	}
}

// TestBareGroupCommandsShowHelpAndSucceed is the other half of the contract:
// with no arguments every group still prints its help and exits 0.
func TestBareGroupCommandsShowHelpAndSucceed(t *testing.T) {
	bin := buildThrowawayBinary(t)
	home := t.TempDir()

	groups := [][]string{
		{"repo"}, {"issue"}, {"issue", "label"}, {"pr"}, {"gist"}, {"org"},
		{"label"}, {"milestone"}, {"notification"}, {"release"}, {"webhook"},
		{"user"}, {"ssh-key"}, {"search"}, {"config"}, {"auth"}, {"alias"},
		{"extension"}, {"agent"}, {"completion"}, {"repo", "star"},
		{"repo", "watch"}, {"repo", "protect"}, {"repo", "tag"},
		{"repo", "file"}, {"repo", "commits"}, {"repo", "collaborator"},
	}
	for _, args := range groups {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			out, err := runWithHome(t, bin, home, nil, args...)
			if err != nil {
				t.Fatalf("gitee %s (no args) should exit 0, got %v:\n%s",
					strings.Join(args, " "), err, out)
			}
			if !strings.Contains(out, "Usage:") && !strings.Contains(out, "Available Commands:") {
				t.Errorf("gitee %s (no args) should print help; got:\n%s",
					strings.Join(args, " "), out)
			}
		})
	}
}

// TestGroupEditDistance pins the suggestion helper's threshold so a future
// refactor cannot silently widen it to "always suggests something".
func TestGroupEditDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"creat", "create", 1},
		{"labl", "label", 1},
		{"member", "members", 1},
		{"craete", "create", 2},
		{"totally-different", "repo", 15},
	}
	for _, tc := range cases {
		if got := editDistance(tc.a, tc.b); got != tc.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
