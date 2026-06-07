package status

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
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Style helpers — constructed once at package init rather than per render
// call. Per-call construction is sub-millisecond wasted, but consistent
// house style is to hoist them (see cmd/auth/status.go).
var (
	headerStyle = color.New(color.Bold).SprintFunc()
	cyanStyle   = color.New(color.FgCyan).SprintFunc()
	dimStyle    = color.New(color.Faint).SprintFunc()
)

// errNotInWorkTree is returned by resolveCurrentRepo when the user runs
// `gitee status` outside a git working tree. The caller treats it the
// same as any other resolve error (print message, exit 0) but the
// dedicated sentinel keeps the message specific to that case.
var errNotInWorkTree = fmt.Errorf("not inside a git working tree. Use --repo owner/repo to specify a repository")

// resolveCurrentRepo returns (owner, repo) on success, or a non-nil error
// describing why resolution failed. The caller should print the error
// message to stderr and exit 0 — we do NOT treat resolution failure as
// a fatal error because the user might just be running `gitee status`
// in /tmp or a non-checked-out project.
//
// Resolution order:
//  1. --repo flag (global) — user explicit
//  2. `git config --local gitee.default_repo` (v0.10.0)
//  3. `git remote get-url origin` parsed for owner/repo
func resolveCurrentRepo() (owner, repo string, err error) {
	if cmd.GlobalRepo != "" {
		o, r := utils.ParseOwnerRepo(cmd.GlobalRepo)
		if o != "" && r != "" {
			return o, r, nil
		}
		return "", "", fmt.Errorf("--repo=%q is not in owner/repo format", cmd.GlobalRepo)
	}
	// Check git local config before falling back to git remote (v0.10.0)
	if defaultRepo, cfgErr := utils.GitConfigGet("gitee.default_repo"); cfgErr == nil && defaultRepo != "" {
		o, r := utils.ParseOwnerRepo(defaultRepo)
		if o != "" && r != "" {
			return o, r, nil
		}
	}
	if !utils.IsInsideWorkTree() {
		return "", "", errNotInWorkTree
	}
	remote, gerr := utils.GitRemoteURL("origin")
	if gerr != nil {
		return "", "", fmt.Errorf("no 'origin' remote configured: %v", gerr)
	}
	o, r, perr := utils.ParseGiteeRemoteURL(remote)
	if perr != nil {
		return "", "", fmt.Errorf("origin %q is not a Gitee repository: %v", remote, perr)
	}
	return o, r, nil
}

// section is the result of one logical section in the status output.
// We use a small struct to keep the runStatus body linear and to make
// the JSON output predictable. Count is intentionally a separate field
// (not derived from len(Items)) so JSON consumers can read it without
// having to walk the items array.
type section struct {
	Title string        `json:"title"`
	Count int           `json:"count"`
	Items []sectionItem `json:"items"`
}

// sectionItem.Number is `any` (not int/string) because the Gitee v5 issue
// API returns Number as a STRING while the PR API returns it as an int.
// We preserve the source type so an agent parsing the JSON can detect
// this asymmetry without us silently coercing and hiding it.
type sectionItem struct {
	Number any    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url,omitempty"`
	Branch string `json:"branch,omitempty"`
}

func runStatus(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveCurrentRepo()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil // exit 0 — see resolveCurrentRepo doc comment
	}

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
		return fmt.Errorf("could not determine current user (login is empty in /user response)")
	}

	issuesAssigned, err := fetchListAssigned(ctx, client, owner, repoName, username)
	if err != nil {
		return err
	}
	prsOpen, err := fetchListOpenPRs(ctx, client, owner, repoName)
	if err != nil {
		return err
	}
	prsForReview := filterPRsAwaitingReview(prsOpen, username)

	return render(owner, repoName, username, issuesAssigned, prsForReview, len(prsOpen))
}

// fetchListAssigned returns open issues in the repo where the current
// user is the assignee. Gitee v5's /repos/{o}/{r}/issues supports
// `?assignee=<username>` as a server-side filter, so we don't have to
// fetch everything and filter.
func fetchListAssigned(ctx context.Context, client *api.Client, owner, repo, username string) ([]types.Issue, error) {
	qv := url.Values{}
	qv.Set("state", "open")
	qv.Set("assignee", username)
	path := api.IssuesList(owner, repo) + "?" + qv.Encode()
	return api.FetchListOrAll[types.Issue](ctx, client, path, cmd.GlobalPaginate,
		cmd.GlobalPerPage, cmd.GlobalLimit, cmd.GlobalPage,
		"listing issues assigned to you")
}

// fetchListOpenPRs returns all open PRs in the repo. Gitee v5 has no
// ?reviewer=<me> filter on /repos/{o}/{r}/pulls (unlike GitHub), so we
// fetch everything and filter client-side. This is the documented
// "limitation" of the Gitee equivalent of `gh status` — see
// cmd/status/status_agenthelp.go "Known limitations" section.
func fetchListOpenPRs(ctx context.Context, client *api.Client, owner, repo string) ([]types.PullRequest, error) {
	qv := url.Values{}
	qv.Set("state", "open")
	path := api.PRsList(owner, repo) + "?" + qv.Encode()
	return api.FetchListOrAll[types.PullRequest](ctx, client, path, cmd.GlobalPaginate,
		cmd.GlobalPerPage, cmd.GlobalLimit, cmd.GlobalPage,
		"listing open PRs")
}

// filterPRsAwaitingReview returns the subset of `prs` where the current
// user is NOT the author (i.e. someone else is asking for the user's
// review). We don't have access to "has this user already reviewed" from
// the PR list endpoint, so a PR where the user reviewed and the author
// updated will still appear here. This matches `gh status`'s
// "review-requested" approximation.
func filterPRsAwaitingReview(prs []types.PullRequest, username string) []types.PullRequest {
	out := make([]types.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if pr.User.Login != username {
			out = append(out, pr)
		}
	}
	return out
}

func render(owner, repo, username string, assigned []types.Issue, review []types.PullRequest, openTotal int) error {
	// JSON mode: emit a single machine-readable object so agents can
	// parse it without scraping color codes or table borders.
	if cmd.GlobalOutput == "json" {
		out := struct {
			Repo     string  `json:"repo"`
			User     string  `json:"user"`
			Assigned section `json:"assigned_to_you"`
			Review   section `json:"awaiting_review"`
			OpenPRs  int     `json:"open_prs_total"`
		}{
			Repo: fmt.Sprintf("%s/%s", owner, repo),
			User: username,
			Assigned: section{
				Title: "Issues assigned to you",
				Count: len(assigned),
				Items: issuesToItems(assigned),
			},
			Review: section{
				Title: "PRs awaiting your review",
				Count: len(review),
				Items: prsToItems(review),
			},
			OpenPRs: openTotal,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	// Human-readable table mode.
	if branch := utils.CurrentBranch(); branch != "" {
		fmt.Printf("%s  on branch %s\n", cyanStyle(fmt.Sprintf("%s/%s", owner, repo)), cyanStyle(branch))
	} else {
		fmt.Printf("%s\n", cyanStyle(fmt.Sprintf("%s/%s", owner, repo)))
	}
	fmt.Printf("%s %s\n\n", dimStyle("current user:"), username)

	renderSection("Issues assigned to you", len(assigned), func(w *tabwriter.Writer) {
		for _, i := range assigned {
			fmt.Fprintf(w, "#%s\t%s\n", i.Number, utils.Truncate(i.Title, 60))
		}
	})

	renderSection("PRs awaiting your review", len(review), func(w *tabwriter.Writer) {
		for _, pr := range review {
			branch := ""
			if pr.Head != nil {
				branch = pr.Head.Ref
			}
			fmt.Fprintf(w, "#%d\t%s\t%s\n", pr.Number, utils.Truncate(pr.Title, 60), dimStyle(branch))
		}
	})

	if len(review) == 0 && openTotal > 0 {
		fmt.Printf("  %s %d open PR(s) in this repo, all authored by you.\n", dimStyle("note:"), openTotal)
	}
	return nil
}

func renderSection(title string, count int, body func(w *tabwriter.Writer)) {
	fmt.Printf("%s  %s\n", headerStyle(title), headerStyle(fmt.Sprintf("(%d)", count)))
	if count == 0 {
		fmt.Printf("  %s\n\n", dimStyle("none"))
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	body(w)
	_ = w.Flush()
	fmt.Println()
}

func issuesToItems(issues []types.Issue) []sectionItem {
	out := make([]sectionItem, 0, len(issues))
	for _, i := range issues {
		// Preserve the source type: Gitee v5 issues return Number as a STRING,
		// while PRs return it as an int. `any` lets the JSON encoder emit
		// the original type without coercion — see sectionItem.Number doc.
		out = append(out, sectionItem{Number: i.Number, Title: i.Title, URL: i.HTMLURL})
	}
	return out
}

func prsToItems(prs []types.PullRequest) []sectionItem {
	out := make([]sectionItem, 0, len(prs))
	for _, p := range prs {
		branch := ""
		if p.Head != nil {
			branch = p.Head.Ref
		}
		out = append(out, sectionItem{Number: p.Number, Title: p.Title, URL: p.HTMLURL, Branch: branch})
	}
	return out
}
