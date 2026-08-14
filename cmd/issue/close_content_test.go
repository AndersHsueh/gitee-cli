package issue

import (
	"strings"
	"testing"
)

// TestCloseAgentHelpContent verifies the close verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee issue close --agent-help` actually prints.
func TestCloseAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee issue close",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Next steps",
		"## OAuth scope required",
		"## Rate limit",
		"## Implementation notes (Gitee v5 quirk)",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(issueCloseAgentHelp, want) {
			t.Errorf("issue close --agent-help missing section: %q", want)
		}
	}
}
