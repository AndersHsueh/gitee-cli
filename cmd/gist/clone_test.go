package gist

import (
	"strings"
	"testing"
)

func TestCloneAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist clone",
		"Purpose",
		"Pre-conditions",
		"Required arguments",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"OAuth scope required",
		"Rate limit",
		"Next steps",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(gistCloneAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
