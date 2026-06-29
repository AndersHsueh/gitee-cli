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
