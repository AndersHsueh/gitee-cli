package gist

import (
	"strings"
	"testing"
)

// TestCreateAgentHelpContent verifies the create verb's agent-targeted
// help string contains all the standard sections.
func TestCreateAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist create",
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
		if !strings.Contains(gistCreateAgentHelp, want) {
			t.Errorf("gist create --agent-help missing section: %q", want)
		}
	}
}

// TestCreateAgentHelpFlagsPresent ensures every flag from the audit is
// mentioned in the markdown.
func TestCreateAgentHelpFlagsPresent(t *testing.T) {
	required := []string{
		"--description",
		"--public",
		"--filename",
		"--file",
		"--interactive",
		"-i",
	}
	for _, want := range required {
		if !strings.Contains(gistCreateAgentHelp, want) {
			t.Errorf("gist create --agent-help missing flag: %q", want)
		}
	}
}
