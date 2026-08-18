package auth

import (
	"fmt"

	"gitee-cli/cmd"
	"gitee-cli/internal/config"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:      "logout",
	Short:    "Log out and remove stored token",
	PreRunE:  cmd.AgentHelpNA("auth logout"),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
		s, err := config.Load()
		if err != nil {
			// If settings file doesn't exist, treat as already logged out
			fmt.Printf("Logged out (no credentials found)\n")
			return nil
		}
		hostname := rt.Hostname
		if _, ok := s.Hosts[hostname]; !ok {
			fmt.Printf("Not logged in to %s\n", hostname)
			return nil
		}
		delete(s.Hosts, hostname)
		if err := config.Save(s); err != nil {
			return fmt.Errorf("saving settings: %w", err)
		}
		fmt.Printf("Logged out of %s\n", hostname)
		return nil
	},
}
