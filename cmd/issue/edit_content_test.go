package issue

import (
	"strings"
	"testing"
)

// TestEditAgentHelpContent verifies the edit verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee issue edit --agent-help` actually prints.
func TestEditAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee issue edit",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"last-writer-wins",
		"## Next steps",
		"## OAuth scope required",
		"## Rate limit",
		"## Implementation notes (Gitee v5 quirk)",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(issueEditAgentHelp, want) {
			t.Errorf("issue edit --agent-help missing section: %q", want)
		}
	}
}
