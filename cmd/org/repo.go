package org

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var orgRepoFlags struct {
	org   string
	limit int
}

var repoCmd = &cobra.Command{
	Use:     "repos <org>",
	Short:   "List organization repositories",
	Example: `  gitee org repos my-org`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(orgRepoAgentHelp)
		}
		return nil
	},
	RunE: runOrgRepos,
}

func init() {
	f := repoCmd.Flags()
	f.StringVar(&orgRepoFlags.org, "org", "", "Organization name")
	f.IntVarP(&orgRepoFlags.limit, "limit", "L", 30, "Maximum number of repos")
}

func runOrgRepos(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	org := orgRepoFlags.org
	if len(args) > 0 {
		org = args[0]
	}
	if org == "" {
		return fmt.Errorf("organization name is required")
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	perPage := orgRepoFlags.limit
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}
	qv := url.Values{}
	qv.Set("per_page", fmt.Sprintf("%d", perPage))
	path := api.OrgRepos(org) + "?" + qv.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var repos []types.Repository
	if err := client.Get(ctx, path, &repos); err != nil {
		return fmt.Errorf("listing org repos: %w", err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(repos, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found")
		return nil
	}

	w := output.NewTableWriter()
	fmt.Fprintln(w, "NAME\tVISIBILITY\tLANGUAGE\tSTARS\tFORKS")
	for _, r := range repos {
		vis := "public"
		if r.Private {
			vis = "private"
		}
		lang := r.Language
		if lang == "" {
			lang = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\n", r.FullName, vis, lang, r.StarsCount, r.ForksCount)
	}
	return w.Flush()
}

// orgRepoAgentHelp is the agent-targeted help for `gitee org repos`.
const orgRepoAgentHelp = `# gitee org repos — Agent Help

## Purpose
Enumerate every repository owned by a Gitee organization, with
visibility, language, stars, and forks. Use this to discover which
repos belong to an org before drilling into a specific one.

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- The org name resolves to an existing organization
- Private repos only show up if the token has access to them

## Required arguments
  <org>   Organization login (e.g. "my-org")

## Optional flags
  --org          Equivalent to passing <org> as a positional arg
  --limit / -L   Maximum number of repos to return (default 30, max 100)
  --output       "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET on /orgs/{org}/repos.

## Post-conditions (on success)
- Stdout contains a table of repos (NAME, VISIBILITY, LANGUAGE, STARS,
  FORKS) or a JSON array of Repository objects
- Repo state is unchanged
- If the org has no repos visible to you, "No repositories found" is
  printed

## Output (stdout, table mode)
  NAME                    VISIBILITY  LANGUAGE  STARS  FORKS
  my-org/awesome          public      Go        128    12
  my-org/internal-api     private     Python    3      0

## Output (stdout, json mode)
  [
    {
      "full_name": "my-org/awesome",
      "private": false,
      "language": "Go",
      "stargazers_count": 128,
      "forks_count": 12,
      ...
    }
  ]

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 403 Forbidden:    no read access to private repos in the org
- 404 Not Found:    org name doesn't exist

## Race condition warning ⚠️
The org's repo set is live server-side state: repos are created,
deleted, renamed, and flipped between public and private by org
members at any time. A ` + "`" + `full_name` + "`" + ` picked from this snapshot can
404 on ` + "`" + `gitee repo view` + "`" + ` or ` + "`" + `gitee repo clone` + "`" + ` moments later
(rename/delete), and a previously visible repo gone private surfaces
as 403. Stars/forks counters are point-in-time. This command
returns one page (--limit is per_page, clamped to 1-100); repos
past that page are not included. Read-only and always safe to retry.

## Next steps (orchestration hints)
After enumerating repos, an agent typically wants to:
1. Pick a ` + "`" + `full_name` + "`" + ` and run ` + "`" + `gitee repo view my-org/awesome` + "`" + `
2. Or ` + "`" + `gitee repo clone my-org/awesome` + "`" + ` to fetch locally
3. Or ` + "`" + `gitee issue list -R my-org/awesome --state open` + "`" + ` for triage

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  # Pull repo full_names as a JSON array, sort by stars
  gitee org repos my-org --output json | jq -r '.[].full_name'

  # Cap at 100 (Gitee per_page max)
  gitee org repos my-org --limit 100

## See also
- ` + "`" + `gitee repo list` + "`" + `     — list repos owned by the current user
- ` + "`" + `gitee repo view` + "`" + `    — read one repo's full metadata
- ` + "`" + `gitee org view` + "`" + `     — org-level metadata
- ` + "`" + `gitee org members` + "`" + `  — list members
`
