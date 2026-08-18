package search

import (
	"context"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

// TestReposAgentHelpContent verifies the repos verb's agent-targeted help
// string contains all the standard sections. If a section is dropped
// during a refactor, this test fails immediately — keeping the doc
// in sync with what `gitee search repos --agent-help` actually prints.
func TestReposAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee search repos",
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
		// Every flag listed in the audit must appear in the markdown body.
		"--language",
		"--sort",
		"--order",
		"--limit / -L",
	}
	for _, want := range required {
		if !strings.Contains(searchReposAgentHelp, want) {
			t.Errorf("search repos --agent-help missing section/flag: %q", want)
		}
	}
	// Spot-check the race section carries command-specific guidance,
	// not a bare "point-in-time snapshot" placeholder.
	if !strings.Contains(searchReposAgentHelp, "renamed, deleted, or switched to private") {
		t.Error("search repos race section missing repo-rename/delete guidance")
	}
}

// TestSearchReposArgsBypassWithAgentHelp verifies the
// AgentHelpAwareArgs wrapper on cobra.MinimumNArgs(1) lets
// `gitee search repos --agent-help` reach PreRunE without a
// <query> positional.
func TestSearchReposArgsBypassWithAgentHelp(t *testing.T) {
	searchReposCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{AgentHelp: true}))
	if err := searchReposCmd.Args(searchReposCmd, nil); err != nil {
		t.Errorf("search repos --agent-help with no positional should pass Args, got: %v", err)
	}
}

// TestSearchIssuesArgsBypassWithAgentHelp verifies the wrapper for
// `gitee search issues --agent-help`.
func TestSearchIssuesArgsBypassWithAgentHelp(t *testing.T) {
	searchIssuesCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{AgentHelp: true}))
	if err := searchIssuesCmd.Args(searchIssuesCmd, nil); err != nil {
		t.Errorf("search issues --agent-help with no positional should pass Args, got: %v", err)
	}
}

// TestSearchUsersArgsBypassWithAgentHelp verifies the wrapper for
// `gitee search users --agent-help`.
func TestSearchUsersArgsBypassWithAgentHelp(t *testing.T) {
	searchUsersCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{AgentHelp: true}))
	if err := searchUsersCmd.Args(searchUsersCmd, nil); err != nil {
		t.Errorf("search users --agent-help with no positional should pass Args, got: %v", err)
	}
}
