package release

import (
	"strings"
	"testing"
)

func TestEditAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee release edit",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(releaseEditAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
