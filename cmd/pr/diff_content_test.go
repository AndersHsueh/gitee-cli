package pr

import (
	"strings"
	"testing"
)

// TestDiffAgentHelpContent verifies the diff verb's agent-targeted
// help string contains all the standard sections.
func TestDiffAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr diff",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(prDiffAgentHelp, want) {
			t.Errorf("pr diff --agent-help missing section: %q", want)
		}
	}
}

func TestDiffAgentHelpFlags(t *testing.T) {
	mustMention := []string{
		"--owner",
		"--repo",
	}
	for _, want := range mustMention {
		if !strings.Contains(prDiffAgentHelp, want) {
			t.Errorf("pr diff --agent-help missing flag %q", want)
		}
	}
}

// TestDiffAgentHelpRaceCondition asserts the race-condition section is
// command-specific: the diff is a snapshot and a force-push between
// diff and approve/merge means reviewing code you never read.
func TestDiffAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"snapshot",
		"force-push",
		"re-fetch",
	}
	for _, want := range mustMention {
		if !strings.Contains(prDiffAgentHelp, want) {
			t.Errorf("pr diff --agent-help race section missing: %q", want)
		}
	}
}
