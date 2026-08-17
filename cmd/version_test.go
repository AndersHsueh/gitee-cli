package cmd

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionAgentHelpIsNA verifies `gitee version --agent-help` prints
// the operational/config n/a notice to stderr and exits 0, instead of
// dumping the real version string (`gitee version 0.12.6`) into stdout.
// The version verb is operational: agents that probe for documentation
// have no use for a literal version number, and the dump is a trap
// (token-wasting, plus inconsistent with how every other operational
// command responds to --agent-help).
//
// Locks the IK923X step-1 contract for the version command.
func TestVersionAgentHelpIsNA(t *testing.T) {
	bin := buildThrowawayBinary(t)

	run := exec.Command(bin, "version", "--agent-help")
	run.Env = []string{
		"HOME=" + t.TempDir(),
		"PATH=/usr/bin:/bin",
		"GITEE_TOKEN=",
		"GITEE_HOSTNAME=invalid.example.invalid",
	}
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("gitee version --agent-help failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(string(out), "operational/config command") {
		t.Errorf("expected n/a notice on stderr, got:\n%s", out)
	}
	if !strings.Contains(string(out), "Use --help for usage") {
		t.Errorf("expected notice to redirect to --help, got:\n%s", out)
	}
}

// TestVersionAgentHelpDoesNotDumpVersion makes the regression explicit:
// `gitee version --agent-help` MUST NOT print the "gitee version <semver>"
// line that the real Run emits. If it does, the PreRunE hook has been
// bypassed and we're back to the version-leak trap.
//
// We assert the absence of the literal "gitee version " prefix (the real
// Run prints it) — this is intentionally substring-based, not
// "0.12.6"-specific, so the test stays correct across future releases.
func TestVersionAgentHelpDoesNotDumpVersion(t *testing.T) {
	bin := buildThrowawayBinary(t)

	run := exec.Command(bin, "version", "--agent-help")
	run.Env = []string{
		"HOME=" + t.TempDir(),
		"PATH=/usr/bin:/bin",
		"GITEE_TOKEN=",
		"GITEE_HOSTNAME=invalid.example.invalid",
	}
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("gitee version --agent-help failed: %v\noutput:\n%s", err, out)
	}
	if strings.Contains(string(out), "gitee version ") {
		t.Errorf("version --agent-help leaked the real version string:\n%s", out)
	}
}

// TestVersionCmdWiring sanity-checks version is registered on root, so a
// future refactor that drops it fails fast here rather than silently
// breaking the agent-help matrix.
func TestVersionCmdWiring(t *testing.T) {
	if versionCmd == nil {
		t.Fatal("versionCmd is nil")
	}
	if versionCmd.PreRunE == nil {
		t.Error("versionCmd has no PreRunE — operational/config n/a notice would not fire on --agent-help")
	}
	if versionCmd.Run == nil {
		t.Error("versionCmd has no Run — the real version output is unreachable")
	}
	found := false
	for _, c := range RootCommand().Commands() {
		if c.Name() == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("version not registered on root command")
	}
}

// buildThrowawayBinary builds the gitee binary into a temp dir so
// subprocess tests don't depend on the user's $GOPATH/bin install.
// t.Cleanup ensures the file is removed even on test failure.
//
// Mirrors the same helper that lives in cmd/auth/agent_help_test.go
// and cmd/config/agent_help_test.go — kept per-package on purpose
// (no shared test util package in this repo) so each subcommand can
// grow its own coverage without a cross-cutting refactor.
//
// The build path is `./gitee` (the main package, at cmd/gitee/main.go)
// because this test lives in the cmd/ package and go test's cwd is the
// package directory.
func buildThrowawayBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gitee-test")
	c := exec.Command("go", "build", "-o", bin, "./gitee")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("build throwaway gitee: %v\n%s", err, out)
	}
	return bin
}
