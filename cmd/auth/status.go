package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/config"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Example: `  gitee auth status
  gitee auth status --hostname enterprise.gitee.com`,
	PreRunE: cmd.AgentHelpNA("auth status"),
	RunE:    runStatus,
}

func init() {
	statusCmd.Flags().Bool("show-token", false, "Show full token (use with caution)")
}

func runStatus(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	hostname := rt.Hostname
	token := rt.Token
	showToken, _ := cobraCmd.Flags().GetBool("show-token")

	if token == "" {
		token = config.GetToken(hostname)
	}
	if token == "" {
		return fmt.Errorf("not authenticated. Run 'gitee auth login' to authenticate")
	}

	apiClient, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var user types.User
	if err := apiClient.Get(ctx, "/user", &user); err != nil {
		return fmt.Errorf("verifying token: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────────────┐\n")
	fmt.Printf("│  %s  Authenticated to %s  │\n", green("✓"), bold(hostname))
	fmt.Printf("└─────────────────────────────────────────────────┘\n")
	fmt.Printf("\n")

	// User info
	fmt.Printf("%s %s\n", bold("User:"), bold(user.Login))
	if user.Name != "" && user.Name != user.Login {
		fmt.Printf("  Name:     %s\n", user.Name)
	}
	if user.Email != "" {
		fmt.Printf("  Email:    %s\n", user.Email)
	}
	if user.HTMLURL != "" {
		fmt.Printf("  Profile:  %s\n", cyan(user.HTMLURL))
	}
	if user.Bio != "" {
		fmt.Printf("  Bio:      %s\n", user.Bio)
	}
	fmt.Printf("  User ID:  %d\n", user.ID)

	// Stats
	fmt.Printf("\n%s\n", bold("Stats:"))
	fmt.Printf("  Repos:      %d\n", user.PublicRepos)
	fmt.Printf("  Gists:      %d\n", user.PublicGists)
	fmt.Printf("  Followers:  %d\n", user.Followers)
	fmt.Printf("  Following:  %d\n", user.Following)

	// Token info
	fmt.Printf("\n%s\n", bold("Token:"))
	if showToken {
		fmt.Printf("  Token:    %s\n", token)
		fmt.Printf("  %s\n", color.New(color.FgYellow).SprintfFunc()("  ⚠️  Warning: Exposing token in plain text!"))
	} else {
		fmt.Printf("  Token:    %s\n", maskToken(token))
		fmt.Printf("  Hint:     Use --show-token to display full token\n")
	}

	// Token source: GlobalToken is bound only to the --token flag, never to
	// os.Getenv. Check the flag first (flag > env > config precedence), then
	// the env var; config file is the fallback.
	tokenSource := "config file"
	if rt.Token != "" {
		tokenSource = "--token flag"
	} else if os.Getenv("GITEE_TOKEN") != "" {
		tokenSource = "GITEE_TOKEN environment variable"
	}
	fmt.Printf("  Source:   %s\n", tokenSource)

	fmt.Printf("\n")
	return nil
}

// maskToken returns a redacted form of the token. Tokens shorter than 8
// characters are fully masked to avoid revealing meaningful prefixes.
func maskToken(token string) string {
	if len(token) < 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + "..." + token[len(token)-4:]
}
