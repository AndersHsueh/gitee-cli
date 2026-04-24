package main

import (
	"gitee-cli/cmd"
	"gitee-cli/cmd/auth"
	"gitee-cli/cmd/config"
	"gitee-cli/cmd/issue"
	"gitee-cli/cmd/search"
)

func main() {
	cmd.RegisterCommands(
		auth.AuthCmd,
		issue.IssueCmd,
		config.ConfigCmd,
		search.SearchCmd,
	)
	cmd.Execute()
}
