package api

import (
	"testing"

	"gitee-cli/cmd"
)

// TestRawArgsBypassWithAgentHelp verifies the AgentHelpAwareArgs
// wrapper on cobra.ExactArgs(2) lets `gitee api raw --agent-help`
// reach PreRunE without the <method> <path> positional pair. This
// is the task-A part of the api raw fix; task B adds the PreRunE
// hook so the doc actually prints before auth checks.
func TestRawArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	if err := rawCmd.Args(rawCmd, nil); err != nil {
		t.Errorf("api raw --agent-help with no positional should pass Args, got: %v", err)
	}
}

// TestAPIArgsBypassWithAgentHelp verifies the same wrapper for the
// parent `gitee api` command (the raw subcommand is also registered
// under it for backwards compatibility).
func TestAPIArgsBypassWithAgentHelp(t *testing.T) {
	orig := cmd.GlobalAgentHelp
	defer func() { cmd.GlobalAgentHelp = orig }()
	cmd.GlobalAgentHelp = true
	if err := apiCmd.Args(apiCmd, nil); err != nil {
		t.Errorf("api --agent-help with no positional should pass Args, got: %v", err)
	}
}