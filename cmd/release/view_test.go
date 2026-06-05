package release

import (
	"strings"
	"testing"
)

func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee release view",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(releaseViewAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
