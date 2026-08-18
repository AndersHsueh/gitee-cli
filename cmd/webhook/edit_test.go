package webhook

import (
	"strings"
	"testing"
)

func TestEditAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee webhook edit",
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
		"Race condition warning ⚠️",
		"Next steps",
		"OAuth scope required",
		"Rate limit",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(webhookEditAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
