package pr

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
	"github.com/spf13/cobra"
)

var reviewFlags struct {
	owner          string
	repo           string
	approve        bool
	requestChanges bool
	comment        bool
	body           string
}

var reviewCmd = &cobra.Command{
	Use:   "review <number>",
	Short: "Review a pull request",
	Example: `  gitee pr review 42 --repo owner/repo --approve
  gitee pr review 42 --repo owner/repo --request-changes --body "Needs tests"`,
	RunE: runReview,
}

func init() {
	f := reviewCmd.Flags()
	f.StringVar(&reviewFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&reviewFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.BoolVar(&reviewFlags.approve, "approve", false, "Approve the PR")
	f.BoolVar(&reviewFlags.requestChanges, "request-changes", false, "Request changes on the PR")
	f.BoolVar(&reviewFlags.comment, "comment", false, "Comment on the PR")
	f.StringVarP(&reviewFlags.body, "body", "b", "", "Review comment")
}

func runReview(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(reviewFlags.owner, reviewFlags.repo)
	if err != nil {
		return err
	}

	var prNum int
	fmt.Sscanf(args[0], "%d", &prNum)

	// Determine review event
	event := "COMMENT"
	if reviewFlags.approve {
		event = "APPROVE"
	} else if reviewFlags.requestChanges {
		event = "REQUEST_CHANGES"
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.PRReviewRequest{
		Body:  reviewFlags.body,
		Event: event,
	}

	path := api.PRsReview(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Post(ctx, path, req, nil); err != nil {
		return fmt.Errorf("reviewing PR: %w", err)
	}

	fmt.Printf("PR #%d reviewed: %s\n", prNum, event)
	return nil
}
