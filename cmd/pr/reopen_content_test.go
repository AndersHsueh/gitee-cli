package pr

import (
	"strings"
	"testing"
)

// TestReopenAgentHelpContent verifies the reopen verb's agent-targeted
// help string contains all the standard sections.
func TestReopenAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr reopen",
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
		if !strings.Contains(prReopenAgentHelp, want) {
			t.Errorf("pr reopen --agent-help missing section: %q", want)
		}
	}
}

func TestReopenAgentHelpFlags(t *testing.T) {
	mustMention := []string{
		"--owner",
		"--repo",
	}
	for _, want := range mustMention {
		if !strings.Contains(prReopenAgentHelp, want) {
			t.Errorf("pr reopen --agent-help missing flag %q", want)
		}
	}
}

// TestReopenAgentHelpRaceCondition asserts the race-condition section
// is command-specific: the head may have been force-pushed while the
// PR was closed, so the diff must be re-inspected after reopen.
func TestReopenAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"force-pushed",
		"already open",
		"gitee pr diff",
	}
	for _, want := range mustMention {
		if !strings.Contains(prReopenAgentHelp, want) {
			t.Errorf("pr reopen --agent-help race section missing: %q", want)
		}
	}
}
