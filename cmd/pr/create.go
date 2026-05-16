package pr

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var createFlags struct {
	owner  string
	repo   string
	title  string
	body   string
	source string
	target string
	draft  bool
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request",
	Example: `  gitee pr create --repo owner/repo --title "Add feature" --source feature --target main
  gitee pr create --repo owner/repo --title "Fix bug" --source fix --target dev --draft`,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVar(&createFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&createFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVarP(&createFlags.title, "title", "t", "", "PR title (required)")
	f.StringVarP(&createFlags.body, "body", "b", "", "PR description")
	f.StringVar(&createFlags.source, "source", "", "Source branch name (required)")
	f.StringVar(&createFlags.target, "target", "", "Target branch name (required)")
	f.BoolVar(&createFlags.draft, "draft", false, "Create as draft PR")
	_ = createCmd.MarkFlagRequired("title")
	_ = createCmd.MarkFlagRequired("source")
	_ = createCmd.MarkFlagRequired("target")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolvePRRepo(createFlags.owner, createFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.PRCreateRequest{
		Title: createFlags.title,
		Body:  createFlags.body,
		Head:  createFlags.source,
		Base:  createFlags.target,
		Draft: createFlags.draft,
	}

	path := api.PRsCreate(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Post(ctx, path, req, &pr); err != nil {
		return fmt.Errorf("creating PR: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s PR #%d created: %s\n", green("✓"), pr.Number, bold(pr.Title))
	fmt.Printf("  URL: %s\n", color.CyanString(pr.HTMLURL))
	return nil
}
