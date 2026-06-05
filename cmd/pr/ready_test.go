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
