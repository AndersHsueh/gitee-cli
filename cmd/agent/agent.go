package agent

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

// AgentCmd is the top-level command for agent-related tooling.
var AgentCmd = &cobra.Command{
	Use:   "agent <subcommand>",
	Short: "Agent tooling — onboard, scaffold, and configure AI agent workflows",
	Long: `Tools for bootstrapping AI agent workflows with gitee-cli.

The 'agent' command group helps you set up agent-friendly project
configuration so that AI coding tools (Claude Code, Cursor, Cline,
etc.) can use gitee-cli effectively and reliably.

Start with 'gitee agent onboard' to generate a .gitee/agent.md
file with pre-built orchestration templates.`,
	// GroupRunE: unknown verbs exit non-zero with a suggestion; no args shows help.
	RunE: cmd.GroupRunE,
}

func init() {
	AgentCmd.AddCommand(onboardCmd)
}
