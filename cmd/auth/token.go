package auth

import (
	"fmt"
	"os"

	"gitee-cli/cmd"
	"gitee-cli/internal/config"

	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the auth token",
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Fprintln(os.Stderr, "auth token is an operational/config command. Use --help for usage.")
			os.Exit(0)
		}
		return nil
	},
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		token := cmd.GlobalToken
		if token == "" {
			token = config.GetToken(cmd.ResolveHostname())
		}
		if token == "" {
			return fmt.Errorf("not authenticated")
		}
		fmt.Fprintln(os.Stdout, token)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored token",
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Fprintln(os.Stderr, "auth logout is an operational/config command. Use --help for usage.")
			os.Exit(0)
		}
		return nil
	},
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		s, err := config.Load()
		if err != nil {
			// If settings file doesn't exist, treat as already logged out
			fmt.Printf("Logged out (no credentials found)\n")
			return nil
		}
		hostname := cmd.ResolveHostname()
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
