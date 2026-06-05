package pr

import (
	"strings"
	"testing"
)

func TestChecksAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr checks",
		"Purpose",
		"Pre-conditions",
		"Required arguments",
		"Optional flags",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition",
		"Next steps",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(prChecksAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
