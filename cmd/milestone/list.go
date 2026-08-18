package milestone

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var listFlags struct {
	repo  string
	limit int
	state string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List milestones in a repository",
	Example: `  gitee milestone list --repo owner/repo
  gitee milestone list -R owner/repo --state closed --limit 50`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(milestoneListAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		dc, err := cmd.NewDomainClient()
		if err != nil {
			return err
		}
		return dc.MilestoneList(listFlags.repo, listFlags.state, listFlags.limit)
	},
}

func init() {
	f := listCmd.Flags()
	f.StringVarP(&listFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of milestones to fetch")
	f.StringVar(&listFlags.state, "state", "", "Filter by state: open|closed (default: all)")
}

const milestoneListAgentHelp = `# gitee milestone list — Agent Help

## Purpose
List all milestones defined in a repository. Use this to discover what
delivery targets exist, check open issue counts per milestone, or audit
which milestones are overdue (compare ` + "`" + `due_on` + "`" + ` with today).

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- Repository is readable

## Required flags
  --repo / -R   owner/repo (falls back to global --repo)

## Optional flags
  --limit / -L   Maximum number of milestones (default 30, max 100 per page)
  --state        Filter by exact state: open or closed (default: all)
  --output       "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None.

## Post-conditions (on success)
- Stdout contains a table of milestones (NUMBER, TITLE, STATE, OPEN, CLOSED, DUE)
- Repo state is unchanged

## Output (stdout, table mode)
  NUMBER  TITLE                              STATE  OPEN  CLOSED  DUE
  1       v1.0                               open   12    3       2026-09-30
  2       v1.1                               open   4     0       2026-12-15
  3       v0.9 (legacy)                      closed 0     25

## Output (stdout, json mode)
  [
    {"id":1,"number":1,"title":"v1.0","state":"open","open_issues":12,"closed_issues":3,"due_on":"2026-09-30T00:00:00+08:00"},
    ...
  ]

## Common errors
- 404 Not Found: repo does not exist or is private
- 401 Unauthorized: token missing or expired

## Race condition warning ⚠️
The result is a point-in-time snapshot of the repo's milestone set.
The OPEN/CLOSED issue counts move every time an issue is opened,
closed, or reassigned, so counts parsed from this output can be stale
within seconds — do not feed them into automation without a fresh
` + "`" + `gitee milestone view` + "`" + `.
- create→list gap: a milestone just created (by you or another agent)
  may be missing from this list for 1-2s due to indexing lag. Do NOT
  use this command to verify a create you just ran; use the number
  from the create output instead.
- delete→list gap: a just-deleted milestone may briefly still appear.

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour authenticated.

## Example
  gitee milestone list -R myorg/myrepo --state open --output json

## See also
- ` + "`" + `gitee milestone view` + "`" + `   — full details for a single milestone
- ` + "`" + `gitee milestone create` + "`" + ` — add a new milestone
- ` + "`" + `gitee issue list` + "`" + `     — filter issues by milestone number
`