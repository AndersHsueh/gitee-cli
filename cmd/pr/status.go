package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Style helpers for table rendering.
var (
	prStatusHeaderStyle = color.New(color.Bold).SprintFunc()
	prStatusDimStyle    = color.New(color.Faint).SprintFunc()
)

// prStatusSection mirrors the section struct from cmd/status for JSON output.
type prStatusSection struct {
	Title string         `json:"title"`
	Count int            `json:"count"`
	Items []prStatusItem `json:"items"`
}

type prStatusItem struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Repo   string `json:"repo"`
	URL    string `json:"url,omitempty"`
	Branch string `json:"branch,omitempty"`
	Author string `json:"author"`
}

var prStatusFlags struct {
	limit int
}

var prStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cross-repo PR status for the current user",
	Long: `Show a summary of pull requests across your Gitee repositories:
PRs awaiting your review and PRs you've opened.

Iterates over your recently-pushed repositories and fetches open
PRs for each one concurrently. Use --limit to control how many
repos to inspect.`,
	Example: `  gitee pr status
  gitee pr status --limit 100
  gitee pr status --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prStatusAgentHelp)
		}
		return nil
	},
	RunE: runPRStatus,
}

func init() {
	f := prStatusCmd.Flags()
	f.IntVarP(&prStatusFlags.limit, "limit", "L", 50, "Max repos to scan (0 for unlimited)")

	PRCmd.AddCommand(prStatusCmd)
}

func runPRStatus(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	username, err := cmd.ResolveCurrentUsername(ctx, client)
	if err != nil {
		return err
	}
	if username == "" {
		return fmt.Errorf("could not determine current user")
	}

	// Fetch up to --limit recently pushed repos the user owns.
	// limit=0 means "no cap" (FetchAllPages semantics).
	repoLimit := prStatusFlags.limit
	qv := url.Values{}
	qv.Set("sort", "pushed")
	qv.Set("direction", "desc")
	qv.Set("type", "owner")
	qv.Set("per_page", "100")
	reposPath := api.ReposList("") + "?" + qv.Encode()

	var repos []types.Repository
	if err := api.FetchAllPages(ctx, client, reposPath, 100, repoLimit, &repos); err != nil {
		return fmt.Errorf("listing repos: %w", err)
	}

	// Concurrently scan each repo for open PRs.
	type repoPRs struct {
		repo string
		prs  []types.PullRequest
	}
	var (
		mu      sync.Mutex
		results []repoPRs
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 5) // max 5 concurrent requests
	)

	for _, r := range repos {
		wg.Add(1)
		go func(owner, name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			q := url.Values{}
			q.Set("state", "open")
			q.Set("per_page", "30")
			prPath := api.PRsList(owner, name) + "?" + q.Encode()

			var prs []types.PullRequest
			if err := client.Get(ctx, prPath, &prs); err != nil {
				return // skip repos that error out
			}
			if len(prs) == 0 {
				return
			}
			mu.Lock()
			results = append(results, repoPRs{repo: owner + "/" + name, prs: prs})
			mu.Unlock()
		}(r.Owner.Login, r.Name)
	}
	wg.Wait()

	// Categorize PRs.
	var awaitingReview, openedByYou []prStatusItem
	for _, r := range results {
		for _, pr := range r.prs {
			item := prStatusItem{
				Number: pr.Number,
				Title:  pr.Title,
				Repo:   r.repo,
				URL:    pr.HTMLURL,
				Author: pr.User.Login,
			}
			if pr.Head != nil {
				item.Branch = pr.Head.Ref
			}
			if pr.User.Login == username {
				openedByYou = append(openedByYou, item)
			} else {
				awaitingReview = append(awaitingReview, item)
			}
		}
	}

	// Render.
	if cmd.GlobalOutput == "json" {
		out := map[string]prStatusSection{
			"awaiting_your_review": {
				Title: "Pull requests awaiting your review",
				Count: len(awaitingReview),
				Items: awaitingReview,
			},
			"opened_by_you": {
				Title: "Pull requests you opened",
				Count: len(openedByYou),
				Items: openedByYou,
			},
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("%s %s\n\n", prStatusDimStyle("current user:"), username)

	if len(awaitingReview) == 0 && len(openedByYou) == 0 {
		fmt.Printf("  %s\n", prStatusDimStyle("no open PRs found across your repos"))
		return nil
	}

	renderPRSection("Pull requests awaiting your review", awaitingReview)
	renderPRSection("Pull requests you opened", openedByYou)

	return nil
}

func renderPRSection(title string, items []prStatusItem) {
	fmt.Printf("%s  %s\n", prStatusHeaderStyle(title), prStatusHeaderStyle(fmt.Sprintf("(%d)", len(items))))
	if len(items) == 0 {
		fmt.Printf("  %s\n\n", prStatusDimStyle("none"))
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPO\t#\tTITLE\tBRANCH")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t#%d\t%s\t%s\n",
			item.Repo, item.Number,
			utils.Truncate(item.Title, 50),
			prStatusDimStyle(item.Branch))
	}
	_ = w.Flush()
	fmt.Println()
}

const prStatusAgentHelp = `# gitee pr status — Agent Help

## Purpose
Show a cross-repository summary of open pull requests for the authenticated
user. Iterates over the user's recently-pushed repositories (up to 50) and
fetches open PRs for each one concurrently (5 parallel requests). Results
are grouped into two sections: PRs you opened, and PRs where you are NOT
the author (awaiting your review).

This is the Gitee equivalent of ` + "`" + `gh pr status` + "`" + `. Unlike ` + "`" + `gitee status` + "`" + `
(which works on a single repo), this command is cross-repo by design.

## Pre-conditions
- Authenticated (token must be valid)
- User has at least one repository on Gitee

## Required flags
None. The command auto-discovers repos.

## Optional flags
  --limit / -L        Max repos to scan (default 50, passed to FetchAllPages)
  --output / -o       table|json (default table)

## Idempotency
YES. Pure read — no state is modified.

## Atomicity
Not applicable (read-only). Partial results may appear if a single repo
errors out (the failure is logged to the goroutine but does not abort
the entire scan).

## Side effects
None. No emails, no webhooks, no notifications.

## Post-conditions (on success)
- Stdout contains a table (or JSON) of open PRs grouped by category.

## Output (stdout, table mode)

  Pull requests awaiting your review  (3)
  REPO                       #     TITLE                                BRANCH
  owner/repo1                42    Fix login bug                       fix/auth
  owner/repo2                17    Add tests                           test/unit

  Pull requests you opened  (1)
  REPO                       #     TITLE                                BRANCH
  owner/repo3                5     Update README                       docs/readme

## Common errors
- 401 Unauthorized: run ` + "`" + `gitee auth login` + "`" + ` or set GITEE_TOKEN
- No repos found: user has no repositories, or token lacks repo scope
- Individual repo errors: silently skipped; the scan continues but the
  failing repo's PRs are absent from the result

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour authenticated (one request per repo + one for repo listing).
With 50 repos: ~51 API calls.

## Race conditions
- A PR merged/closed between the repo listing and the PR fetch may not appear.
  The scan is a point-in-time snapshot, not a real-time feed.
- PRs opened by the user but reviewed by others will appear in "you opened",
  not "awaiting review" — Gitee v5 has no explicit "review-requested" filter
  on the PR list endpoint (same limitation as ` + "`" + `gitee status` + "`" + `).

## Example
  gitee pr status
  gitee pr status --output json | jq '.awaiting_your_review.items[] | {repo, number, title}'

## See also
- ` + "`" + `gitee status` + "`" + `         — single-repo status (issues + PRs in one view)
- ` + "`" + `gitee issue status` + "`" + `    — cross-repo issue summary
- ` + "`" + `gitee pr list` + "`" + `          — list PRs in a specific repo
`
