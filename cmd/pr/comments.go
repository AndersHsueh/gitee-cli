package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
	"github.com/spf13/cobra"
)

var commentsFlags struct {
	owner string
	repo  string
}

var commentsCmd = &cobra.Command{
	Use:     "comments <number>",
	Short:   "List comments on a pull request",
	Example: `  gitee pr comments 42 --repo owner/repo`,
	Args:    cobra.ExactArgs(1),
	RunE:    runComments,
}

func init() {
	f := commentsCmd.Flags()
	f.StringVar(&commentsFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&commentsFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runComments(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolvePRRepo(commentsFlags.owner, commentsFlags.repo)
	if err != nil {
		return err
	}

	var prNum int
	fmt.Sscanf(args[0], "%d", &prNum)

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.PRsComments(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var comments []types.PRComment
	if err := client.Get(ctx, path, &comments); err != nil {
		return fmt.Errorf("listing PR comments: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(comments, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Comments on PR #%d:\n", prNum)
	for _, c := range comments {
		fmt.Printf("  [%s] %s: %s\n", c.CreatedAt, c.User.Login, c.Body)
	}
	return nil
}
