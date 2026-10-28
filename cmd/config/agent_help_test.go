package config

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigAgentHelpIsNA verifies every config subcommand prints
// the operational/config n/a notice to stderr and exits 0 when
// probed with --agent-help. Config verbs are write/operational;
// they have no agent-help documentation, and probing them with
// --agent-help used to either hit the args validator (get/set/unset)
// or actually execute the verb (list, which prints the preferences).
// Either behavior is a trap; the n/a notice is the right answer.
func TestConfigAgentHelpIsNA(t *testing.T) {
	bin := buildThrowawayBinary(t)

	cases := []struct {
		name string
		args []string
	}{
		{"list", []string{"config", "list", "--agent-help"}},
		{"get", []string{"config", "get", "output", "--agent-help"}},
		{"set", []string{"config", "set", "output", "json", "--agent-help"}},
		{"unset", []string{"config", "unset", "output", "--agent-help"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			run := exec.Command(bin, c.args...)
			run.Env = []string{
				"HOME=" + t.TempDir(),
				"PATH=/usr/bin:/bin",
				"GITEE_TOKEN=",
				"GITEE_HOSTNAME=invalid.example.invalid",
			}
			out, err := run.CombinedOutput()
			if err != nil {
				t.Fatalf("gitee %s --agent-help failed: %v\noutput:\n%s",
					strings.Join(c.args, " "), err, out)
			}
			if !strings.Contains(string(out), "operational/config command") {
				t.Errorf("expected n/a notice on stderr, got:\n%s", out)
			}
		})
	}
}

// TestConfigCmdWiring sanity-checks all 4 config subcommands are
// registered, so a future refactor that drops one fails fast here.
func TestConfigCmdWiring(t *testing.T) {
	if ConfigCmd == nil {
		t.Fatal("ConfigCmd is nil")
	}
	got := map[string]bool{}
	for _, c := range ConfigCmd.Commands() {
		got[c.Name()] = true
	}
	for _, want := range []string{"list", "get", "set", "unset"} {
		if !got[want] {
			t.Errorf("ConfigCmd missing subcommand %q", want)
		}
	}
}

func buildThrowawayBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gitee-test")
	cmd := exec.Command("go", "build", "-o", bin, "../../cmd/gitee")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build throwaway gitee: %v\n%s", err, out)
	}
	return bin
}
