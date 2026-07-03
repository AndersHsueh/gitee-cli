package main

import (
	"gitee-cli/cmd"
	"gitee-cli/cmd/agent"
	"gitee-cli/cmd/alias"
	"gitee-cli/cmd/api"
	"gitee-cli/cmd/auth"
	"gitee-cli/cmd/config"
	cmdext "gitee-cli/cmd/extension"
	"gitee-cli/cmd/gist"
	"gitee-cli/cmd/help_topics"
	"gitee-cli/cmd/issue"
	"gitee-cli/cmd/label"
	"gitee-cli/cmd/milestone"
	"gitee-cli/cmd/notification"
	"gitee-cli/cmd/org"
	"gitee-cli/cmd/pr"
	"gitee-cli/cmd/release"
	"gitee-cli/cmd/repo"
	"gitee-cli/cmd/search"
	sshkey "gitee-cli/cmd/ssh-key"
	gpgkey "gitee-cli/cmd/gpg-key"
	"gitee-cli/cmd/status"
	"gitee-cli/cmd/webhook"
	gitext "gitee-cli/internal/extension"
)

func main() {
	cmd.RegisterCommands(
		auth.AuthCmd,
		issue.IssueCmd,
		repo.RepoCmd,
		pr.PRCmd,
		pr.CoCmd, // `gitee co` alias of `gitee pr checkout`
		gist.GistCmd,
		org.OrgCmd,
		api.APICmd,
		config.ConfigCmd,
		search.SearchCmd,
		// P0 additions
		release.ReleaseCmd,
		label.LabelCmd,
		milestone.MilestoneCmd,
		notification.NotificationCmd,
		// P1a additions
		sshkey.SSHKeyCmd,
		// gpg-key added in P2-9 — endpoints/types already existed; only cmd layer is new.
		gpgkey.GPGKeyCmd,
		// P1b additions
		webhook.WebhookCmd,
		// P2a additions
		status.StatusCmd,
		// P2b additions
		cmdext.ExtensionCmd,
		// P2c additions
		alias.AliasCmd,
		// Agent tooling (v0.10.0)
		agent.AgentCmd,
	)
	// Dynamic loaders: register extension commands from
	// ~/.gitee-cli/extensions/. Skipped when --no-extensions is set.
	cmd.RegisterDynamicCommands(gitext.LoadCommands)
	// Cross-cutting help topics (environment, exit-codes, formatting,
	// reference). Must be registered after all built-in commands so
	// 'gitee help reference' walks the full tree.
	help_topics.RegisterHelpTopics(cmd.RootCommand())
	cmd.Execute()
}
