package label

import (
	"strings"
	"testing"
)

func TestDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee label delete",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"already gone",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(labelDeleteAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
