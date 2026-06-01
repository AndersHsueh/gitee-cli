package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
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
	head  string
	base  string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Example: `  gitee pr list --repo owner/repo
  gitee pr list --repo owner/repo --state closed
  gitee pr list --repo owner/repo --head user:feature
  gitee pr list --repo owner/repo --paginate             # fetch ALL PRs
  gitee pr list --repo owner/repo --paginate --limit 0   # same, explicit`,
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.StringVar(&listFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&listFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVar(&listFlags.state, "state", "open", "Filter by state: open|closed|merged|all")
	f.StringVar(&listFlags.head, "head", "", "Filter by head branch (user:ref)")
	f.StringVar(&listFlags.base, "base", "", "Filter by base branch")
	// --limit / --per-page / --paginate are global persistent flags.
	// See cmd/root.go for their definitions. --limit is a TOTAL cap
	// across pages when --paginate is set (was a per-page size, which
	// was a bug — see v0.4.1 changelog).
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

	// Build base query (without page/per_page — FetchAllPages handles those)
	qv := url.Values{}
	qv.Set("state", listFlags.state)
	if listFlags.head != "" {
		qv.Set("head", listFlags.head)
	}
	if listFlags.base != "" {
		qv.Set("base", listFlags.base)
	}
	basePath := api.PRsList(owner, repoName) + "?" + qv.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	perPage := cmd.GlobalPerPage
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	var prs []types.PullRequest
	if cmd.GlobalPaginate {
		if err := api.FetchAllPages(ctx, client, basePath, perPage, cmd.GlobalLimit, &prs); err != nil {
			return fmt.Errorf("listing PRs: %w", err)
		}
	} else {
		// Single page (legacy behavior). --page and --per-page apply directly.
		pagePath, err := addPRPageParams(basePath, cmd.GlobalPage, perPage)
		if err != nil {
			return fmt.Errorf("building page URL: %w", err)
		}
		if err := client.Get(ctx, pagePath, &prs); err != nil {
			// Gitee API returns 404 when there are no PRs, treat as empty result
			if api.IsNotFound(err) {
				prs = []types.PullRequest{}
			} else {
				return fmt.Errorf("listing PRs: %w", err)
			}
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
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	return utils.ResolveRepo(owner, repo, nil)
}

// addPRPageParams is a local helper because the issue package's
// addPageParams is unexported. It does the same thing.
func addPRPageParams(basePath string, page, perPage int) (string, error) {
	u, err := url.Parse(basePath)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	u.RawQuery = q.Encode()
	return u.String(), nil
}
