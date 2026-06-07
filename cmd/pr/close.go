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

var closeFlags struct {
	owner string
	repo  string
}

var closeCmd = &cobra.Command{
	Use:     "close <number>",
	Short:   "Close a pull request",
	Example: `  gitee pr close 42 --repo owner/repo`,
	RunE:    runClose,
}

func init() {
	f := closeCmd.Flags()
	f.StringVar(&closeFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&closeFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runClose(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(closeFlags.owner, closeFlags.repo)
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

	req := map[string]string{"state": "closed"}
	var pr types.PullRequest
	if err := client.Patch(ctx, path, req, &pr); err != nil {
		return fmt.Errorf("closing PR: %w", err)
	}

	fmt.Printf("PR #%d closed\n", prNum)
	return nil
}
