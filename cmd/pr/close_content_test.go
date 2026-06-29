package pr

import (
	"strings"
	"testing"
)

// TestCloseAgentHelpContent verifies the close verb's agent-targeted
// help string contains all the standard sections.
func TestCloseAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr close",
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
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(prCloseAgentHelp, want) {
			t.Errorf("pr close --agent-help missing section: %q", want)
		}
	}
}

// TestCloseAgentHelpReversibility asserts the help text explicitly
// calls out that the action is reversible. Close is destructive-ish,
// and an agent must know how to undo it.
func TestCloseAgentHelpReversibility(t *testing.T) {
	mustMention := []string{
		"reopen",
		"reversible",
		"--owner",
		"--repo",
	}
	for _, want := range mustMention {
		if !strings.Contains(prCloseAgentHelp, want) {
			t.Errorf("pr close --agent-help missing reversibility hint: %q", want)
		}
	}
}
