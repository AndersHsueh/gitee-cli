package pr

import (
	"strings"
	"testing"
)

// TestCommentsAgentHelpContent verifies the comments verb's
// agent-targeted help string contains all the standard sections.
func TestCommentsAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr comments",
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
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(prCommentsAgentHelp, want) {
			t.Errorf("pr comments --agent-help missing section: %q", want)
		}
	}
}

func TestCommentsAgentHelpFlags(t *testing.T) {
	mustMention := []string{
		"--owner",
		"--repo",
	}
	for _, want := range mustMention {
		if !strings.Contains(prCommentsAgentHelp, want) {
			t.Errorf("pr comments --agent-help missing flag %q", want)
		}
	}
}
