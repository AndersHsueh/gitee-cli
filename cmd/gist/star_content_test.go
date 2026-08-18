package gist

import (
	"strings"
	"testing"
)

// TestStarAgentHelpContent verifies the star verb's agent-targeted
// help string contains all the standard sections.
func TestStarAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist star",
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
		if !strings.Contains(gistStarAgentHelp, want) {
			t.Errorf("gist star --agent-help missing section: %q", want)
		}
	}
}

// TestStarAgentHelpFlagsPresent ensures no flags are referenced
// (star has no flags per the audit).
func TestStarAgentHelpFlagsPresent(t *testing.T) {
	for _, banned := range []string{"--limit", "--description", "--public", "--file", "--raw"} {
		if strings.Contains(gistStarAgentHelp, banned) {
			t.Errorf("gist star --agent-help should not mention %q (star has no flags)", banned)
		}
	}
}

// TestStarAgentHelpRaceSection verifies the Race condition section is
// command-specific and free of placeholder text.
func TestStarAgentHelpRaceSection(t *testing.T) {
	if strings.Contains(gistStarAgentHelp, "TODO(domain)") {
		t.Error("gist star --agent-help still contains TODO(domain) placeholder")
	}
	if !strings.Contains(gistStarAgentHelp, "already-starred") {
		t.Error("gist star race section should state retrying is a no-op on an already-starred gist")
	}
}
