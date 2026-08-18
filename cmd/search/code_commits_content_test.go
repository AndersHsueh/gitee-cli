package search

import (
	"strings"
	"testing"
)

// TestCodeAgentHelpContent verifies the code verb's agent-targeted help
// string contains all the standard sections, including the race-condition
// warning added for the 11-dimension schema. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc in
// sync with what `gitee search code --agent-help` actually prints.
func TestCodeAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee search code",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
		// Every flag listed in the audit must appear in the markdown body.
		"--language",
		"--owner",
		"--sort",
		"--order",
		"--limit / -L",
	}
	for _, want := range required {
		if !strings.Contains(searchCodeAgentHelp, want) {
			t.Errorf("search code --agent-help missing section/flag: %q", want)
		}
	}
	// Spot-check the race section carries command-specific guidance:
	// a hit is only valid at index time; re-fetch before acting.
	if !strings.Contains(searchCodeAgentHelp, "re-fetch the file content immediately before") {
		t.Error("search code race section missing re-fetch-before-acting guidance")
	}
}

// TestCommitsAgentHelpContent verifies the commits verb's agent-targeted
// help string contains all the standard sections, including the
// race-condition warning added for the 11-dimension schema.
func TestCommitsAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee search commits",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
		"--limit / -L",
	}
	for _, want := range required {
		if !strings.Contains(searchCommitsAgentHelp, want) {
			t.Errorf("search commits --agent-help missing section/flag: %q", want)
		}
	}
	// Spot-check the race section covers the indexing-lag hazard:
	// do not re-search to confirm a commit you just created.
	if !strings.Contains(searchCommitsAgentHelp, "do not re-search to confirm a commit you just") {
		t.Error("search commits race section missing indexing-lag guidance")
	}
}
