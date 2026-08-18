package label

import (
	"strings"
	"testing"
)

func TestCreateAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee label create",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"Do NOT",
		"re-list to verify creation",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(labelCreateAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
