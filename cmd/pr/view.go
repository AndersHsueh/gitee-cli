package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var viewFlags struct {
	owner string
	repo  string
}

var viewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View pull request details",
	Example: `  gitee pr view 42 --repo owner/repo
  gitee pr view 42`,
	RunE: runView,
}

func init() {
	f := viewCmd.Flags()
	f.StringVar(&viewFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&viewFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(viewFlags.owner, viewFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: must be an integer", args[0])
	}

	path := api.PRsGet(owner, repoName, prNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Get(ctx, path, &pr); err != nil {
		return fmt.Errorf("viewing PR: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(pr, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	state := pr.State
	if pr.Merged {
		state = "merged"
	}
	fmt.Fprintf(w, "Title:\t%s\n", pr.Title)
	fmt.Fprintf(w, "State:\t%s\n", state)
	fmt.Fprintf(w, "Author:\t%s\n", pr.User.Login)
	fmt.Fprintf(w, "Base:\t%s\n", pr.Base.Ref)
	fmt.Fprintf(w, "Head:\t%s\n", pr.Head.Ref)
	fmt.Fprintf(w, "URL:\t%s\n", pr.HTMLURL)
	if pr.Body != "" {
		fmt.Fprintf(w, "\n%s\n", pr.Body)
	}
	if pr.Comments > 0 || pr.Commits > 0 || pr.Additions > 0 {
		fmt.Fprintf(w, "\nStats:\t+%d/-%d in %d commits with %d comments\n",
			pr.Additions, pr.Deletions, pr.Commits, pr.Comments)
	}
	return w.Flush()
}
