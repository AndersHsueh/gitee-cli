package milestone

import (
	"gitee-cli/cmd"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var createFlags struct {
	repo        string
	title       string
	description string
	dueOn       string
	state       string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new milestone",
	Example: `  gitee milestone create --repo owner/repo --title v1.0
  gitee milestone create -R owner/repo -t v1.1 --due-on 2026-12-15 --description "Q4 release"`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(milestoneCreateAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		req := types.MilestoneCreateRequest{
			Title:       createFlags.title,
			Description: createFlags.description,
			DueOn:       createFlags.dueOn,
			State:       createFlags.state,
		}
		dc, err := cmd.NewDomainClientFrom(rt)
		if err != nil {
			return err
		}
		return dc.MilestoneCreate(createFlags.repo, req)
	},
}

func init() {
	f := createCmd.Flags()
	f.StringVarP(&createFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&createFlags.title, "title", "t", "", "Milestone title (required)")
	f.StringVarP(&createFlags.description, "description", "d", "", "Milestone description")
	f.StringVar(&createFlags.dueOn, "due-on", "", "Due date in YYYY-MM-DD format (optional)")
	f.StringVar(&createFlags.state, "state", "open", "Initial state: open or closed (default: open)")
}

const milestoneCreateAgentHelp = `# gitee milestone create — Agent Help

## Purpose
Add a new milestone to a repository. Milestones are scoped to a single
repo and must be created before they can be assigned to issues or PRs.

## Pre-conditions
- Authenticated with write access to the repo

## Required flags
  --title / -t   Milestone title (required, non-empty)

## Optional flags
  --repo / -R         owner/repo
  --description / -d  Free-form description (markdown supported)
  --due-on            Due date as ` + "`" + `YYYY-MM-DD` + "`" + ` (Gitee parses this as midnight local time)
  --state             open or closed (default: open)

## Idempotency
NOT idempotent. Re-running with the same title creates a NEW milestone
(milestones are not deduplicated server-side by title; the server
assigns a fresh ` + "`" + `number` + "`" + `). Use ` + "`" + `gitee milestone edit` + "`" + ` to
rename an existing milestone instead.

## Atomicity
Atomic. Either the milestone is created with all supplied fields, or no
change happens.

## Side effects
- The new milestone is immediately available in the issue/PR UI's
  milestone picker
- Webhooks subscribed to milestone events may fire

## Post-conditions (on success)
- A new milestone with the given title (and optional description / due
  date / state) exists in the repo
- Stdout shows success + URL

## Output (stdout, table mode)
  ✓ Milestone created successfully
    Number: 7
    Title:  v1.0
    URL:    https://gitee.com/owner/repo/milestones/7

## Common errors
- 422 Unprocessable: title is empty or due_on is malformed
- 401 Unauthorized: token expired
- 403 Forbidden:    token lacks write scope

## Race condition warning ⚠️
The shared mutable state is the repo's milestone set. Gitee does NOT
deduplicate by title: two agents (or a retried call) running this
command with the same --title produce TWO milestones with different
numbers, both appearing in the issue/PR milestone picker.
- create→list gap: indexing can lag 1-2s, so an immediate ` + "`" + `gitee milestone list` + "`" + ` may NOT show the new milestone. Use the Number from THIS
  command's output — do not re-list to find it.
- On timeout / network error, DO NOT blindly retry: the first attempt
  may have succeeded server-side. Run ` + "`" + `gitee milestone list` + "`" + ` first and
  reuse the existing number if the title is already there.

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour.

## Example
  gitee milestone create -R myorg/myrepo -t "v2.0" -d "Major rewrite" --due-on 2027-01-31

## See also
- ` + "`" + `gitee milestone list` + "`" + `   — see what already exists
- ` + "`" + `gitee milestone edit` + "`" + `   — change title / due date / state later
`
