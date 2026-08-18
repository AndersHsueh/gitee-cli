package issue

import (
	"context"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

func TestCommentsAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee issue comments",
		"Purpose",
		"Pre-conditions",
		"Required arguments",
		"Optional flags",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"comment list",
		"OAuth scope required",
		"Rate limit",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(issueCommentsAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

// TestIssueCommentsArgsBypassWithAgentHelp verifies the
// AgentHelpAwareArgs wrapper on cobra.ExactArgs(1) lets
// `gitee issue comments --agent-help` reach PreRunE without the
// required <number> positional.
func TestIssueCommentsArgsBypassWithAgentHelp(t *testing.T) {
	commentsCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{AgentHelp: true}))
	if err := commentsCmd.Args(commentsCmd, nil); err != nil {
		t.Errorf("issue comments --agent-help with no positional should pass Args, got: %v", err)
	}
}

// TestIssueViewArgsBypassWithAgentHelp verifies the wrapper for
// `gitee issue view --agent-help`.
func TestIssueViewArgsBypassWithAgentHelp(t *testing.T) {
	viewCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{AgentHelp: true}))
	if err := viewCmd.Args(viewCmd, nil); err != nil {
		t.Errorf("issue view --agent-help with no positional should pass Args, got: %v", err)
	}
}
