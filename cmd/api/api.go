package api

import (
	"github.com/spf13/cobra"
)

var APICmd = &cobra.Command{
	Use:   "api <method> <path>",
	Short: "Make raw API requests",
	Long: `Make raw HTTP requests to the Gitee API.
Useful for testing or accessing endpoints not yet covered by dedicated commands.

Examples:
  gitee api GET /user
  gitee api GET /user/repos?type=owner
  gitee api POST /user/repos --field name=my-repo --field private=true
  gitee api DELETE /repos/owner/repo`,
}

func init() {
	APICmd.AddCommand(rawCmd)
}
