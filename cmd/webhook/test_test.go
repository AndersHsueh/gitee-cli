package webhook

import (
	"strings"
	"testing"
)

func TestTestAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee webhook test",
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
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(webhookTestAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
