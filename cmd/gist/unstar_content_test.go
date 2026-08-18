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
		"## Race condition warning ⚠️",
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

// TestUnstarAgentHelpRaceSection verifies the Race condition section is
// command-specific and free of placeholder text.
func TestUnstarAgentHelpRaceSection(t *testing.T) {
	if strings.Contains(gistUnstarAgentHelp, "TODO(domain)") {
		t.Error("gist unstar --agent-help still contains TODO(domain) placeholder")
	}
	if !strings.Contains(gistUnstarAgentHelp, "treat it as success") {
		t.Error("gist unstar race section should state 404 on an already-gone star is success")
	}
}
