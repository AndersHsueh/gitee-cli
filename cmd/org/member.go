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

var memberFlags struct {
	org   string
	limit int
}

var memberCmd = &cobra.Command{
	Use:     "members <org>",
	Short:   "List organization members",
	Example: `  gitee org members my-org`,
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(orgMemberAgentHelp)
		}
		return nil
	},
	RunE: runMembers,
}

func init() {
	f := memberCmd.Flags()
	f.StringVar(&memberFlags.org, "org", "", "Organization name")
	f.IntVarP(&memberFlags.limit, "limit", "L", 30, "Maximum number of members")
}

func runMembers(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	org := memberFlags.org
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

	perPage := memberFlags.limit
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}
	qv := url.Values{}
	qv.Set("per_page", fmt.Sprintf("%d", perPage))
	path := api.OrgMembers(org) + "?" + qv.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var members []types.OrgMember
	if err := client.Get(ctx, path, &members); err != nil {
		return fmt.Errorf("listing org members: %w", err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(members, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(members) == 0 {
		fmt.Println("No members found")
		return nil
	}

	w := output.NewTableWriter()
	fmt.Fprintln(w, "LOGIN\tNAME\tEMAIL\tSTATE")
	for _, m := range members {
		email := m.Email
		if email == "" {
			email = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.Login, m.Name, email, m.State)
	}
	return w.Flush()
}

// orgMemberAgentHelp is the agent-targeted help for `gitee org members`.
const orgMemberAgentHelp = `# gitee org members — Agent Help

## Purpose
List members of a Gitee organization with their display name, email,
and active state. Use this to identify owners/maintainers for routing
notifications, audits, or access checks.

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- The org name resolves to an existing organization
- Token can read org membership (any token the org permits works)

## Required arguments
  <org>   Organization login (e.g. "my-org")

## Optional flags
  --org          Equivalent to passing <org> as a positional arg
  --limit / -L   Maximum number of members to return (default 30, max 100)
  --output       "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET on /orgs/{org}/members.

## Post-conditions (on success)
- Stdout contains a table of members (LOGIN, NAME, EMAIL, STATE) or
  a JSON array of OrgMember objects
- Repo state is unchanged
- If the org has no members visible to you, "No members found" is printed

## Output (stdout, table mode)
  LOGIN     NAME         EMAIL                 STATE
  alice     Alice Liu    alice@example.com     active
  bob       Bob Chen     -                     active

## Output (stdout, json mode)
  [
    {
      "login": "alice",
      "name": "Alice Liu",
      "email": "alice@example.com",
      "state": "active"
    }
  ]

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 404 Not Found:    org name doesn't exist

## Race condition warning ⚠️
The org's member roster is live server-side state: admins invite and
remove members continuously. This output is a snapshot — a ` + "`" + `login` + "`" + `
picked for issue assignment or notification may no longer be a member
by the time you act on it. With ` + "`" + `--paginate` + "`" + `, mutations
mid-traversal can make pages overlap or skip entries — dedupe by
` + "`" + `login` + "`" + `. Also note the ` + "`" + `state` + "`" + ` field: only ` + "`" + `active` + "`" + `
members can actually receive assignments. Read-only and always safe
to retry.

## Next steps (orchestration hints)
After listing members, an agent typically wants to:
1. Pick a ` + "`" + `login` + "`" + ` and run ` + "`" + `gitee user view <login>` + "`" + `
2. Filter the list down to a single owner for assigning issues
3. Cross-reference with ` + "`" + `gitee org repos` + "`" + ` to map members to repos

## OAuth scope required
projects (read) — org membership is a protected resource

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  # Pull all member logins as a JSON array
  gitee org members my-org --output json | jq -r '.[].login'

  # Cap at 100 results (Gitee per_page max)
  gitee org members my-org --limit 100

## See also
- ` + "`" + `gitee org view` + "`" + `    — org-level metadata
- ` + "`" + `gitee user view` + "`" + `  — read one user's profile
- ` + "`" + `gitee org repos` + "`" + `  — list repos in the org
`
