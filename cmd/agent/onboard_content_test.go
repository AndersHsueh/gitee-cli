package agent

import (
	"strings"
	"testing"
)

// TestOnboardAgentHelpContent verifies the onboard verb's agent-targeted
// help string contains all the standard sections. If a section is
// dropped during a refactor, this test fails immediately — keeping the
// doc in sync with what `gitee agent onboard --agent-help` actually
// prints. Mirrors the structure of cmd/alias/content_test.go.
func TestOnboardAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee agent onboard",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(onboardAgentHelp, want) {
			t.Errorf("agent onboard --agent-help missing section: %q", want)
		}
	}
	// Spot-check the race section: onboard writes a local file, so the
	// hazard is the exists-check-then-write TOCTOU window, not the API.
	if !strings.Contains(onboardAgentHelp, "TOCTOU") {
		t.Error("onboard race section missing local TOCTOU write guidance")
	}
}

// TestOnboardAgentHelpListsEveryFlag ensures both flags from the audit
// appear in the markdown body, so an agent reading --agent-help can
// see them without grepping source.
func TestOnboardAgentHelpListsEveryFlag(t *testing.T) {
	for _, flag := range []string{"--dir", "--force"} {
		if !strings.Contains(onboardAgentHelp, flag) {
			t.Errorf("onboard --agent-help does not mention flag %q", flag)
		}
	}
}

// TestAgentCmdWiring ensures the parent command and the onboard
// subcommand are properly registered. Catches the "added the verb file
// but forgot to AddCommand in init()" mistake that the compiler would
// also catch — but a test gives a clearer error message.
func TestAgentCmdWiring(t *testing.T) {
	if AgentCmd == nil {
		t.Fatal("AgentCmd is nil — root registration will panic")
	}
	if AgentCmd.Use != "agent <subcommand>" {
		t.Errorf("AgentCmd.Use = %q, want %q", AgentCmd.Use, "agent <subcommand>")
	}
	got := map[string]bool{}
	for _, c := range AgentCmd.Commands() {
		got[c.Name()] = true
	}
	for _, want := range []string{"onboard"} {
		if !got[want] {
			t.Errorf("AgentCmd missing subcommand %q", want)
		}
	}
}
