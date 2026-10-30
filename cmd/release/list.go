package release

import (
	"context"
	"fmt"
	"net/url"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
	"gitee-cli/internal/output"
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
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
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
	rt := cmd.RuntimeFromCmd(cobraCmd)
	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}
	opts := cmd.NewCmdkitOptionsFrom(rt, client)
	return cmdkit.RunList[types.Release](opts, cmdkit.ListSpec[types.Release]{
		ResolveOwnerRepo: func() (string, string, error) {
			return resolveReleaseRepo(listFlags.repo)
		},
		Path: func(owner, repoName string) string {
			path := api.ReleasesList(owner, repoName)
			q := url.Values{}
			q.Set("per_page", fmt.Sprintf("%d", listFlags.limit))
			q.Set("page", fmt.Sprintf("%d", listFlags.page))
			return path + "?" + q.Encode()
		},
		// release list caps per_page with --limit and does a single GET
		// (page via --page) — no --paginate support.
		Fetch: func(ctx context.Context, client *api.Client, path string) ([]types.Release, error) {
			var releases []types.Release
			if err := client.Get(ctx, path, &releases); err != nil {
				return nil, fmt.Errorf("listing releases: %w", err)
			}
			return releases, nil
		},
		Columns: []string{"TAG", "NAME", "PRERELEASE", "PUBLISHED"},
		Row: func(r types.Release) []string {
			pre := ""
			if r.Prerelease {
				pre = "yes"
			}
			pub := "-"
			if r.PublishedAt != nil {
				pub = r.PublishedAt.Format("2006-01-02")
			}
			return []string{r.TagName, r.Name, pre, pub}
		},
		EmptyText: func(_, _ string) string {
			return "No releases found"
		},
		Footer: func(count int) string {
			return fmt.Sprintf("\nTotal: %d release(s)\n", count)
		},
		JSON: func(items []types.Release) error {
			return output.PrintJSON(items)
		},
	})
}

// resolveReleaseRepo resolves owner/name from --repo flag or global --repo.
// Delegates to utils.ResolveRepo so parsing and the git-config fallback are
// shared across all command groups (label, webhook). The local flag wins;
// the global --repo is used only when the local flag is empty.
func resolveReleaseRepo(repo string) (string, string, error) {
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	return utils.ResolveRepo("", repo, nil)
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
  --limit / -L   Page size sent as per_page on this one request (default 30).
                 The CLI does not clamp it; Gitee rejects values above 100.
  --page         Page number (default 1). This verb returns that single page.
                 It does not walk further pages.

## Idempotency
YES. Read-only. No side effects on the repository.

## Atomicity
One GET. Either that single page of releases is returned, or an error.

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

## Race condition warning ⚠️
The output is a snapshot of the repo's release set at fetch time. A
release created or deleted after this call is not reflected, so do not
act on stale output: re-check with ` + "`" + `gitee release view <tag>` + "`" + `
immediately before any follow-up action. Paginating with --page while
releases are being added can shift page boundaries, producing duplicate
or skipped entries — prefer a fresh single-page read in scripts. If you
just created a release, use the tag from the create output instead of
re-listing; Gitee indexing can lag 1-2 seconds.

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
