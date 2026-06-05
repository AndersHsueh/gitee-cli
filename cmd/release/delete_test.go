package release

import (
	"strings"
	"testing"
)

func TestDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee release delete",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(releaseDeleteAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
