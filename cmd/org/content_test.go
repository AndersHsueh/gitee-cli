package org

import (
	"strings"
	"testing"
)

// TestListAgentHelpContent verifies the list verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee org list --agent-help` actually prints.
func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee org list",
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
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(orgListAgentHelp, want) {
			t.Errorf("org list --agent-help missing section: %q", want)
		}
	}
}

// TestViewAgentHelpContent verifies the view verb's agent-targeted help
// string contains all the standard sections.
func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee org view",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
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
		if !strings.Contains(orgViewAgentHelp, want) {
			t.Errorf("org view --agent-help missing section: %q", want)
		}
	}
}

// TestMemberAgentHelpContent verifies the members verb's agent-targeted
// help string contains all the standard sections.
func TestMemberAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee org members",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
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
		if !strings.Contains(orgMemberAgentHelp, want) {
			t.Errorf("org members --agent-help missing section: %q", want)
		}
	}
}

// TestRepoAgentHelpContent verifies the repos verb's agent-targeted
// help string contains all the standard sections.
func TestRepoAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee org repos",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
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
		if !strings.Contains(orgRepoAgentHelp, want) {
			t.Errorf("org repos --agent-help missing section: %q", want)
		}
	}
}

// TestIssueAgentHelpContent verifies the issues verb's agent-targeted
// help string contains all the standard sections.
func TestIssueAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee org issues",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Race condition warning",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(orgIssueAgentHelp, want) {
			t.Errorf("org issues --agent-help missing section: %q", want)
		}
	}
}

// TestOrgCmdWiring ensures the parent command and the 5 subcommands
// are properly registered. Catches the "added the verb file but forgot
// to AddCommand in init()" mistake that the compiler would also catch
// — but a test gives a clearer error message.
func TestOrgCmdWiring(t *testing.T) {
	if OrgCmd == nil {
		t.Fatal("OrgCmd is nil — root registration will panic")
	}
	if OrgCmd.Use != "org <subcommand>" {
		t.Errorf("OrgCmd.Use = %q, want %q", OrgCmd.Use, "org <subcommand>")
	}
	got := map[string]bool{}
	for _, c := range OrgCmd.Commands() {
		got[c.Name()] = true
	}
	for _, want := range []string{"list", "view", "members", "repos", "issues"} {
		if !got[want] {
			t.Errorf("OrgCmd missing subcommand %q", want)
		}
	}
}
