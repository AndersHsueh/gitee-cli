package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"
	"github.com/spf13/cobra"
)

var listFlags struct {
	owner string
	repo  string
	state string
	limit int
	head  string
	base  string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Example: `  gitee pr list --repo owner/repo
  gitee pr list --repo owner/repo --state closed
  gitee pr list --repo owner/repo --head user:feature`,
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.StringVar(&listFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&listFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVar(&listFlags.state, "state", "open", "Filter by state: open|closed|merged|all")
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of PRs")
	f.StringVar(&listFlags.head, "head", "", "Filter by head branch (user:ref)")
	f.StringVar(&listFlags.base, "base", "", "Filter by base branch")
}

func runList(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolvePRRepo(listFlags.owner, listFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	qv := url.Values{}
	qv.Set("state", listFlags.state)
	qv.Set("per_page", fmt.Sprintf("%d", listFlags.limit))
	if listFlags.head != "" {
		qv.Set("head", listFlags.head)
	}
	if listFlags.base != "" {
		qv.Set("base", listFlags.base)
	}
	path := api.PRsList(owner, repoName) + "?" + qv.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var prs []types.PullRequest
	if err := client.Get(ctx, path, &prs); err != nil {
		// Gitee API returns 404 when there are no PRs, treat as empty result
		if isNotFoundError(err) {
			prs = []types.PullRequest{}
		} else {
			return fmt.Errorf("listing PRs: %w", err)
		}
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(prs, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(prs) == 0 {
		fmt.Println("No pull requests found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tTITLE\tSTATE\tHEAD→BASE\tAUTHOR\tUPDATED")
	for _, pr := range prs {
		headToBase := ""
		if pr.Head != nil && pr.Base != nil {
			headToBase = fmt.Sprintf("%s→%s", pr.Head.Ref, pr.Base.Ref)
		}
		state := pr.State
		if pr.Merged {
			state = "merged"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			pr.Number, pr.Title, state, headToBase, pr.User.Login, utils.TruncateDate(pr.UpdatedAt))
	}
	return w.Flush()
}

func resolvePRRepo(owner, repo string) (string, string, error) {
	combined := repo
	if combined == "" {
		combined = cmd.GlobalRepo
	}
	if combined == "" {
		return "", "", fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}
	o, r := utils.ParseOwnerRepo(combined)
	if o == "" || r == "" {
		return "", "", fmt.Errorf("invalid --repo format %q, expected owner/repo", combined)
	}
	return o, r, nil
}

// isNotFoundError checks if the error is an HTTP 404 not found error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "404") && strings.Contains(errStr, "not found")
}
