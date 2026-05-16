package issue

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/pkg/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var createFlags struct {
	title     string
	body      string
	labels    string
	assignees string
	milestone int
	repo      string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Example: `  # Create an issue in the specified repo
  gitee issue create --repo owner/repo --title "Bug: crash on startup"

  # With body and labels
  gitee issue create --repo owner/repo --title "Feature request" \
    --body "Please add dark mode" --labels "enhancement,help wanted"`,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVarP(&createFlags.title, "title", "t", "", "Issue title (required)")
	f.StringVarP(&createFlags.body, "body", "b", "", "Issue body (supports Markdown)")
	f.StringVarP(&createFlags.labels, "labels", "l", "", "Labels (comma-separated)")
	f.StringVarP(&createFlags.assignees, "assignees", "a", "", "Assignees (comma-separated usernames)")
	f.IntVarP(&createFlags.milestone, "milestone", "m", 0, "Milestone number")
	f.StringVarP(&createFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	_ = createCmd.MarkFlagRequired("title")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	// Resolve --repo: local flag > global --repo
	repo := createFlags.repo
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	if repo == "" {
		return fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid --repo format %q, expected owner/repo", repo)
	}
	owner, repoName := parts[0], parts[1]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"repo":  repoName, // Gitee v5: repo is just the name, owner is in the URL path
		"title": createFlags.title,
	}
	if createFlags.body != "" {
		payload["body"] = createFlags.body
	}
	if createFlags.labels != "" {
		payload["labels"] = createFlags.labels
	}
	if createFlags.assignees != "" {
		payload["assignees"] = createFlags.assignees
	}
	if createFlags.milestone > 0 {
		payload["milestone_number"] = createFlags.milestone
	}

	var issue types.Issue
	// Gitee v5: POST /repos/{owner}/issues (NOT /repos/{owner}/{repo}/issues)
	path := fmt.Sprintf("/repos/%s/issues", owner)

	fmt.Printf("Creating issue in %s/%s...\n", owner, repoName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Post(ctx, path, payload, &issue); err != nil {
		return fmt.Errorf("creating issue: %w", err)
	}

	// Success output
	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s Issue #%s created\n", green("✓"), bold(issue.Number))
	fmt.Printf("  Title : %s\n", issue.Title)
	fmt.Printf("  State : %s\n", issue.State)
	fmt.Printf("  URL   : %s\n", color.CyanString(issue.HTMLURL))
	return nil
}
