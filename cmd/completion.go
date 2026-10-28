package cmd

import (
	"github.com/spf13/cobra"
)

// This file replaces cobra's built-in `completion` command with a
// custom one that participates in the --agent-help contract. Cobra's
// default completion subcommands (bash/fish/powershell/zsh) all dump
// the actual shell script when probed with --agent-help, which is the
// exact trap the agent-help system is designed to avoid. We disable
// cobra's default via rootCmd.CompletionOptions.DisableDefaultCmd
// (see root.go) and register four equivalent subcommands here, each
// with a PreRunE that short-circuits to the operational/config n/a
// notice when --agent-help is set.
//
// The actual completion script generation reuses cobra's
// Gen*Completion helpers, so output is byte-for-byte identical to
// what cobra's built-in would have produced for any other invocation
// (e.g. `gitee completion bash > /etc/bash_completion.d/gitee`).
//
// We do NOT replicate the --no-descriptions flag that cobra's built-in
// offers; IK923X step 1 only requires --agent-help compliance. If a
// future step needs --no-descriptions parity, add it then.

// completionCmd is the parent command for shell-completion subcommands.
// We keep a package-level reference (mirroring versionCmd) so
// completion_test.go can assert the wiring without walking the tree.
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate the autocompletion script for the specified shell",
	Long: `Generate the autocompletion script for gitee for the specified shell.
See each sub-command's help for details on how to use the generated script.`,
	// DisableFlagParsing is intentionally NOT set; we want --agent-help
	// to flow through PersistentPreRunE/PreRunE so the n/a notice fires.
	//
	// GroupRunE: `gitee completion <unknown-shell>` must exit non-zero
	// instead of printing help with a 0 status; bare `gitee completion`
	// still shows help.
	RunE: GroupRunE,
}

// bashCompletionCmd / fishCompletionCmd / powershellCompletionCmd /
// zshCompletionCmd are the four leaf commands. They share the same
// shape (PreRunE → AgentHelpNA, RunE → Gen*Completion) so we use a
// small factory to keep them in lockstep.
var (
	bashCompletionCmd = newShellCompletionCmd("bash", func(cmd *cobra.Command, _ []string) error {
		return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), true)
	})
	fishCompletionCmd = newShellCompletionCmd("fish", func(cmd *cobra.Command, _ []string) error {
		return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
	})
	powershellCompletionCmd = newShellCompletionCmd("powershell", func(cmd *cobra.Command, _ []string) error {
		return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
	})
	zshCompletionCmd = newShellCompletionCmd("zsh", func(cmd *cobra.Command, _ []string) error {
		return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
	})
)

// newShellCompletionCmd builds a subcommand under completionCmd with
// the standard PreRunE (n/a notice on --agent-help) and the supplied
// RunE. The shellName is used both as the cobra Use value and as the
// "completion <shell>" prefix in the n/a notice (so the message
// matches what an agent would type: `gitee completion bash --agent-help`).
func newShellCompletionCmd(shellName string, runE func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:   shellName,
		Short: "Generate the autocompletion script for " + shellName,
		// cobra's built-in uses NoArgs + DisableFlagsInUseLine + NoFileCompletions
		// for the leaf commands; we mirror them so the custom subcommand behaves
		// identically to the cobra default.
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     cobra.NoFileCompletions,
		PreRunE:               AgentHelpNA("completion " + shellName),
		RunE:                  runE,
	}
}

func init() {
	// Disable cobra's built-in `completion` command tree so our
	// custom subcommands below are the only ones under `gitee completion`.
	// Set on rootCmd directly (not on completionCmd) so the option is
	// honored before cobra's initDefaultCompletionCmd runs.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	completionCmd.AddCommand(bashCompletionCmd)
	completionCmd.AddCommand(fishCompletionCmd)
	completionCmd.AddCommand(powershellCompletionCmd)
	completionCmd.AddCommand(zshCompletionCmd)
	rootCmd.AddCommand(completionCmd)
}
