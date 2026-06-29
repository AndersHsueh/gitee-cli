package issue

import (
	"strings"
	"testing"
)

// TestCommentAgentHelpContent verifies the comment verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee issue comment --agent-help` actually prints.
func TestCommentAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee issue comment",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Next steps",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(issueCommentAgentHelp, want) {
			t.Errorf("issue comment --agent-help missing section: %q", want)
		}
	}
}
