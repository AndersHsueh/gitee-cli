package label

import (
	"strings"
	"testing"
)

func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee label list",
		"Purpose",
		"Pre-conditions",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"point-in-time snapshot",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(labelListAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
