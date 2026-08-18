package pr

import (
	"strings"
	"testing"
)

func TestReadyAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr ready",
		"Purpose",
		"Pre-conditions",
		"Required arguments",
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
		if !strings.Contains(prReadyAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

// TestReadyAgentHelpRaceCondition asserts the race-condition section is
// command-specific: the draft flag can be flipped back concurrently
// (re-drafted), so a follow-up merge must re-verify on fresh output.
func TestReadyAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"re-drafts",
		".draft",
		"gitee pr view",
	}
	for _, want := range mustMention {
		if !strings.Contains(prReadyAgentHelp, want) {
			t.Errorf("agent help race section missing: %q", want)
		}
	}
}
