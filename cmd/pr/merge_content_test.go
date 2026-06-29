package pr

import (
	"strings"
	"testing"
)

// TestMergeAgentHelpContent verifies the merge verb's agent-targeted
// help string contains all the standard sections.
func TestMergeAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr merge",
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
		if !strings.Contains(prMergeAgentHelp, want) {
			t.Errorf("pr merge --agent-help missing section: %q", want)
		}
	}
}

func TestMergeAgentHelpFlags(t *testing.T) {
	mustMention := []string{
		"--owner",
		"--repo",
		"--squash",
		"--squash-message",
	}
	for _, flag := range mustMention {
		if !strings.Contains(prMergeAgentHelp, flag) {
			t.Errorf("pr merge --agent-help missing flag %q", flag)
		}
	}
}
