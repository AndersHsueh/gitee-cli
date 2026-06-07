package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var reopenFlags struct {
	owner string
	repo  string
}

var reopenCmd = &cobra.Command{
	Use:     "reopen <number>",
	Short:   "Reopen a pull request",
	Example: `  gitee pr reopen 42 --repo owner/repo`,
	RunE:    runReopen,
}

func init() {
	f := reopenCmd.Flags()
	f.StringVar(&reopenFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&reopenFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runReopen(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(reopenFlags.owner, reopenFlags.repo)
	if err != nil {
		return err
	}

	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: must be an integer", args[0])
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.PRsGet(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	req := map[string]string{"state": "open"}
	var pr types.PullRequest
	if err := client.Patch(ctx, path, req, &pr); err != nil {
		return fmt.Errorf("reopening PR: %w", err)
	}

	fmt.Printf("PR #%d reopened\n", prNum)
	return nil
}
