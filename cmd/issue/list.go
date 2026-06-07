package issue

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
	state     string
	labels    string
	owner     string
	assignee  string
	milestone string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues in a repository",
	Example: `  gitee issue list --repo owner/repo
  gitee issue list --repo owner/repo --state closed --labels bug
  gitee issue list --repo owner/repo --paginate          # fetch ALL open issues
  gitee issue list --repo owner/repo --paginate --limit 0  # same, explicit`,
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.StringVar(&listFlags.state, "state", "open", "Filter by state: open|closed|all")
	f.StringVar(&listFlags.labels, "labels", "", "Filter by labels (comma-separated)")
	f.StringVar(&listFlags.owner, "owner", "", "Repository owner")
	f.StringVar(&listFlags.assignee, "assignee", "", "Filter by assignee username")
	f.StringVar(&listFlags.milestone, "milestone", "", "Filter by milestone")
	// --limit / --per-page / --paginate are global persistent flags.
	// See cmd/root.go for their definitions. --limit here is a TOTAL cap
	// across pages when --paginate is set (was previously a per-page size,
	// which was a bug — see v0.4.1 changelog).
}

func runList(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveRepo(listFlags.owner, cmd.GlobalRepo)
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
	if listFlags.labels != "" {
		qv.Set("labels", listFlags.labels)
	}
	if listFlags.assignee != "" {
		qv.Set("assignee", listFlags.assignee)
	}
	if listFlags.milestone != "" {
		qv.Set("milestone", listFlags.milestone)
	}
	basePath := fmt.Sprintf("/repos/%s/%s/issues?%s", owner, repoName, qv.Encode())

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	perPage := cmd.GlobalPerPage
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	var issues []types.Issue
	if cmd.GlobalPaginate {
		// Walk all pages up to --limit. The agent use case: `gitee issue list
		// --paginate --limit 0` fetches everything, which is what the AI
		// pipeline needs to see newly-created issues that landed past page 1.
		if err := api.FetchAllPages(ctx, client, basePath, perPage, cmd.GlobalLimit, &issues); err != nil {
			return fmt.Errorf("listing issues: %w", err)
		}
	} else {
		// Single page (legacy behavior). --page and --per-page apply directly.
		// --limit is ignored when not paginating (a single page either fits
		// within perPage or it doesn't — there's no "total" to cap).
		pagePath, err := addPageParams(basePath, cmd.GlobalPage, perPage)
		if err != nil {
			return fmt.Errorf("building page URL: %w", err)
		}
		if err := client.Get(ctx, pagePath, &issues); err != nil {
			return fmt.Errorf("listing issues: %w", err)
		}
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(issues, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(issues) == 0 {
		fmt.Printf("No %s issues found in %s/%s\n", listFlags.state, owner, repoName)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tTITLE\tSTATE\tLABELS\tUPDATED")
	for _, i := range issues {
		lbls := make([]string, 0, len(i.Labels))
		for _, l := range i.Labels {
			lbls = append(lbls, l.Name)
		}
		labelStr := strings.Join(lbls, ",")
		if labelStr == "" {
			labelStr = "-"
		}
		title := utils.Truncate(i.Title, 60)
		updated := i.UpdatedAt.Format("2006-01-02")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", i.Number, title, i.State, labelStr, updated)
	}
	_ = w.Flush()
	return nil
}

// addPageParams appends/overrides ?page= and ?per_page= on a base path that
// may already contain a query string. It mirrors what FetchAllPages does
// internally but is exported so the single-page path can use the same
// query-building logic.
func addPageParams(basePath string, page, perPage int) (string, error) {
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
