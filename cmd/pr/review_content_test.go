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
		"## Race condition warning ⚠️",
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

// TestReviewAgentHelpRaceCondition asserts the race-condition section
// is command-specific: a force-push between diff-read and --approve
// means approving unseen code, and a retry can post a duplicate review.
func TestReviewAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"force-pushed",
		"duplicate review",
		"head SHA",
	}
	for _, want := range mustMention {
		if !strings.Contains(prReviewAgentHelp, want) {
			t.Errorf("pr review --agent-help race section missing: %q", want)
		}
	}
}
