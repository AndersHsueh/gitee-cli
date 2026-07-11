package pr

import (
	"strings"
	"testing"
)

// TestPRStatusAgentHelpContent verifies the agent-targeted help string
// for `gitee pr status` contains all the standard sections.
func TestPRStatusAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr status",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"--limit / -L",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Common errors",
		"## OAuth scope required",
		"## Rate limit",
		"## Race conditions",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(prStatusAgentHelp, want) {
			t.Errorf("pr status --agent-help missing section: %q", want)
		}
	}
}

// TestPRStatusLimitFlagRegistered verifies that --limit/-L is registered
// on the pr status command. Before this fix, the agent-help text
// documented --limit/-L but init() never registered the flag, so
// `gitee pr status --limit 100` failed with 'unknown flag: --limit'
// and the repo scan was hardcoded to 50. Mirrors issue/status.go:66.
func TestPRStatusLimitFlagRegistered(t *testing.T) {
	resetPRStatusFlags()
	defer resetPRStatusFlags()

	if err := prStatusCmd.ParseFlags([]string{"--limit", "100"}); err != nil {
		t.Fatalf("parsing --limit flag: %v", err)
	}
	if prStatusFlags.limit != 100 {
		t.Errorf("expected prStatusFlags.limit=100 after --limit 100, got %d", prStatusFlags.limit)
	}

	// Verify the short form -L works too.
	resetPRStatusFlags()
	if err := prStatusCmd.ParseFlags([]string{"-L", "25"}); err != nil {
		t.Fatalf("parsing -L flag: %v", err)
	}
	if prStatusFlags.limit != 25 {
		t.Errorf("expected prStatusFlags.limit=25 after -L 25, got %d", prStatusFlags.limit)
	}
}

// resetPRStatusFlags restores the package-level flag state between tests.
func resetPRStatusFlags() {
	prStatusCmd.SetArgs(nil)
	prStatusFlags.limit = 50
}
