package gist

import (
	"strings"
	"testing"
)

// TestUnstarAgentHelpContent verifies the unstar verb's agent-targeted
// help string contains all the standard sections.
func TestUnstarAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist unstar",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Common errors",
		"## Next steps",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(gistUnstarAgentHelp, want) {
			t.Errorf("gist unstar --agent-help missing section: %q", want)
		}
	}
}

// TestUnstarAgentHelpFlagsPresent ensures no flags are referenced
// (unstar has no flags per the audit).
func TestUnstarAgentHelpFlagsPresent(t *testing.T) {
	for _, banned := range []string{"--limit", "--description", "--public", "--file", "--raw"} {
		if strings.Contains(gistUnstarAgentHelp, banned) {
			t.Errorf("gist unstar --agent-help should not mention %q (unstar has no flags)", banned)
		}
	}
}
