package auth

import (
	"fmt"
	"os"

	"gitee-cli/cmd"
	"gitee-cli/internal/config"

	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:      "token",
	Short:    "Print the auth token",
	PreRunE:  cmd.AgentHelpNA("auth token"),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
		token := rt.Token
		if token == "" {
			token = config.GetToken(rt.Hostname)
		}
		if token == "" {
			return fmt.Errorf("not authenticated")
		}
		fmt.Fprintln(os.Stdout, token)
		return nil
	},
}
