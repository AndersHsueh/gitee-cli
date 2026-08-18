package pr

import (
	"strings"
	"testing"
)

// TestViewAgentHelpContent verifies the view verb's agent-targeted help
// string contains all the standard sections.
func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr view",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(prViewAgentHelp, want) {
			t.Errorf("pr view --agent-help missing section: %q", want)
		}
	}
}

func TestViewAgentHelpFlags(t *testing.T) {
	mustMention := []string{
		"--owner",
		"--repo",
	}
	for _, flag := range mustMention {
		if !strings.Contains(prViewAgentHelp, flag) {
			t.Errorf("pr view --agent-help missing flag %q", flag)
		}
	}
}

// TestViewAgentHelpRaceCondition asserts the race-condition section is
// command-specific: the metadata is a snapshot, and a write chained off
// it must re-verify state on fresh output.
func TestViewAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"point-in-time snapshot",
		"force-pushed",
		"FRESH",
	}
	for _, want := range mustMention {
		if !strings.Contains(prViewAgentHelp, want) {
			t.Errorf("pr view --agent-help race section missing: %q", want)
		}
	}
}
