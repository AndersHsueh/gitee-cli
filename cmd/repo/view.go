package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var viewFlags struct {
	owner string
	repo  string
}

var viewCmd = &cobra.Command{
	Use:   "view [<owner>/<repo>]",
	Short: "View repository details",
	Example: `  gitee repo view owner/repo
  gitee repo view --repo owner/repo`,
	RunE: runView,
}

func init() {
	f := viewCmd.Flags()
	f.StringVar(&viewFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&viewFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveRepo(viewFlags.owner, viewFlags.repo, args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReposGet(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Get(ctx, path, &repo); err != nil {
		return fmt.Errorf("viewing repository: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(repo, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	vis := "public"
	if repo.Private {
		vis = "private"
	}
	fmt.Fprintf(w, "Name:\t%s\n", repo.FullName)
	fmt.Fprintf(w, "Description:\t%s\n", repo.Description)
	fmt.Fprintf(w, "Visibility:\t%s\n", vis)
	fmt.Fprintf(w, "Language:\t%s\n", repo.Language)
	fmt.Fprintf(w, "Stars:\t%d\n", repo.StarsCount)
	fmt.Fprintf(w, "Forks:\t%d\n", repo.ForksCount)
	fmt.Fprintf(w, "Watchers:\t%d\n", repo.WatchersCount)
	fmt.Fprintf(w, "Default Branch:\t%s\n", repo.DefaultBranch)
	fmt.Fprintf(w, "Open Issues:\t%d\n", repo.OpenIssuesCount)
	fmt.Fprintf(w, "URL:\t%s\n", repo.HTMLURL)
	if repo.PushedAt != "" {
		fmt.Fprintf(w, "Pushed At:\t%s\n", utils.TruncateDate(repo.PushedAt))
	}
	if repo.CreatedAt != "" {
		fmt.Fprintf(w, "Created At:\t%s\n", utils.TruncateDate(repo.CreatedAt))
	}
	return w.Flush()
}

func resolveRepo(owner, repo string, args []string) (string, string, error) {
	return utils.ResolveRepo(owner, repo, args)
}
