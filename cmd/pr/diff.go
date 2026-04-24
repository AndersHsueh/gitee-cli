package pr

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
)

var diffFlags struct {
	owner string
	repo  string
}

var diffCmd = &cobra.Command{
	Use:   "diff <number>",
	Short: "Show pull request diff",
	Example: `  gitee pr diff 42 --repo owner/repo`,
	RunE: runDiff,
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

	var prNum int
	fmt.Sscanf(args[0], "%d", &prNum)

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.PRsGet(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// Get raw response (which will be JSON containing diff_url)
	body, _, err := client.GetRaw(ctx, path)
	if err != nil {
		return fmt.Errorf("getting PR diff: %w", err)
	}

	// Output the raw response (could be diff or JSON)
	io.WriteString(cobraCmd.OutOrStdout(), string(body))
	return nil
}
