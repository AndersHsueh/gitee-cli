package pr

import (
	"strings"
	"testing"

	"gitee-cli/cmd"
)

// TestCommentsAgentHelpContent verifies the comments verb's
// agent-targeted help string contains all the standard sections.
func TestCommentsAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr comments",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
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
		if !strings.Contains(prCommentsAgentHelp, want) {
			t.Errorf("pr comments --agent-help missing section: %q", want)
		}
	}
}

func TestCommentsAgentHelpFlags(t *testing.T) {
	mustMention := []string{
		"--owner",
		"--repo",
	}
	for _, want := range mustMention {
		if !strings.Contains(prCommentsAgentHelp, want) {
			t.Errorf("pr comments --agent-help missing flag %q", want)
		}
	}
}

// TestPRCommentsArgsBypassWithAgentHelp verifies the
// AgentHelpAwareArgs wrapper on cobra.ExactArgs(1) lets
// `gitee pr comments --agent-help` reach PreRunE without the
// required <number> positional.
func TestPRCommentsArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	if err := commentsCmd.Args(commentsCmd, nil); err != nil {
		t.Errorf("pr comments --agent-help with no positional should pass Args, got: %v", err)
	}
}

// TestCommentsAgentHelpRaceCondition asserts the race-condition section
// is command-specific: the conversation thread is the shared mutable
// state, and the write side (gitee pr comment) is non-idempotent.
func TestCommentsAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"conversation thread",
		"NOT idempotent",
		"re-list",
	}
	for _, want := range mustMention {
		if !strings.Contains(prCommentsAgentHelp, want) {
			t.Errorf("pr comments --agent-help race section missing: %q", want)
		}
	}
}
