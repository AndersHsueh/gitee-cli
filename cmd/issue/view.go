package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/pkg/types"
)

var viewFlags struct {
	repo string
}

var viewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View details of an issue",
	Args:  cobra.ExactArgs(1),
	Example: `  gitee issue view 42 --repo owner/repo`,
	RunE: runView,
}

func init() {
	viewCmd.Flags().StringVarP(&viewFlags.repo, "repo", "R", "", "Repository (owner/repo)")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	repo := viewFlags.repo
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	if repo == "" {
		return fmt.Errorf("--repo is required")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid --repo format, expected owner/repo")
	}
	owner, repoName := parts[0], parts[1]
	number := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repoName, number)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var issue types.Issue
	if err := client.Get(ctx, path, &issue); err != nil {
		return fmt.Errorf("viewing issue: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(issue, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s  #%s\n\n", bold(issue.Title), issue.Number)
	fmt.Printf("  State  : %s\n", issue.State)
	fmt.Printf("  Author : %s\n", issue.User.Login)
	if issue.Assignee != nil {
		fmt.Printf("  Assignee: %s\n", issue.Assignee.Login)
	}
	if len(issue.Labels) > 0 {
		names := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			names[i] = l.Name
		}
		fmt.Printf("  Labels : %s\n", strings.Join(names, ", "))
	}
	fmt.Printf("  URL    : %s\n", cyan(issue.HTMLURL))
	fmt.Printf("  Created: %s\n", issue.CreatedAt.Format("2006-01-02 15:04"))
	if issue.Body != "" {
		fmt.Printf("\n%s\n", issue.Body)
	}
	return nil
}
