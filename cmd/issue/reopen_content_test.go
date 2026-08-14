package issue

import (
	"strings"
	"testing"
)

// TestReopenAgentHelpContent verifies the reopen verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee issue reopen --agent-help` actually prints.
func TestReopenAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee issue reopen",
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
		if !strings.Contains(issueReopenAgentHelp, want) {
			t.Errorf("issue reopen --agent-help missing section: %q", want)
		}
	}
}
