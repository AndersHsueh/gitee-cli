package auth

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/config"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Args:  cobra.NoArgs,
	Short: "Authenticate with a Gitee host",
	Long: `Authenticate with gitee.com or a Gitee Enterprise instance.

By default, login reads an existing token from
~/.gitee-cli/settings.json (or the GITEE_TOKEN environment variable)
and verifies it against the /user endpoint.

Pass --token to log in with a new token; on success the token is
written to ~/.gitee-cli/settings.json so later commands can use it
without re-authenticating.`,
	PreRunE: cmd.AgentHelpNA("auth login"),
	RunE:    runLogin,
}

func init() {
	f := loginCmd.Flags()
	f.StringP("token", "t", "", "Use a specific token for authentication; saved to settings.json on success")
}

func runLogin(cobraCmd *cobra.Command, args []string) error {
	_ = cmd.RuntimeFromCmd(cobraCmd)
	hostname := cmd.ResolveHostname()

	// Resolve the effective token. The local --token flag (registered
	// in init() above) shadows the persistent --token flag bound to
	// rt.Token, so GlobalToken is always empty here. The real
	// fallback chain is: --token flag > config.GetToken (which checks
	// GITEE_TOKEN env var, then ~/.gitee-cli/settings.json).
	tokenFlag, _ := cobraCmd.Flags().GetString("token")
	token := tokenFlag
	if token == "" {
		token = config.GetToken(hostname)
	}

	if token == "" {
		printNoTokenHint(hostname)
		return fmt.Errorf("not authenticated")
	}

	// Build a client just for verification — don't pollute rt.Token
	// with the resolved token, since status.go and NewAPIClient read it back.
	apiClient := api.NewClient(token, apiBaseURL(hostname), cmd.GlobalVerbose, cmd.GlobalTimeout)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var user types.User
	if err := apiClient.Get(ctx, "/user", &user); err != nil {
		return fmt.Errorf("verifying token against %s: %w", hostname, err)
	}

	// Persist when the caller explicitly supplied --token so future
	// invocations skip the network round-trip. Tokens sourced from the
	// env var or settings.json are already in place.
	if tokenFlag != "" {
		if err := persistToken(hostname, tokenFlag, user.Login); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Logged in to %s as %s\n", green("✓"), bold(hostname), bold(user.Login))
	return nil
}

func apiBaseURL(hostname string) string {
	return cmd.ResolveAPIBaseURL(hostname)
}

// persistToken writes a fresh token + username into ~/.gitee-cli/settings.json
// without disturbing other hosts or preferences already on disk.
func persistToken(hostname, token, username string) error {
	s, err := config.Load()
	if err != nil {
		return err
	}
	hc, ok := s.Hosts[hostname]
	if !ok {
		hc = config.HostConfig{Protocol: "https"}
	}
	// Don't downgrade a stored token to one the user just passed.
	if hc.Token == token {
		return nil
	}
	hc.Token = token
	hc.Username = username
	hc.Protocol = "https"
	s.Hosts[hostname] = hc
	return config.Save(s)
}

func printNoTokenHint(hostname string) {
	fmt.Printf("No token found for %s.\n\n", hostname)
	fmt.Println("Set one of the following and re-run 'gitee auth login':")
	fmt.Println("  - environment variable:  export GITEE_TOKEN=<your-token>")
	fmt.Println("  - settings.json entry:   pass --token <your-token>")
	fmt.Printf("  - or edit ~/.gitee-cli/settings.json and add a token under hosts.%s\n", hostname)
}
