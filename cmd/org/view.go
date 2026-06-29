package org

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var viewFlags struct {
	org string
}

var viewCmd = &cobra.Command{
	Use:     "view [<org>]",
	Short:   "View organization details",
	Example: `  gitee org view my-org`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(orgViewAgentHelp)
		}
		return nil
	},
	RunE: runView,
}

func init() {
	f := viewCmd.Flags()
	f.StringVar(&viewFlags.org, "org", "", "Organization name")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	org := viewFlags.org
	if len(args) > 0 {
		org = args[0]
	}
	if org == "" {
		return fmt.Errorf("organization name is required")
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.OrgGet(org)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var o types.Organization
	if err := client.Get(ctx, path, &o); err != nil {
		return fmt.Errorf("viewing org: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(o, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Login:\t%s\n", o.Login)
	fmt.Fprintf(w, "Name:\t%s\n", o.Name)
	fmt.Fprintf(w, "Description:\t%s\n", o.Description)
	fmt.Fprintf(w, "Email:\t%s\n", o.Email)
	fmt.Fprintf(w, "Blog:\t%s\n", o.Blog)
	fmt.Fprintf(w, "Public Repos:\t%d\n", o.PublicRepos)
	fmt.Fprintf(w, "Followers:\t%d\n", o.Followers)
	fmt.Fprintf(w, "Following:\t%d\n", o.Following)
	fmt.Fprintf(w, "URL:\t%s\n", o.HTMLURL)
	return w.Flush()
}

// orgViewAgentHelp is the agent-targeted help for `gitee org view`.
const orgViewAgentHelp = `# gitee org view — Agent Help

## Purpose
Fetch a single organization's profile and metadata from Gitee
(public_repos, followers, blog, etc.). Use this to verify an org exists,
grab its description, or pull its URL for chaining.

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- The org name resolves to an existing organization

## Required arguments
  <org>   Organization login (the slug, e.g. "my-org")

## Optional flags
  --org          Equivalent to passing <org> as a positional arg
  --output       "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET on /orgs/{org}.

## Post-conditions (on success)
- Stdout contains a key/value block (Login, Name, Description, Email,
  Blog, Public Repos, Followers, Following, URL) or a JSON Organization
- Repo state is unchanged

## Output (stdout, table mode)
  Login:        my-org
  Name:         My Organization
  Description:  An example org
  Email:        admin@example.com
  Blog:         https://example.com
  Public Repos: 12
  Followers:    5
  Following:    1
  URL:          https://gitee.com/my-org

## Output (stdout, json mode)
  {
    "login": "my-org",
    "name": "My Organization",
    "description": "An example org",
    "public_repos": 12,
    ...
  }

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 404 Not Found:    org name doesn't exist or has been renamed

## Next steps (orchestration hints)
After reading org metadata, an agent typically wants to:
1. Run ` + "`" + `gitee org repos <login>` + "`" + ` to enumerate its repos
2. Run ` + "`" + `gitee org members <login>` + "`" + ` to see who can act on it
3. Run ` + "`" + `gitee org issues <login> --state open` + "`" + ` to triage

## OAuth scope required
None (public endpoint, but auth raises the rate limit)

## Rate limit
Gitee API v5: 1000 req/hour authenticated; 60 req/hour anonymous.

## Example
  # Read the org as JSON, extract the website
  gitee org view my-org --output json | jq -r '.blog'

  # Equivalent using --org flag (no positional)
  gitee org view --org my-org

## See also
- ` + "`" + `gitee org list` + "`" + `    — enumerate the user's orgs
- ` + "`" + `gitee org repos` + "`" + `  — list repos in an org
- ` + "`" + `gitee org members` + "`" + ` — list members
`
