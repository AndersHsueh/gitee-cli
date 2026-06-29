package issue

import (
	"strings"
	"testing"
)

// TestListAgentHelpContent verifies the list verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee issue list --agent-help` actually prints.
func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee issue list",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Next steps",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(issueListAgentHelp, want) {
			t.Errorf("issue list --agent-help missing section: %q", want)
		}
	}
}
