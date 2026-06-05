package webhook

import (
	"strings"
	"testing"
)

// TestCreateAgentHelpContent verifies that the agent help text contains
// every section promised in the documentation. This is the static contract
// that agents rely on: if a section is removed from the text, this test
// fails and forces the maintainer to update the docs.
func TestCreateAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee webhook create",
		"Purpose",
		"Pre-conditions",
		"Required flags",
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
		if !strings.Contains(webhookCreateAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
