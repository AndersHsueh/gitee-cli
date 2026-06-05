package pr

import (
	"strings"
	"testing"
)

func TestEditAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr edit",
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
		"Next steps",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(prEditAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
