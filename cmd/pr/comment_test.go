package pr

import (
	"strings"
	"testing"
)

func TestCommentAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr comment",
		"Purpose",
		"Pre-conditions",
		"Required arguments",
		"Required flags",
		"Optional flags",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(prCommentAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

// TestCommentAgentHelpRaceCondition asserts the race-condition section
// is command-specific: the command is NOT idempotent (a retry after an
// ambiguous failure posts a duplicate comment) and concurrent posters
// interleave in the conversation thread.
func TestCommentAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"duplicate",
		"gitee pr comments",
		"re-list",
	}
	for _, want := range mustMention {
		if !strings.Contains(prCommentAgentHelp, want) {
			t.Errorf("agent help race section missing: %q", want)
		}
	}
}
