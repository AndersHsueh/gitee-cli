package release

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var viewFlags struct {
	repo string
	tag  string
}

var viewCmd = &cobra.Command{
	Use:   "view <tag>",
	Short: "View details of a release",
	Example: `  gitee release view v1.0.0 --repo owner/repo
  gitee release view v1.0.0 -R owner/repo --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(releaseViewAgentHelp)
		}
		return nil
	},
	RunE: runView,
}

func init() {
	f := viewCmd.Flags()
	f.StringVarP(&viewFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&viewFlags.tag, "tag", "t", "", "Release tag name (e.g. v1.0.0)")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	tag := viewFlags.tag
	if tag == "" && len(args) >= 1 {
		tag = args[0]
	}
	if tag == "" {
		return fmt.Errorf("tag is required (use --tag or pass as first arg)")
	}

	owner, repoName, err := resolveReleaseRepo(viewFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.ReleasesGet(owner, repoName, tag)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var release types.Release
	if err := client.Get(ctx, path, &release); err != nil {
		return fmt.Errorf("getting release %s: %w", tag, err)
	}

	if rt.Output == "json" {
		return output.PrintJSON(release)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Release %s\n", bold("─"), bold(release.TagName))
	if release.Name != "" {
		fmt.Printf("  Name:        %s\n", release.Name)
	}
	if release.TargetCommitish != "" {
		fmt.Printf("  Target:      %s\n", release.TargetCommitish)
	}
	pre := "no"
	if release.Prerelease {
		pre = "yes"
	}
	fmt.Printf("  Prerelease:  %s\n", pre)
	if release.PublishedAt != nil {
		fmt.Printf("  Published:   %s\n", release.PublishedAt.Format(time.RFC3339))
	}
	if release.Author.Login != "" {
		fmt.Printf("  Author:      %s\n", release.Author.Login)
	}
	if release.Body != "" {
		fmt.Println()
		fmt.Println(bold("Body:"))
		fmt.Println(release.Body)
	}
	if len(release.Assets) > 0 {
		fmt.Println()
		fmt.Println(bold("Assets:"))
		for _, a := range release.Assets {
			fmt.Printf("  %s  %s  %s\n", green("•"), a.Name, cyan(a.DownloadURL))
		}
	}
	fmt.Println()
	if release.HTMLURL != "" {
		fmt.Printf("URL: %s\n", color.YellowString(release.HTMLURL))
	}
	return nil
}

const releaseViewAgentHelp = `# gitee release view — Agent Help

## Purpose
Read the full details of a single release by tag name. Returns title, body
(Markdown rendered), target commit, prerelease flag, author, asset list,
and the HTML URL.

## Pre-conditions
- Authenticated
- Tag exists in the repo (returns 404 otherwise)

## Required flags
  --tag / -t  (or first positional arg)   Tag name, e.g. v1.0.0
  --repo / -R                              owner/repo

## Optional flags
  --output / -o   table|json (default table)

## Idempotency
YES. Pure GET.

## Atomicity
Atomic.

## Side effects
None.

## Post-conditions (on success)
- Stdout contains release details in table or json format

## Output (stdout, table mode)
  ─ Release v1.0.0
    Name:        1.0.0 Stable
    Target:      main
    Prerelease:  no
    Published:   2026-05-20T12:00:00Z
    Author:      octocat
    URL:         https://gitee.com/owner/repo/releases/tag/v1.0.0

## Common errors
- 404 Not Found: tag has no release attached (the git tag exists but no release metadata)

## Race condition warning ⚠️
The metadata and asset list returned here are a snapshot. An asset can
be uploaded or deleted between this view and a follow-up download or
delete-asset call — the follow-up then 404s or fetches different bytes
than you saw. If the gap is more than a few seconds and your action
depends on what you saw, re-run this command immediately before acting.
Right after ` + "`" + `gitee release create` + "`" + `, this command may briefly
return 404 while Gitee indexing catches up (1-2 seconds); retry instead
of concluding the release is missing.

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour authenticated.

## Example
  gitee release view v1.0.0 -R myorg/myrepo --output json | jq .body

## See also
- ` + "`" + `gitee release list` + "`" + `     — enumerate releases
- ` + "`" + `gitee release download` + "`" + ` — fetch assets
`
