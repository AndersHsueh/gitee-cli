package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Args:  cobra.NoArgs,
	Short: "Show authentication status",
	Example: `  gitee auth status
  gitee auth status --hostname enterprise.gitee.com
  gitee auth status -o json`,
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
		if rt.Output == "json" {
			_ = writeStatusJSON(cobraCmd.OutOrStdout(), statusPayload{
				Authenticated: false,
				Hostname:      hostname,
				Message:       "not authenticated. Run 'gitee auth login' to authenticate",
			})
		}
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

	tokenSource := statusTokenSource(rt)
	if rt.Output == "json" {
		tok := &statusToken{
			Masked: maskToken(token),
			Source: tokenSource,
		}
		if showToken {
			tok.Value = token
		}
		return writeStatusJSON(cobraCmd.OutOrStdout(), statusPayload{
			Authenticated: true,
			Hostname:      hostname,
			User: &statusUser{
				Login:       user.Login,
				Name:        user.Name,
				Email:       user.Email,
				HTMLURL:     user.HTMLURL,
				ID:          user.ID,
				PublicRepos: user.PublicRepos,
				PublicGists: user.PublicGists,
				Followers:   user.Followers,
				Following:   user.Following,
			},
			Token: tok,
		})
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

	fmt.Printf("  Source:   %s\n", tokenSource)

	fmt.Printf("\n")
	return nil
}

type statusPayload struct {
	Authenticated bool         `json:"authenticated"`
	Hostname      string       `json:"hostname"`
	Message       string       `json:"message,omitempty"`
	User          *statusUser  `json:"user,omitempty"`
	Token         *statusToken `json:"token,omitempty"`
}

type statusUser struct {
	Login       string `json:"login"`
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	HTMLURL     string `json:"html_url,omitempty"`
	ID          int64  `json:"id"`
	PublicRepos int    `json:"public_repos"`
	PublicGists int    `json:"public_gists"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
}

type statusToken struct {
	Masked string `json:"masked"`
	Source string `json:"source"`
	Value  string `json:"value,omitempty"`
}

func statusTokenSource(rt *cmd.Runtime) string {
	// rt.Token is bound only to the --token flag, never to os.Getenv.
	// Flag > env > config file.
	if rt != nil && rt.Token != "" {
		return "--token flag"
	}
	if os.Getenv("GITEE_TOKEN") != "" {
		return "GITEE_TOKEN environment variable"
	}
	return "config file"
}

func writeStatusJSON(w io.Writer, payload statusPayload) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("writing auth status json: %w", err)
	}
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
