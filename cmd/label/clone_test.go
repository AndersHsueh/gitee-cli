package label

import (
	"strings"
	"testing"
)

func TestCloneAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee label clone",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"Two agents cloning into the same target concurrently",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(labelCloneAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
