package pr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
	"github.com/spf13/cobra"
)

var diffFlags struct {
	owner string
	repo  string
}

var diffCmd = &cobra.Command{
	Use:     "diff <number>",
	Short:   "Show pull request diff",
	Example: `  gitee pr diff 42 --repo owner/repo`,
	RunE:    runDiff,
}

func init() {
	f := diffCmd.Flags()
	f.StringVar(&diffFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&diffFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runDiff(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(diffFlags.owner, diffFlags.repo)
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Get(ctx, api.PRsGet(owner, repoName, prNum), &pr); err != nil {
		return fmt.Errorf("getting PR: %w", err)
	}

	if pr.DiffURL == "" {
		return fmt.Errorf("PR #%d has no diff_url", prNum)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pr.DiffURL, nil)
	if err != nil {
		return fmt.Errorf("building diff request: %w", err)
	}

	httpClient := &http.Client{Timeout: time.Duration(cmd.GlobalTimeout) * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching diff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("diff request returned HTTP %d", resp.StatusCode)
	}

	_, err = io.Copy(cobraCmd.OutOrStdout(), resp.Body)
	return err
}
