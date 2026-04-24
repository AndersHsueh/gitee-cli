package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/config"
)

type userResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		hostname := cmd.GlobalHostname
		token := cmd.GlobalToken
		if token == "" {
			token = config.GetToken(hostname)
		}
		if token == "" {
			return fmt.Errorf("not authenticated. Set GITEE_TOKEN or run 'gitee auth login'")
		}

		apiClient, err := cmd.NewAPIClient()
		if err != nil {
			return err
		}

		var user userResponse
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
		defer cancel()

		if err := apiClient.Get(ctx, "/user", &user); err != nil {
			return fmt.Errorf("verifying token: %w", err)
		}

		green := color.New(color.FgGreen).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()

		fmt.Printf("%s Logged in to %s as %s\n",
			green("✓"),
			bold(hostname),
			bold(user.Login),
		)
		if user.Name != "" {
			fmt.Printf("  Name: %s\n", user.Name)
		}
		// Redact token: show only last 4 chars
		masked := "****" + token[len(token)-4:]
		fmt.Printf("  Token: %s\n", masked)
		return nil
	},
}
