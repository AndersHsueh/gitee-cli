package org

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
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
	opts := cmd.NewCmdkitOptions(client)
	return cmdkit.RunList[types.Organization](opts, cmdkit.ListSpec[types.Organization]{
		// org list lists the authenticated user's orgs — no target repo.
		ResolveOwnerRepo: func() (string, string, error) {
			return "", "", nil
		},
		Path: func(_, _ string) string {
			// Clamp per_page to Gitee v5's 1-100 range before sending it.
			perPage := listFlags.limit
			if perPage < 1 {
				perPage = 30
			}
			if perPage > 100 {
				perPage = 100
			}
			qv := url.Values{}
			qv.Set("per_page", fmt.Sprintf("%d", perPage))
			return api.OrgsList() + "?" + qv.Encode()
		},
		// org list caps per_page with --limit and does a single GET — no
		// --paginate support.
		Fetch: func(ctx context.Context, client *api.Client, path string) ([]types.Organization, error) {
			var orgs []types.Organization
			if err := client.Get(ctx, path, &orgs); err != nil {
				return nil, fmt.Errorf("listing orgs: %w", err)
			}
			return orgs, nil
		},
		Columns: []string{"LOGIN", "NAME", "DESCRIPTION", "PUBLIC REPOS"},
		Row: func(o types.Organization) []string {
			desc := utils.Truncate(o.Description, 50)
			if desc == "" {
				desc = "-"
			}
			return []string{o.Login, o.Name, desc, strconv.Itoa(o.PublicRepos)}
		},
		EmptyText: func(_, _ string) string {
			return "No organizations found"
		},
	})
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

## Race condition warning ⚠️
Your org membership is live server-side state: an org admin can add
or remove you, and orgs can be renamed, at any time. This output is a
snapshot — a ` + "`" + `login` + "`" + ` picked from it can 404 on
` + "`" + `gitee org view <login>` + "`" + ` seconds later if the org was renamed or
your membership revoked. If a follow-up action depends on membership,
re-run this list (or ` + "`" + `gitee org view` + "`" + `) immediately before acting
rather than trusting an older read. Read-only and always safe to
retry.

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
