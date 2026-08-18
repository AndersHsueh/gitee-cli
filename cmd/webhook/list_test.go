package webhook

import (
	"strings"
	"testing"
)

func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee webhook list",
		"Purpose",
		"Pre-conditions",
		"Required flags",
		"Optional flags",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"OAuth scope required",
		"Rate limit",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(webhookListAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
