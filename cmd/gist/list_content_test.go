package gist

import (
	"strings"
	"testing"
)

// TestListAgentHelpContent verifies the list verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee gist list --agent-help` actually prints.
func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist list",
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
		"## Race condition warning ⚠️",
		"## Next steps",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(gistListAgentHelp, want) {
			t.Errorf("gist list --agent-help missing section: %q", want)
		}
	}
}

// TestListAgentHelpFlagsPresent ensures every flag advertised in the
// audit (cmd.GlobalAgentHelp audit.missing_commands.flags) is mentioned
// in the markdown. Catches the "added the flag but forgot to document
// it in --agent-help" mistake that would otherwise only be noticed
// by an agent reading the doc.
func TestListAgentHelpFlagsPresent(t *testing.T) {
	required := []string{
		"--limit",
		"-L",
	}
	for _, want := range required {
		if !strings.Contains(gistListAgentHelp, want) {
			t.Errorf("gist list --agent-help missing flag: %q", want)
		}
	}
}

// TestListAgentHelpRaceSection verifies the Race condition section is
// command-specific and free of placeholder text.
func TestListAgentHelpRaceSection(t *testing.T) {
	if strings.Contains(gistListAgentHelp, "TODO(domain)") {
		t.Error("gist list --agent-help still contains TODO(domain) placeholder")
	}
	if !strings.Contains(gistListAgentHelp, "indexing lag") {
		t.Error("gist list race section should mention indexing lag for fresh gists")
	}
}
