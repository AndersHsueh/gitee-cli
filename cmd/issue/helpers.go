package issue

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
)

// resolveRepo returns owner and repoName from --owner/--repo flags or the
// combined --repo owner/repo flag on the root command.
func resolveRepo(owner, repo string) (string, string, error) {
	// If both are provided explicitly, use them
	if owner != "" && repo != "" {
		return owner, repo, nil
	}
	// Try combined format
	combined := repo
	if combined == "" {
		combined = cmd.GlobalRepo
	}
	if combined == "" {
		return "", "", fmt.Errorf("--repo is required (e.g. --repo owner/repo or --owner owner --repo name)")
	}
	parts := strings.SplitN(combined, "/", 2)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}
	// If owner is set separately
	if owner != "" {
		return owner, combined, nil
	}
	return "", "", fmt.Errorf("invalid --repo format %q, expected owner/repo", combined)
}

var editFlags struct {
	owner     string
	repo      string
	title     string
	body      string
	labels    string
	assignees string
	milestone int
}

var closeFlags struct {
	owner string
	repo  string
}

var reopenFlags struct {
	owner string
	repo  string
}

var commentFlags struct {
	owner string
	repo  string
	body  string
}

var editCmd = &cobra.Command{
	Use:   "edit <number>",
	Short: "Edit an issue",
	Example: `  gitee issue edit 42 --repo owner/repo --title "New title"`,
	RunE:  runEdit,
}

var closeCmd = &cobra.Command{
	Use:   "close <number>",
	Short: "Close an issue",
	Example: `  gitee issue close 42 --repo owner/repo`,
	RunE:  runClose,
}

var reopenCmd = &cobra.Command{
	Use:   "reopen <number>",
	Short: "Reopen an issue",
	Example: `  gitee issue reopen 42 --repo owner/repo`,
	RunE:  runReopen,
}

var commentCmd = &cobra.Command{
	Use:   "comment <number>",
	Short: "Add a comment to an issue",
	Example: `  gitee issue comment 42 --repo owner/repo --body "Comment text"`,
	RunE:  runComment,
}

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Manage issue labels",
}

var labelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List labels in a repository",
	RunE: func(c *cobra.Command, args []string) error {
		owner, _ := c.Flags().GetString("owner")
		repo, _ := c.Flags().GetString("repo")
		o, r, err := resolveRepo(owner, repo)
		if err != nil {
			return err
		}
		apiClient, err := cmd.NewAPIClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
		defer cancel()
		var labels []types.Label
		path := fmt.Sprintf("/repos/%s/%s/labels", o, r)
		if err := apiClient.Get(ctx, path, &labels); err != nil {
			return err
		}
		for _, l := range labels {
			fmt.Printf("  %s  %s\n", color.New(color.FgHiBlack).Sprintf("#%s", l.Color), l.Name)
		}
		return nil
	},
}

func init() {
	f := editCmd.Flags()
	f.StringVar(&editFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&editFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVar(&editFlags.title, "title", "", "New title")
	f.StringVar(&editFlags.body, "body", "", "New body")
	f.StringVar(&editFlags.labels, "labels", "", "Comma-separated labels")
	f.StringVar(&editFlags.assignees, "assignees", "", "Comma-separated assignees")
	f.IntVar(&editFlags.milestone, "milestone", 0, "Milestone number")

	f = closeCmd.Flags()
	f.StringVar(&closeFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&closeFlags.repo, "repo", "R", "", "Repository name or owner/repo")

	f = reopenCmd.Flags()
	f.StringVar(&reopenFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&reopenFlags.repo, "repo", "R", "", "Repository name or owner/repo")

	f = commentCmd.Flags()
	f.StringVar(&commentFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&commentFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVar(&commentFlags.body, "body", "", "Comment body (required)")
	_ = commentCmd.MarkFlagRequired("body")

	labelCmd.AddCommand(labelListCmd)
	labelListCmd.Flags().String("owner", "", "Repository owner")
	labelListCmd.Flags().String("repo", "", "Repository name or owner/repo")
}

func runEdit(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("issue number is required")
	}
	owner, repoName, err := resolveRepo(editFlags.owner, editFlags.repo)
	if err != nil {
		return err
	}
	var issueNum int
	fmt.Sscanf(args[0], "%d", &issueNum)

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.IssueEditRequest{}
	if editFlags.title != "" {
		req.Title = editFlags.title
	}
	if editFlags.body != "" {
		req.Body = editFlags.body
	}
	if editFlags.labels != "" {
		req.Labels = editFlags.labels
	}
	if editFlags.assignees != "" {
		req.Assignees = editFlags.assignees
	}
	if editFlags.milestone > 0 {
		req.Milestone = editFlags.milestone
	}

	path := api.IssuesEdit(owner, repoName, issueNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var issue types.Issue
	if err := client.Patch(ctx, path, req, &issue); err != nil {
		return fmt.Errorf("editing issue: %w", err)
	}
	fmt.Printf("Issue #%d updated\n", issueNum)
	return nil
}

func runClose(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("issue number is required")
	}
	owner, repoName, err := resolveRepo(closeFlags.owner, closeFlags.repo)
	if err != nil {
		return err
	}
	var issueNum int
	fmt.Sscanf(args[0], "%d", &issueNum)

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := map[string]string{"state": "closed"}
	path := api.IssuesEdit(owner, repoName, issueNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var issue types.Issue
	if err := client.Patch(ctx, path, req, &issue); err != nil {
		return fmt.Errorf("closing issue: %w", err)
	}
	fmt.Printf("Issue #%d closed\n", issueNum)
	return nil
}

func runReopen(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("issue number is required")
	}
	owner, repoName, err := resolveRepo(reopenFlags.owner, reopenFlags.repo)
	if err != nil {
		return err
	}
	var issueNum int
	fmt.Sscanf(args[0], "%d", &issueNum)

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := map[string]string{"state": "open"}
	path := api.IssuesEdit(owner, repoName, issueNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var issue types.Issue
	if err := client.Patch(ctx, path, req, &issue); err != nil {
		return fmt.Errorf("reopening issue: %w", err)
	}
	fmt.Printf("Issue #%d reopened\n", issueNum)
	return nil
}

func runComment(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("issue number is required")
	}
	owner, repoName, err := resolveRepo(commentFlags.owner, commentFlags.repo)
	if err != nil {
		return err
	}
	var issueNum int
	fmt.Sscanf(args[0], "%d", &issueNum)

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.IssueCommentRequest{Body: commentFlags.body}
	path := api.IssuesComment(owner, repoName, issueNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Post(ctx, path, req, nil); err != nil {
		return fmt.Errorf("commenting on issue: %w", err)
	}
	fmt.Printf("Comment added to issue #%d\n", issueNum)
	return nil
}
