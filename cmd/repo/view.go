package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
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
		fmt.Fprintf(w, "Pushed At:\t%s\n", repo.PushedAt[:10])
	}
	if repo.CreatedAt != "" {
		fmt.Fprintf(w, "Created At:\t%s\n", repo.CreatedAt[:10])
	}
	return w.Flush()
}

// resolveRepo returns owner and repoName from flags or args.
func resolveRepo(owner, repo string, args []string) (string, string, error) {
	if len(args) > 0 {
		o, r := splitOwnerRepo(args[0])
		if o != "" && r != "" {
			return o, r, nil
		}
	}
	if owner != "" && repo != "" {
		return owner, repo, nil
	}
	combined := repo
	if combined == "" {
		combined = cmd.GlobalRepo
	}
	if combined == "" {
		return "", "", fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}
	o, r := splitOwnerRepo(combined)
	return o, r, nil
}

func splitOwnerRepo(input string) (owner, repo string) {
	for i := 0; i < len(input); i++ {
		if input[i] == '/' {
			return input[:i], input[i+1:]
		}
	}
	return "", input
}
