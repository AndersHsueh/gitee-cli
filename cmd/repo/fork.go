package repo

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

var forkFlags struct {
	owner string
	repo  string
}

var forkCmd = &cobra.Command{
	Use:   "fork [<owner>/<repo>]",
	Short: "Fork a repository",
	Example: `  gitee repo fork owner/repo
  gitee repo fork --repo owner/repo`,
	RunE: runFork,
}

func init() {
	f := forkCmd.Flags()
	f.StringVar(&forkFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&forkFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runFork(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveRepo(forkFlags.owner, forkFlags.repo, args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReposFork(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Post(ctx, path, nil, &repo); err != nil {
		return fmt.Errorf("forking repository: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Forked %s/%s to %s\n", green("✓"), bold(owner), bold(repoName), repo.FullName)
	fmt.Printf("  URL: %s\n", color.CyanString(repo.HTMLURL))
	return nil
}
