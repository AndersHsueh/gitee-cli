package label

import (
	"strings"
	"testing"
)

func TestCreateAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee label create",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(labelCreateAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
