package alias

import (
	"strings"
	"testing"
)

// TestSetAgentHelpContent verifies the set verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee alias set --agent-help` actually prints.
func TestSetAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee alias set",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(aliasSetAgentHelp, want) {
			t.Errorf("alias set --agent-help missing section: %q", want)
		}
	}
}

func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee alias list",
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
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(aliasListAgentHelp, want) {
			t.Errorf("alias list --agent-help missing section: %q", want)
		}
	}
}

func TestDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee alias delete",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(aliasDeleteAgentHelp, want) {
			t.Errorf("alias delete --agent-help missing section: %q", want)
		}
	}
}

// TestAliasCmdWiring ensures the parent command and the 3 subcommands
// are properly registered. Catches the "added the verb file but forgot
// to AddCommand in init()" mistake that the compiler would also catch
// — but a test gives a clearer error message.
func TestAliasCmdWiring(t *testing.T) {
	if AliasCmd == nil {
		t.Fatal("AliasCmd is nil — root registration will panic")
	}
	if AliasCmd.Use != "alias <subcommand>" {
		t.Errorf("AliasCmd.Use = %q, want %q", AliasCmd.Use, "alias <subcommand>")
	}
	got := map[string]bool{}
	for _, c := range AliasCmd.Commands() {
		got[c.Name()] = true
	}
	for _, want := range []string{"set", "list", "delete"} {
		if !got[want] {
			t.Errorf("AliasCmd missing subcommand %q", want)
		}
	}
}
