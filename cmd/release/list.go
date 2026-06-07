package release

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/handler"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var listFlags struct {
	repo  string
	limit int
	page  int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List releases in a repository",
	Example: `  gitee release list --repo owner/repo
  gitee release list --repo owner/repo --limit 50`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(releaseListAgentHelp)
		}
		return nil
	},
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.StringVarP(&listFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of releases to return")
	f.IntVar(&listFlags.page, "page", 1, "Page number (1-based)")
}

func runList(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveReleaseRepo(listFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReleasesList(owner, repoName)
	q := url.Values{}
	q.Set("per_page", fmt.Sprintf("%d", listFlags.limit))
	q.Set("page", fmt.Sprintf("%d", listFlags.page))
	fullPath := path + "?" + q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var releases []types.Release
	if err := client.Get(ctx, fullPath, &releases); err != nil {
		return fmt.Errorf("listing releases: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return handler.PrintJSON(releases)
	}

	if len(releases) == 0 {
		fmt.Println("No releases found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TAG\tNAME\tPRERELEASE\tPUBLISHED")
	for _, r := range releases {
		pre := ""
		if r.Prerelease {
			pre = "yes"
		}
		pub := "-"
		if r.PublishedAt != nil {
			pub = r.PublishedAt.Format("2006-01-02")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.TagName, r.Name, pre, pub)
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d release(s)\n", len(releases))
	return nil
}

// resolveReleaseRepo resolves owner/name from --repo flag or global --repo.
// Delegates to utils.ResolveRepoFromFlag so the global --repo fallback
// and parsing logic are shared across all command groups (label, webhook).
func resolveReleaseRepo(repo string) (string, string, error) {
	return utils.ResolveRepoFromFlag(repo, cmd.GlobalRepo)
}

const releaseListAgentHelp = `# gitee release list — Agent Help

## Purpose
List releases in a Gitee repository. Use this to discover what versions
have been shipped, check the latest prerelease, or audit release history.

## Pre-conditions
- Authenticated (gitee auth status)
- Repository is readable (public OR the token has read access)

## Required flags
  --repo / -R   owner/repo (falls back to global --repo)

## Optional flags
  --limit / -L   Maximum number of releases (default 30, max 100)
  --page         Page number (default 1; use --paginate to traverse all)

## Idempotency
YES. Read-only. No side effects on the repository.

## Atomicity
Atomic. Either the full page of releases is returned, or an error.

## Side effects
None. Pure GET.

## Post-conditions (on success)
- Stdout contains a table of releases (tag, name, prerelease, published date)
- Repository state is unchanged

## Output (stdout, table mode)
  TAG       NAME           PRERELEASE  PUBLISHED
  v1.2.0    1.2.0 Stable               2026-05-20
  v1.2.0-rc1 1.2.0 RC      yes         2026-05-15

## Parsing releases
Use --output json to get the full JSON array, then jq:
  gitee release list -R owner/repo --output json | jq '.[].tag_name'

## Common errors
- 404 Not Found: repo does not exist or is private
- 401 Unauthorized: token missing or expired

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  gitee release list -R myorg/myrepo --limit 10 --output json

## See also
- ` + "`" + `gitee release view` + "`" + `     — read a specific release
- ` + "`" + `gitee release create` + "`" + `   — ship a new version
`
