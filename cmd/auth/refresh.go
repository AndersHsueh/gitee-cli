package auth

import (
	"fmt"
	"os"

	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh stored authentication credentials",
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Fprintln(os.Stderr, "auth refresh is an operational/config command. Use --help for usage.")
			os.Exit(0)
		}
		return nil
	},
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		fmt.Println("Token refresh is not yet supported. Please re-authenticate with 'gitee auth login'.")
		return nil
	},
}
