package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var mergeFlags struct {
	owner     string
	repo      string
	squash    bool
	squashMsg string
}

var mergeCmd = &cobra.Command{
	Use:   "merge <number>",
	Short: "Merge a pull request",
	Example: `  gitee pr merge 42 --repo owner/repo
  gitee pr merge 42 --repo owner/repo --squash`,
	RunE: runMerge,
}

func init() {
	f := mergeCmd.Flags()
	f.StringVar(&mergeFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&mergeFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.BoolVar(&mergeFlags.squash, "squash", false, "Squash commits when merging")
	f.StringVar(&mergeFlags.squashMsg, "squash-message", "", "Commit message for squash merge")
}

func runMerge(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(mergeFlags.owner, mergeFlags.repo)
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

	req := types.PRMergeRequest{
		Squash:        mergeFlags.squash,
		SquashMessage: mergeFlags.squashMsg,
	}

	path := api.PRsMerge(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var result struct {
		Merged bool `json:"merged"`
	}
	if err := client.Put(ctx, path, req, &result); err != nil {
		return fmt.Errorf("merging PR: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	if result.Merged {
		fmt.Printf("%s PR #%d merged\n", green("✓"), prNum)
	} else {
		fmt.Printf("PR #%d merge status: %v\n", prNum, result.Merged)
	}
	return nil
}
