package gist

import (
	"strings"
	"testing"
)

// TestDeleteAgentHelpContent verifies the delete verb's agent-targeted
// help string contains all the standard sections. The destructive
// guidance (Atomicity + Side effects) is required.
func TestDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist delete",
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
		if !strings.Contains(gistDeleteAgentHelp, want) {
			t.Errorf("gist delete --agent-help missing section: %q", want)
		}
	}
}

// TestDeleteAgentHelpFlagsPresent ensures no flags are referenced
// (delete has no flags per the audit).
func TestDeleteAgentHelpFlagsPresent(t *testing.T) {
	// delete has no flags — sanity check that we don't accidentally
	// advertise ones that don't exist.
	for _, banned := range []string{"--limit", "--description", "--public", "--file", "--raw"} {
		if strings.Contains(gistDeleteAgentHelp, banned) {
			t.Errorf("gist delete --agent-help should not mention %q (delete has no flags)", banned)
		}
	}
}

// TestDeleteAgentHelpMentionsIrreversibility is a safety check: the
// destructive command's help MUST call out the irreversible nature so
// agents don't blow away data without warning.
func TestDeleteAgentHelpMentionsIrreversibility(t *testing.T) {
	if !strings.Contains(gistDeleteAgentHelp, "IRREVERSIBLE") {
		t.Error("gist delete --agent-help must call out IRREVERSIBLE action (safety contract)")
	}
	if !strings.Contains(gistDeleteAgentHelp, "no undo") && !strings.Contains(gistDeleteAgentHelp, "NO UNDO") {
		t.Error("gist delete --agent-help must explicitly mention that there is no undo")
	}
}

// TestDeleteAgentHelpRaceSection verifies the Race condition section is
// command-specific and free of placeholder text.
func TestDeleteAgentHelpRaceSection(t *testing.T) {
	if strings.Contains(gistDeleteAgentHelp, "TODO(domain)") {
		t.Error("gist delete --agent-help still contains TODO(domain) placeholder")
	}
	if !strings.Contains(gistDeleteAgentHelp, "retrying a delete is safe") {
		t.Error("gist delete race section should state concurrent-delete 404 is the desired end state")
	}
}
