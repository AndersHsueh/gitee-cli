package gist

import (
	"strings"
	"testing"
)

// TestEditAgentHelpContent verifies the edit verb's agent-targeted help
// string contains all the standard sections.
func TestEditAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist edit",
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
		if !strings.Contains(gistEditAgentHelp, want) {
			t.Errorf("gist edit --agent-help missing section: %q", want)
		}
	}
}

// TestEditAgentHelpFlagsPresent ensures every flag from the audit is
// mentioned in the markdown.
func TestEditAgentHelpFlagsPresent(t *testing.T) {
	required := []string{
		"--description",
		"--file",
	}
	for _, want := range required {
		if !strings.Contains(gistEditAgentHelp, want) {
			t.Errorf("gist edit --agent-help missing flag: %q", want)
		}
	}
}

// TestEditAgentHelpRaceSection verifies the Race condition section is
// command-specific (last-writer-wins) and free of placeholder text.
func TestEditAgentHelpRaceSection(t *testing.T) {
	if strings.Contains(gistEditAgentHelp, "TODO(domain)") {
		t.Error("gist edit --agent-help still contains TODO(domain) placeholder")
	}
	if !strings.Contains(gistEditAgentHelp, "LAST-WRITER-WINS") {
		t.Error("gist edit race section should call out last-writer-wins semantics")
	}
}
