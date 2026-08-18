package alias

import (
	"context"
	"strings"
	"testing"

	"gitee-cli/cmd"
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
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(aliasSetAgentHelp, want) {
			t.Errorf("alias set --agent-help missing section: %q", want)
		}
	}
	// Local command: OAuth/Rate-limit sections must carry the N/A reason.
	if !strings.Contains(aliasSetAgentHelp, "N/A (local command) — no Gitee API request is made") {
		t.Error("alias set --agent-help missing local-command N/A reason")
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
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(aliasListAgentHelp, want) {
			t.Errorf("alias list --agent-help missing section: %q", want)
		}
	}
	// Local command: OAuth/Rate-limit sections must carry the N/A reason.
	if !strings.Contains(aliasListAgentHelp, "N/A (local command) — no Gitee API request is made") {
		t.Error("alias list --agent-help missing local-command N/A reason")
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
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(aliasDeleteAgentHelp, want) {
			t.Errorf("alias delete --agent-help missing section: %q", want)
		}
	}
	// Local command: OAuth/Rate-limit sections must carry the N/A reason.
	if !strings.Contains(aliasDeleteAgentHelp, "N/A (local command) — no Gitee API request is made") {
		t.Error("alias delete --agent-help missing local-command N/A reason")
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

// TestSetArgsPassesWithAgentHelp verifies the AgentHelpAwareArgs
// wrapper around the custom <name> <expansion> validator lets
// `gitee alias set --agent-help` reach PreRunE even when no
// positional args are supplied. Before the wrapper this would fail
// at cobra.ValidateArgs with "accepts 2 arg(s), received 0".
func TestSetArgsPassesWithAgentHelp(t *testing.T) {
	setCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{AgentHelp: true}))
	if err := setCmd.Args(setCmd, nil); err != nil {
		t.Errorf("Args should pass when --agent-help is set, got: %v", err)
	}
}

// TestSetArgsFailsWithoutAgentHelp confirms the wrapper still
// surfaces the inner validator's error in the normal case (no flag).
func TestSetArgsFailsWithoutAgentHelp(t *testing.T) {
	setCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{AgentHelp: false}))
	if err := setCmd.Args(setCmd, nil); err == nil {
		t.Error("Args should fail without --agent-help and no positional args")
	}
}

// TestDeleteArgsPassesWithAgentHelp verifies the wrapper on
// cobra.ExactArgs(1) lets `gitee alias delete --agent-help` reach
// PreRunE without a <name>.
func TestDeleteArgsPassesWithAgentHelp(t *testing.T) {
	deleteCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{AgentHelp: true}))
	if err := deleteCmd.Args(deleteCmd, nil); err != nil {
		t.Errorf("Args should pass when --agent-help is set, got: %v", err)
	}
}
