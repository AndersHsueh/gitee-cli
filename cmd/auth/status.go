package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type userResponse struct {
	ID               int64  `json:"id"`
	Login            string `json:"login"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	AvatarURL        string `json:"avatar_url"`
	HTMLURL          string `json:"html_url"`
	Blog             string `json:"blog"`
	Bio              string `json:"bio"`
	PublicRepos      int    `json:"public_repos"`
	PublicGists      int    `json:"public_gists"`
	Followers        int    `json:"followers"`
	Following        int    `json:"following"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	Type             string `json:"type"`
	StarredURL       string `json:"starred_url"`
	SubscriptionsURL string `json:"subscriptions_url"`
	OrganizationsURL string `json:"organizations_url"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Example: `  gitee auth status
  gitee auth status --hostname enterprise.gitee.com`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().Bool("show-token", false, "Show full token (use with caution)")
}

func runStatus(cobraCmd *cobra.Command, args []string) error {
	hostname := cmd.GlobalHostname
	token := cmd.GlobalToken
	showToken, _ := cobraCmd.Flags().GetBool("show-token")

	if token == "" {
		token = config.GetToken(hostname)
	}
	if token == "" {
		return fmt.Errorf("not authenticated. Run 'gitee auth login' to authenticate")
	}

	apiClient, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var user userResponse
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

	// Token source
	tokenSource := "config file"
	if cmd.GlobalToken != "" {
		tokenSource = "GITEE_TOKEN environment variable or --token flag"
	}
	fmt.Printf("  Source:   %s\n", tokenSource)

	// Scope info (if available in config)
	if scopes := config.GetScopes(hostname); len(scopes) > 0 {
		fmt.Printf("\n%s\n", bold("Scopes:"))
		for _, scope := range scopes {
			fmt.Printf("  - %s\n", scope)
		}
	}

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
