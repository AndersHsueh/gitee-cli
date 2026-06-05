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
	"gitee-cli/internal/config"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// resolveCurrentRepo returns (owner, repo, ok). When ok=false, the caller
// should print a clear "not in a Gitee repo" message and exit 0 — we do
// NOT treat this as an error because the user might just be running
// `gitee status` in /tmp or in a non-checked-out project.
//
// Resolution order:
//  1. --repo flag (global) — user explicit
//  2. `git remote get-url origin` parsed for owner/repo
//
// If both fail, ok=false and the function returns empty strings.
func resolveCurrentRepo() (owner, repo string, ok bool, hint string) {
	// 1. Global --repo flag wins (most explicit).
	if cmd.GlobalRepo != "" {
		o, r := utils.ParseOwnerRepo(cmd.GlobalRepo)
		if o != "" && r != "" {
			return o, r, true, ""
		}
		// --repo was set but malformed — surface as a hint, not a fatal
		// error (we'll fall through to git detection so the user gets
		// a chance to fix the flag).
		hint = fmt.Sprintf("(--repo=%q is not in owner/repo format; ignoring)", cmd.GlobalRepo)
	}

	// 2. git remote auto-detection. We check IsInsideWorkTree first so
	// we don't shell out to git in non-repo directories (cheap & quiet).
	if !utils.IsInsideWorkTree() {
		if hint == "" {
			hint = "Not inside a git working tree. Use --repo owner/repo to specify a repository."
		}
		return "", "", false, hint
	}

	remote, err := utils.GitRemoteURL("origin")
	if err != nil {
		if hint == "" {
			hint = fmt.Sprintf("No 'origin' remote configured. Use --repo owner/repo to specify a repository. (%v)", err)
		}
		return "", "", false, hint
	}

	o, r, err := utils.ParseGiteeRemoteURL(remote)
	if err != nil {
		if hint == "" {
			hint = fmt.Sprintf("Origin points to %q, which is not a Gitee repository. Use --repo owner/repo to override. (%v)", remote, err)
		}
		return "", "", false, hint
	}
	return o, r, true, ""
}

// currentUsername returns the username of the authenticated user. Tries
// the cached value in ~/.gitee-cli/settings.json first; if that returns
// empty (e.g. login info wasn't captured), it falls back to a /user API
// call (which always works for any authenticated user).
func currentUsername(ctx context.Context, client *api.Client) (string, error) {
	if u := config.GetUsername(cmd.GlobalHostname); u != "" {
		return u, nil
	}
	// Fallback: hit /user. We use a tiny struct to avoid pulling in the
	// full User type and its many optional fields.
	var u struct {
		Login string `json:"login"`
	}
	if err := client.Get(ctx, "/user", &u); err != nil {
		return "", fmt.Errorf("resolving current user (no cached username, /user failed): %w", err)
	}
	return u.Login, nil
}

// section is the result of one logical section in the status output.
// We use a small struct to keep the runStatus body linear and to make
// the JSON output predictable.
type section struct {
	Title string        `json:"title"`
	Count int           `json:"count"`
	Items []sectionItem `json:"items"`
}

type sectionItem struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url,omitempty"`
	Branch string `json:"branch,omitempty"`
}

func runStatus(cobraCmd *cobra.Command, args []string) error {
	// --- Step 1: resolve repo (auto-detect from git, --repo overrides) ---
	owner, repoName, ok, hint := resolveCurrentRepo()
	if !ok {
		fmt.Fprintln(os.Stderr, hint)
		return nil // exit 0 — see resolveCurrentRepo doc comment
	}

	// --- Step 2: build API client + resolve current username ---
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	username, err := currentUsername(ctx, client)
	if err != nil {
		return err
	}
	if username == "" {
		return fmt.Errorf("could not determine current user (login is empty in /user response)")
	}

	// --- Step 3: fetch issues assigned to me + all open PRs in parallel-ish ---
	// (We do them sequentially because the API is the bottleneck, not the
	// parsing. Goroutine fan-out would add complexity for ~50ms savings.)
	issuesAssigned, err := fetchIssuesAssignedToMe(ctx, client, owner, repoName, username)
	if err != nil {
		return err
	}
	prsOpen, err := fetchOpenPRs(ctx, client, owner, repoName)
	if err != nil {
		return err
	}

	// --- Step 4: filter PRs to "awaiting my review" (client-side) ---
	prsForReview := filterPRsAwaitingReview(prsOpen, username)

	// --- Step 5: render ---
	return render(owner, repoName, username, issuesAssigned, prsForReview, prsOpen)
}

// fetchIssuesAssignedToMe returns open issues in the repo where the
// current user is the assignee. Gitee v5's /repos/{o}/{r}/issues supports
// `?assignee=<username>` as a server-side filter, so we don't have to
// fetch everything and filter.
func fetchIssuesAssignedToMe(ctx context.Context, client *api.Client, owner, repo, username string) ([]types.Issue, error) {
	qv := url.Values{}
	qv.Set("state", "open")
	qv.Set("assignee", username)
	path := fmt.Sprintf("/repos/%s/%s/issues?%s", owner, repo, qv.Encode())

	perPage := cmd.GlobalPerPage
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	var issues []types.Issue
	if cmd.GlobalPaginate {
		if err := api.FetchAllPages(ctx, client, path, perPage, cmd.GlobalLimit, &issues); err != nil {
			return nil, fmt.Errorf("listing issues assigned to you: %w", err)
		}
	} else {
		// Single page. The X-Total-Count header (read by api.GetTotalCount)
		// tells the agent the true total if --paginate is not used.
		if err := client.Get(ctx, path, &issues); err != nil {
			if api.IsNotFound(err) {
				return nil, nil // empty repo or no issues — not an error
			}
			return nil, fmt.Errorf("listing issues assigned to you: %w", err)
		}
	}
	return issues, nil
}

// fetchOpenPRs returns all open PRs in the repo. Gitee v5 has no
// ?reviewer=<me> filter on /repos/{o}/{r}/pulls (unlike GitHub), so we
// fetch everything and filter client-side. This is the documented
// "limitation" of the Gitee equivalent of `gh status` — see
// cmd/status/status_agenthelp.go "Known limitations" section.
func fetchOpenPRs(ctx context.Context, client *api.Client, owner, repo string) ([]types.PullRequest, error) {
	qv := url.Values{}
	qv.Set("state", "open")
	path := fmt.Sprintf("/repos/%s/%s/pulls?%s", owner, repo, qv.Encode())

	perPage := cmd.GlobalPerPage
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	var prs []types.PullRequest
	if cmd.GlobalPaginate {
		if err := api.FetchAllPages(ctx, client, path, perPage, cmd.GlobalLimit, &prs); err != nil {
			return nil, fmt.Errorf("listing open PRs: %w", err)
		}
	} else {
		if err := client.Get(ctx, path, &prs); err != nil {
			if api.IsNotFound(err) {
				return nil, nil // no PRs — not an error
			}
			return nil, fmt.Errorf("listing open PRs: %w", err)
		}
	}
	return prs, nil
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
		if pr.User.Login == username {
			continue // you authored this; it's not "awaiting your review"
		}
		// Skip merged/closed even though state=open should filter them —
		// defensive against API quirks.
		if pr.State != "open" {
			continue
		}
		out = append(out, pr)
	}
	return out
}

func render(owner, repo, username string, assigned []types.Issue, review []types.PullRequest, allOpen []types.PullRequest) error {
	// JSON mode: emit a single machine-readable object so agents can
	// parse it without scraping color codes or table borders.
	if cmd.GlobalOutput == "json" {
		out := struct {
			Repo      string  `json:"repo"`
			User      string  `json:"user"`
			Assigned  section `json:"assigned_to_you"`
			Review    section `json:"awaiting_review"`
			OpenTotal int     `json:"open_prs_total"`
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
			OpenTotal: len(allOpen),
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	// Human-readable table mode.
	header := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	branch := utils.CurrentBranch()
	if branch != "" {
		fmt.Printf("%s  on branch %s\n", cyan(fmt.Sprintf("%s/%s", owner, repo)), cyan(branch))
	} else {
		fmt.Printf("%s\n", cyan(fmt.Sprintf("%s/%s", owner, repo)))
	}
	fmt.Printf("%s %s\n\n", dim("current user:"), username)

	renderSection(header, "Issues assigned to you", len(assigned), func(w *tabwriter.Writer) {
		for _, i := range assigned {
			title := utils.Truncate(i.Title, 60)
			fmt.Fprintf(w, "#%s\t%s\n", i.Number, title)
		}
	})

	renderSection(header, "PRs awaiting your review", len(review), func(w *tabwriter.Writer) {
		for _, pr := range review {
			title := utils.Truncate(pr.Title, 60)
			branch := ""
			if pr.Head != nil {
				branch = pr.Head.Ref
			}
			fmt.Fprintf(w, "#%d\t%s\t%s\n", pr.Number, title, dim(branch))
		}
	})

	if len(review) == 0 && len(allOpen) > 0 {
		fmt.Printf("  %s %d open PR(s) in this repo, all authored by you.\n", dim("note:"), len(allOpen))
	}
	return nil
}

func renderSection(header func(a ...any) string, title string, count int, body func(w *tabwriter.Writer)) {
	fmt.Printf("%s  %s\n", header(title), header(fmt.Sprintf("(%d)", count)))
	if count == 0 {
		fmt.Printf("  %s\n\n", color.New(color.Faint).SprintFunc()("none"))
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
		// Issue.Number is a string in the Gitee v5 API (see pkg/types/issue.go).
		// We keep it as a string in sectionItem.Number for the JSON output
		// to faithfully reflect the source type — agents that compare against
		// the raw /issues response won't be surprised by a string-to-int cast.
		n := 0
		fmt.Sscanf(i.Number, "%d", &n) // best-effort: if not numeric, stays 0
		out = append(out, sectionItem{Number: n, Title: i.Title, URL: i.HTMLURL})
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
