package pr

import (
	"strings"
	"testing"
)

// TestListAgentHelpContent verifies the list verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc in
// sync with what `gitee pr list --agent-help` actually prints.
func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr list",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(prListAgentHelp, want) {
			t.Errorf("pr list --agent-help missing section: %q", want)
		}
	}
}

// TestListAgentHelpFlags asserts every flag the command supports is
// documented in its agent-help body. A flag is "documented" if its
// long form appears in the help text.
func TestListAgentHelpFlags(t *testing.T) {
	mustMention := []string{
		"--owner",
		"--repo",
		"--state",
		"--head",
		"--base",
	}
	for _, flag := range mustMention {
		if !strings.Contains(prListAgentHelp, flag) {
			t.Errorf("pr list --agent-help missing flag %q", flag)
		}
	}
}

// TestListAgentHelpRaceCondition asserts the race-condition section is
// command-specific: the list is a snapshot, --paginate page boundaries
// can shift mid-walk, and writes must re-verify with a fresh view.
func TestListAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"point-in-time snapshot",
		"page boundaries",
		"gitee pr view",
	}
	for _, want := range mustMention {
		if !strings.Contains(prListAgentHelp, want) {
			t.Errorf("pr list --agent-help race section missing: %q", want)
		}
	}
}
