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
		"## Race condition warning ⚠️",
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

// TestMergeAgentHelpRaceCondition asserts the race-condition section is
// command-specific: mergeability flips between check and merge (head
// force-pushed, base advanced), 422 is the API protecting you, and an
// ambiguous failure must be resolved with a fresh view before retry.
func TestMergeAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"force-pushed",
		"do NOT blind-retry",
		".merged",
		"serialize merges",
	}
	for _, want := range mustMention {
		if !strings.Contains(prMergeAgentHelp, want) {
			t.Errorf("pr merge --agent-help race section missing: %q", want)
		}
	}
}
