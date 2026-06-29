package search

import (
	"strings"
	"testing"
)

// TestUsersAgentHelpContent verifies the users verb's agent-targeted
// help string contains all the standard sections. If a section is
// dropped during a refactor, this test fails immediately — keeping
// the doc in sync with what `gitee search users --agent-help` prints.
func TestUsersAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee search users",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
		// Every flag listed in the audit must appear in the markdown body.
		"--limit / -L",
	}
	for _, want := range required {
		if !strings.Contains(searchUsersAgentHelp, want) {
			t.Errorf("search users --agent-help missing section/flag: %q", want)
		}
	}
}
