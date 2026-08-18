package gist

import (
	"strings"
	"testing"
)

// TestViewAgentHelpContent verifies the view verb's agent-targeted help
// string contains all the standard sections.
func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist view",
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
		"## Race condition warning ⚠️",
		"## Next steps",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(gistViewAgentHelp, want) {
			t.Errorf("gist view --agent-help missing section: %q", want)
		}
	}
}

// TestViewAgentHelpFlagsPresent ensures every flag from the audit is
// mentioned in the markdown.
func TestViewAgentHelpFlagsPresent(t *testing.T) {
	required := []string{
		"--raw",
	}
	for _, want := range required {
		if !strings.Contains(gistViewAgentHelp, want) {
			t.Errorf("gist view --agent-help missing flag: %q", want)
		}
	}
}

// TestViewAgentHelpRaceSection verifies the Race condition section is
// command-specific and free of placeholder text.
func TestViewAgentHelpRaceSection(t *testing.T) {
	if strings.Contains(gistViewAgentHelp, "TODO(domain)") {
		t.Error("gist view --agent-help still contains TODO(domain) placeholder")
	}
	if !strings.Contains(gistViewAgentHelp, "raw_url") {
		t.Error("gist view race section should mention the per-file raw_url fetch gap")
	}
}
