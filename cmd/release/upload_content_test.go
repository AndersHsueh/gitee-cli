package release

import (
	"strings"
	"testing"
)

// TestUploadAgentHelpContent guards the agent-help contract for
// `gitee release upload` (11-dimension schema, IK923X step 4).
func TestUploadAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee release upload",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"no server-side dedup by filename",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(releaseUploadAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
