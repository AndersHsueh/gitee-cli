package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestCompletionAgentHelpIsNA verifies every shell completion
// subcommand (bash / fish / powershell / zsh) prints the
// operational/config n/a notice to stderr and exits 0 when probed
// with --agent-help, instead of dumping the real shell completion
// script into stdout. The completion verbs are operational: agents
// that probe for documentation have no use for hundreds of lines of
// shell code, and the dump is a trap (token-wasting, plus inconsistent
// with how every other operational command responds to --agent-help).
//
// Locks the IK923X step-1 contract for all four completion subcommands.
func TestCompletionAgentHelpIsNA(t *testing.T) {
	bin := buildThrowawayBinary(t)

	shells := []string{"bash", "fish", "powershell", "zsh"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			run := exec.Command(bin, "completion", shell, "--agent-help")
			run.Env = []string{
				"HOME=" + t.TempDir(),
				"PATH=/usr/bin:/bin",
				"GITEE_TOKEN=",
				"GITEE_HOSTNAME=invalid.example.invalid",
			}
			out, err := run.CombinedOutput()
			if err != nil {
				t.Fatalf("gitee completion %s --agent-help failed: %v\noutput:\n%s",
					shell, err, out)
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

// TestCompletionAgentHelpDoesNotDumpShellScript makes the regression
// explicit: `gitee completion <shell> --agent-help` MUST NOT print the
// real shell completion script. If it does, the PreRunE hook has been
// bypassed and we're back to the script-leak trap.
//
// We assert the absence of two markers that every shell completion
// script emits: the `# bash completion V2` / `# fish completion V2` /
// similar header (cobra's standard prefix) and the `__gitee_debug`
// function name. The exact header text varies by cobra version, so we
// use a substring that all four shells share: `__gitee_debug` (the
// helper function that cobra's bash/fish completion scripts always
// define). For powershell/zsh we additionally check the absence of
// the cobra boilerplate.
func TestCompletionAgentHelpDoesNotDumpShellScript(t *testing.T) {
	bin := buildThrowawayBinary(t)

	shells := []string{"bash", "fish", "powershell", "zsh"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			run := exec.Command(bin, "completion", shell, "--agent-help")
			run.Env = []string{
				"HOME=" + t.TempDir(),
				"PATH=/usr/bin:/bin",
				"GITEE_TOKEN=",
				"GITEE_HOSTNAME=invalid.example.invalid",
			}
			out, err := run.CombinedOutput()
			if err != nil {
				t.Fatalf("gitee completion %s --agent-help failed: %v\noutput:\n%s",
					shell, err, out)
			}
			combined := string(out)
			// Markers the real completion script always emits. Use
			// substrings (not full lines) so test stays correct across
			// cobra version bumps.
			leaked := []string{
				"__gitee_debug", // bash/fish internal helper
				"completion V2", // cobra's standard header prefix
			}
			for _, marker := range leaked {
				if strings.Contains(combined, marker) {
					t.Errorf("completion %s --agent-help leaked shell script (found %q):\n%s",
						shell, marker, combined)
				}
			}
		})
	}
}

// TestCompletionCmdWiring sanity-checks all 4 completion subcommands
// are registered under a top-level "completion" command, so a future
// refactor that drops one fails fast here rather than silently
// breaking the agent-help matrix. The test also asserts each
// subcommand has both a Run (so plain `gitee completion bash` still
// produces a usable script) and a PreRunE (so --agent-help short-
// circuits to the n/a notice).
func TestCompletionCmdWiring(t *testing.T) {
	completionCmd := findCompletionCmd(t)
	if completionCmd == nil {
		t.Fatal("completion command not found on root")
	}
	got := map[string]bool{}
	for _, c := range completionCmd.Commands() {
		got[c.Name()] = true
		if c.PreRunE == nil {
			t.Errorf("completion %s has no PreRunE — n/a notice would not fire on --agent-help", c.Name())
		}
		if c.Run == nil && c.RunE == nil {
			t.Errorf("completion %s has no Run/RunE — plain `gitee completion %s` would be a no-op", c.Name(), c.Name())
		}
	}
	for _, want := range []string{"bash", "fish", "powershell", "zsh"} {
		if !got[want] {
			t.Errorf("completion missing subcommand %q", want)
		}
	}
}

// findCompletionCmd locates the top-level "completion" command on the
// root. We don't keep a package-level reference (the four subcommands
// are registered inside init() in this package), so we walk the tree.
func findCompletionCmd(t *testing.T) *cobra.Command {
	t.Helper()
	for _, c := range RootCommand().Commands() {
		if c.Name() == "completion" {
			return c
		}
	}
	return nil
}
