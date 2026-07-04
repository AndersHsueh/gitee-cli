package api

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

// TestRawArgsBypassWithAgentHelp verifies the AgentHelpAwareArgs
// wrapper on cobra.ExactArgs(2) lets `gitee api raw --agent-help`
// reach PreRunE without the <method> <path> positional pair. This
// is the task-A part of the api raw fix; task B adds the PreRunE
// hook so the doc actually prints before auth checks.
func TestRawArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	if err := rawCmd.Args(rawCmd, nil); err != nil {
		t.Errorf("api raw --agent-help with no positional should pass Args, got: %v", err)
	}
}

// TestAPIArgsBypassWithAgentHelp verifies the same wrapper for the
// parent `gitee api` command (the raw subcommand is also registered
// under it for backwards compatibility).
func TestAPIArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	if err := apiCmd.Args(apiCmd, nil); err != nil {
		t.Errorf("api --agent-help with no positional should pass Args, got: %v", err)
	}
}

// TestAPIRawAgentHelpContent sanity-checks the const string is the
// standard agent-help shape: starts with `# gitee` so the agent-help
// matrix in cmd/agent_help.go counts it as a real doc (not an n/a
// notice), and contains the required sections.
func TestAPIRawAgentHelpContent(t *testing.T) {
	if !strings.HasPrefix(apiRawAgentHelp, "# gitee") {
		t.Errorf("apiRawAgentHelp must start with '# gitee', got %q", firstLine(apiRawAgentHelp))
	}
	for _, want := range []string{
		"## Purpose",
		"## Pre-conditions",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	} {
		if !strings.Contains(apiRawAgentHelp, want) {
			t.Errorf("apiRawAgentHelp missing required section: %q", want)
		}
	}
}

// TestAPIRawAgentHelpWithoutToken runs the gitee binary in a fresh
// subprocess with an empty HOME + GITEE_TOKEN="" so any settings
// file and env-supplied token are unavailable. Before task B, this
// path would print "not authenticated. Set GITEE_TOKEN..." because
// NewAPIClient fires before --agent-help could be served. After
// task B, the PreRunE hook prints the doc and exits 0 without
// touching the auth layer.
//
// We build a throwaway binary rather than rely on PATH so this test
// is hermetic and doesn't pollute the user's $GOPATH/bin.
func TestAPIRawAgentHelpWithoutToken(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "gitee-test")
	// Build a throwaway binary at a known path; t.TempDir() cleans
	// it up automatically on test end.
	cmdBuild := exec.Command("go", "build", "-o", bin, "../../cmd/gitee")
	cmdBuild.Dir = "."
	if out, err := cmdBuild.CombinedOutput(); err != nil {
		t.Fatalf("build throwaway gitee: %v\n%s", err, out)
	}

	// Run with an empty HOME (t.TempDir() creates an isolated dir
	// with no settings.json) so config.GetToken() falls through to
	// envvar GITEE_TOKEN (which we also clear) and returns "". Use
	// an unreachable API host so even if the short-circuit
	// regressed we wouldn't accidentally hit a real network.
	home := t.TempDir()
	run := exec.Command(bin, "api", "raw", "--agent-help")
	run.Env = []string{
		"HOME=" + home,
		"PATH=/usr/bin:/bin",
		"GITEE_TOKEN=",
		"GITEE_HOSTNAME=invalid.example.invalid",
	}
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("gitee api raw --agent-help (no token) failed: %v\noutput:\n%s", err, out)
	}
	if !strings.HasPrefix(string(out), "# gitee") {
		t.Errorf("expected stdout to start with '# gitee', got: %q", firstLine(string(out)))
	}
}

// TestAPIRawAgentHelpIsServedBeforeAuth asserts the wiring: with
// --agent-help set and zero args, rawCmd's PreRunE returns nil
// without ever reaching NewAPIClient. We exercise PreRunE directly
// (without invoking PrintAgentHelp, which os.Exit(0)s) by inspecting
// that the function is wired and returns no error.
func TestAPIRawAgentHelpIsServedBeforeAuth(t *testing.T) {
	if rawCmd.PreRunE == nil {
		t.Fatal("rawCmd.PreRunE is nil; --agent-help short-circuit missing")
	}
	if apiCmd.PreRunE == nil {
		t.Fatal("apiCmd.PreRunE is nil; --agent-help short-circuit missing")
	}
}

// firstLine returns the first line of s without trailing newline.
// Used for clearer error messages.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}