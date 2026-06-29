package pr

import (
	"strings"
	"testing"
)

// TestReviewAgentHelpContent verifies the review verb's agent-targeted
// help string contains all the standard sections.
func TestReviewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr review",
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
		if !strings.Contains(prReviewAgentHelp, want) {
			t.Errorf("pr review --agent-help missing section: %q", want)
		}
	}
}

func TestReviewAgentHelpFlags(t *testing.T) {
	mustMention := []string{
		"--owner",
		"--repo",
		"--approve",
		"--request-changes",
		"--comment",
		"--body",
	}
	for _, flag := range mustMention {
		if !strings.Contains(prReviewAgentHelp, flag) {
			t.Errorf("pr review --agent-help missing flag %q", flag)
		}
	}
}
