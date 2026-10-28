package auth

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestAuthAgentHelpIsNA verifies every auth subcommand prints the
// operational/config n/a notice to stderr and exits 0 when probed
// with --agent-help, instead of executing the real verb. Before
// task C, `gitee auth login --agent-help` actually triggered the
// login flow (printing "No token found..."), which is a trap for
// agents that probe for documentation before deciding what to do.
//
// We exercise the wiring by running a freshly-built gitee binary in
// an isolated HOME so no settings.json / token is reachable. If the
// PreRunE hook regresses, the verb would either execute (and hit the
// empty-token branch) or fail with a non-zero exit — either way the
// "operational/config — use --help" string would not appear on stderr.
func TestAuthAgentHelpIsNA(t *testing.T) {
	bin := buildThrowawayBinary(t)

	cases := []struct {
		name string
		args []string
	}{
		{"login", []string{"auth", "login", "--agent-help"}},
		{"logout", []string{"auth", "logout", "--agent-help"}},
		{"refresh", []string{"auth", "refresh", "--agent-help"}},
		{"status", []string{"auth", "status", "--agent-help"}},
		{"switch", []string{"auth", "switch", "--agent-help"}},
		{"token", []string{"auth", "token", "--agent-help"}},
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
			if !strings.Contains(string(out), "Use --help for usage") {
				t.Errorf("expected notice to redirect to --help, got:\n%s", out)
			}
		})
	}
}

// TestAuthAgentHelpDoesNotExecuteLogin makes the regression explicit:
// a probe of `auth login --agent-help` MUST NOT print the "No token
// found..." hint that the real runLogin emits when no token is
// configured. If it does, the PreRunE hook has been bypassed or
// removed and we're back to the trap.
func TestAuthAgentHelpDoesNotExecuteLogin(t *testing.T) {
	bin := buildThrowawayBinary(t)
	run := exec.Command(bin, "auth", "login", "--agent-help")
	run.Env = []string{
		"HOME=" + t.TempDir(),
		"PATH=/usr/bin:/bin",
		"GITEE_TOKEN=",
		"GITEE_HOSTNAME=invalid.example.invalid",
	}
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("gitee auth login --agent-help failed: %v\noutput:\n%s", err, out)
	}
	if strings.Contains(string(out), "No token found") {
		t.Errorf("auth login --agent-help triggered the real login flow:\n%s", out)
	}
}

// TestAuthCmdWiring sanity-checks all 6 auth subcommands are
// registered, so a future refactor that drops one fails fast here
// rather than silently breaking the agent-help matrix.
func TestAuthCmdWiring(t *testing.T) {
	if AuthCmd == nil {
		t.Fatal("AuthCmd is nil")
	}
	got := map[string]bool{}
	for _, c := range AuthCmd.Commands() {
		got[c.Name()] = true
	}
	for _, want := range []string{"login", "logout", "refresh", "status", "switch", "token"} {
		if !got[want] {
			t.Errorf("AuthCmd missing subcommand %q", want)
		}
	}
}

// buildThrowawayBinary builds the gitee binary into a temp dir so
// subprocess tests don't depend on the user's $GOPATH/bin install.
// t.Cleanup ensures the file is removed even on test failure.
func buildThrowawayBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gitee-test")
	cmd := exec.Command("go", "build", "-o", bin, "../../cmd/gitee")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build throwaway gitee: %v\n%s", err, out)
	}
	return bin
}
