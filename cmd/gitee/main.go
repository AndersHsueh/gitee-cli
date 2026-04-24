package main

import (
	"gitee-cli/cmd"
	"gitee-cli/cmd/api"
	"gitee-cli/cmd/auth"
	"gitee-cli/cmd/config"
	"gitee-cli/cmd/gist"
	"gitee-cli/cmd/issue"
	"gitee-cli/cmd/org"
	"gitee-cli/cmd/pr"
	"gitee-cli/cmd/repo"
	"gitee-cli/cmd/search"
)

func main() {
	cmd.RegisterCommands(
		auth.AuthCmd,
		issue.IssueCmd,
		repo.RepoCmd,
		pr.PRCmd,
		gist.GistCmd,
		org.OrgCmd,
		api.APICmd,
		config.ConfigCmd,
		search.SearchCmd,
	)
	cmd.Execute()
}
