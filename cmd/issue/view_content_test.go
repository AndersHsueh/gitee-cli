package issue

import (
	"strings"
	"testing"
)

// TestViewAgentHelpContent verifies the view verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee issue view --agent-help` actually prints.
func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee issue view",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, text mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Next steps",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(issueViewAgentHelp, want) {
			t.Errorf("issue view --agent-help missing section: %q", want)
		}
	}
}
