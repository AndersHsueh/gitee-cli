package extension

import (
	"strings"
	"testing"
)

// TestListAgentHelpContent verifies the list verb's agent-targeted
// help string contains all the standard sections. If a section is
// dropped during a refactor, this test fails immediately — keeping
// the doc in sync with what `gitee extension list --agent-help`
// actually prints.
func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee extension list",
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
		if !strings.Contains(extensionListAgentHelp, want) {
			t.Errorf("extension list --agent-help missing section: %q", want)
		}
	}
}

func TestInstallAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee extension install",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(extensionInstallAgentHelp, want) {
			t.Errorf("extension install --agent-help missing section: %q", want)
		}
	}
}

func TestRemoveAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee extension remove",
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
		if !strings.Contains(extensionRemoveAgentHelp, want) {
			t.Errorf("extension remove --agent-help missing section: %q", want)
		}
	}
}

func TestUpgradeAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee extension upgrade",
		"## Purpose",
		"## Pre-conditions",
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
		if !strings.Contains(extensionUpgradeAgentHelp, want) {
			t.Errorf("extension upgrade --agent-help missing section: %q", want)
		}
	}
}

func TestCreateAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee extension create",
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
		if !strings.Contains(extensionCreateAgentHelp, want) {
			t.Errorf("extension create --agent-help missing section: %q", want)
		}
	}
}

// TestExtensionCmdWiring ensures the parent command and the 5 subcommands
// are properly registered. Catches the "added the verb file but forgot
// to AddCommand in init()" mistake that the compiler would also catch
// — but a test gives a clearer error message.
func TestExtensionCmdWiring(t *testing.T) {
	if ExtensionCmd == nil {
		t.Fatal("ExtensionCmd is nil — root registration will panic")
	}
	if ExtensionCmd.Use != "extension <subcommand>" {
		t.Errorf("ExtensionCmd.Use = %q, want %q", ExtensionCmd.Use, "extension <subcommand>")
	}
	got := map[string]bool{}
	for _, c := range ExtensionCmd.Commands() {
		got[c.Name()] = true
	}
	for _, want := range []string{"list", "install", "remove", "upgrade", "create"} {
		if !got[want] {
			t.Errorf("ExtensionCmd missing subcommand %q", want)
		}
	}
}
