package main

import (
	"gitee-cli/cmd"
	"gitee-cli/cmd/api"
	"gitee-cli/cmd/auth"
	"gitee-cli/cmd/config"
	"gitee-cli/cmd/gist"
	"gitee-cli/cmd/gpg-key"
	"gitee-cli/cmd/issue"
	"gitee-cli/cmd/label"
	"gitee-cli/cmd/org"
	"gitee-cli/cmd/pr"
	"gitee-cli/cmd/release"
	"gitee-cli/cmd/repo"
	"gitee-cli/cmd/search"
	"gitee-cli/cmd/ssh-key"
	"gitee-cli/cmd/webhook"
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
		// P1a additions
		sshkey.SSHKeyCmd,
		gpgkey.GPGKeyCmd,
		// P1b additions
		webhook.WebhookCmd,
	)
	cmd.Execute()
}
