package label

import (
	"strings"
	"testing"
)

func TestEditAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee label edit",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(labelEditAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
