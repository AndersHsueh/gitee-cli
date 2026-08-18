package release

import (
	"strings"
	"testing"
)

func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee release list",
		"Purpose",
		"Pre-conditions",
		"Required flags",
		"Idempotency",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"snapshot of the repo's release set at fetch time",
		"OAuth scope required",
		"Rate limit",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(releaseListAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
