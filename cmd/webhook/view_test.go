package webhook

import (
	"strings"
	"testing"
)

func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee webhook view",
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
		"## Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(webhookViewAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
