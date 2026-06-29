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
