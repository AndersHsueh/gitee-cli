package org

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
	limit int
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List your organizations",
	Example: `  gitee org list`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(orgListAgentHelp)
		}
		return nil
	},
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of orgs")
}

func runList(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	perPage := listFlags.limit
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}
	qv := url.Values{}
	qv.Set("per_page", fmt.Sprintf("%d", perPage))
	path := api.OrgsList() + "?" + qv.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var orgs []types.Organization
	if err := client.Get(ctx, path, &orgs); err != nil {
		return fmt.Errorf("listing orgs: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(orgs, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(orgs) == 0 {
		fmt.Println("No organizations found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "LOGIN\tNAME\tDESCRIPTION\tPUBLIC REPOS")
	for _, o := range orgs {
		desc := utils.Truncate(o.Description, 50)
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", o.Login, o.Name, desc, o.PublicRepos)
	}
	return w.Flush()
}

// orgListAgentHelp is the agent-targeted help for `gitee org list`.
const orgListAgentHelp = `# gitee org list — Agent Help

## Purpose
Enumerate every organization the authenticated user belongs to on Gitee.
Use this to discover which orgs are available before issuing an
` + "`" + `org view` + "`" + `, ` + "`" + `org repos` + "`" + `, or ` + "`" + `org issues` + "`" + ` call.

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- Token is valid (read scope is sufficient — this is /user/orgs)

## Optional flags
  --limit / -L   Maximum number of orgs to return (default 30, max 100)
  --output       "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET on /user/orgs.

## Post-conditions (on success)
- Stdout contains a table of orgs (LOGIN, NAME, DESCRIPTION, PUBLIC REPOS)
  or a JSON array of Organization objects
- Repo state is unchanged
- If the user is in no orgs, "No organizations found" is printed

## Output (stdout, table mode)
  LOGIN        NAME              DESCRIPTION                     PUBLIC REPOS
  my-org       My Organization   An example org                  12
  other-org    Other Org         -                               3

## Output (stdout, json mode)
  [
    {
      "login": "my-org",
      "name": "My Organization",
      "description": "An example org",
      "public_repos": 12,
      ...
    }
  ]

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 403 Forbidden:    token lacks user scope

## Next steps (orchestration hints)
After enumerating orgs, an agent typically wants to:
1. Pick the ` + "`" + `login` + "`" + ` of the target org
2. Run ` + "`" + `gitee org view <login>` + "`" + ` to read metadata
3. Or ` + "`" + `gitee org repos <login>` + "`" + ` to discover repos
4. Or ` + "`" + `gitee org issues <login> --state open` + "`" + ` to triage issues

## OAuth scope required
user_info (read) — /user/orgs requires the authenticated user

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  # Discover orgs as JSON and pipe to jq
  gitee org list --output json | jq -r '.[].login'

  # Limit to 5 results in table mode
  gitee org list --limit 5

## See also
- ` + "`" + `gitee org view` + "`" + `   — read one org's metadata
- ` + "`" + `gitee org repos` + "`" + `  — list repos in an org
- ` + "`" + `gitee org issues` + "`" + ` — list issues across an org
`
