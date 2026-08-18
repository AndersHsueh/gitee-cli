package release

import (
	"strings"
	"testing"
)

// TestDeleteAssetAgentHelpContent guards the agent-help contract for
// `gitee release delete-asset` (11-dimension schema, IK923X step 4).
func TestDeleteAssetAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee release delete-asset",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"the release's asset list",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(releaseDeleteAssetAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
