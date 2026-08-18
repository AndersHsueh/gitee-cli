package label

import (
	"strings"
	"testing"
)

func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee label view",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"renamed or deleted",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(labelViewAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
