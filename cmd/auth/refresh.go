package auth

import (
	"fmt"

	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:      "refresh",
	Short:    "Refresh stored authentication credentials",
	PreRunE:  cmd.AgentHelpNA("auth refresh"),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		fmt.Println("Token refresh is not yet supported. Please re-authenticate with 'gitee auth login'.")
		return nil
	},
}
