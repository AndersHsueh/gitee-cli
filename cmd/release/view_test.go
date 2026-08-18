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
		"Race condition warning ⚠️",
		"metadata and asset list returned here are a snapshot",
		"OAuth scope required",
		"Rate limit",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(releaseViewAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
