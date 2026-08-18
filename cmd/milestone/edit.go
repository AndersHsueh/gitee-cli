package milestone

import (
	"gitee-cli/cmd"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var editFlags struct {
	repo        string
	number      int64
	title       string
	description string
	dueOn       string
	state       string
}

var editCmd = &cobra.Command{
	Use:   "edit <number>",
	Short: "Edit an existing milestone",
	Example: `  gitee milestone edit --repo owner/repo --number 1 --title v1.0-rc
  gitee milestone edit -R owner/repo 2 --state closed
  gitee milestone edit -R owner/repo 3 --due-on 2027-02-28 --description "Updated scope"`,
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(milestoneEditAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		parsed, err := utils.ParseInt64Arg(editFlags.number, args, "milestone")
		if err != nil {
			return err
		}
		req := types.MilestoneUpdateRequest{
			Title:       editFlags.title,
			Description: editFlags.description,
			DueOn:       editFlags.dueOn,
			State:       editFlags.state,
		}
		dc, err := cmd.NewDomainClientFrom(rt)
		if err != nil {
			return err
		}
		return dc.MilestoneEdit(editFlags.repo, int(parsed), req)
	},
}

func init() {
	f := editCmd.Flags()
	f.StringVarP(&editFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.Int64VarP(&editFlags.number, "number", "n", 0, "Milestone number (required, or pass as positional)")
	f.StringVarP(&editFlags.title, "title", "t", "", "New milestone title")
	f.StringVarP(&editFlags.description, "description", "d", "", "New milestone description")
	f.StringVar(&editFlags.dueOn, "due-on", "", "New due date (YYYY-MM-DD). Pass empty string to clear.")
	f.StringVar(&editFlags.state, "state", "", "New state: open or closed (required to change)")
}

const milestoneEditAgentHelp = `# gitee milestone edit — Agent Help

## Purpose
Update a milestone's title, description, due date, or state WITHOUT
deleting and re-creating it. This preserves the milestone's number
(which issues/PRs reference) and its assignment history.

## Pre-conditions
- Authenticated with write access
- The milestone --number currently exists in the repo

## Required flags
  --number / -n   Milestone number (or pass as positional)

## Optional flags
  --repo / -R         owner/repo
  --title / -t        New title
  --description / -d  New description (pass empty string to clear)
  --due-on            New due date ` + "`" + `YYYY-MM-DD` + "`" + ` (pass empty string to clear)
  --state             New state: open or closed

## Idempotency
YES. Re-running with the same values is a server-side no-op.

## Atomicity
Atomic. Either the supplied fields are updated together, or no change.

## Side effects
- Issues/PRs that referenced this milestone keep their reference (the
  number does not change)
- Webhooks subscribed to milestone-edited events may fire

## Post-conditions (on success)
- The milestone reflects the new field values
- All prior issue/PR references continue to resolve

## Output (stdout, table mode)
  ✓ Milestone updated (number=2)
    Title:  v1.1 (revised)
    State:  open

## Common errors
- 404 Not Found: --number doesn't exist
- 422 Unprocessable: invalid state value or malformed due_on
- 403 Forbidden:  token lacks write scope

## Race condition warning ⚠️
The shared mutable state is the milestone's title / description /
due_on / state fields. Gitee applies updates last-write-wins with no
optimistic locking: two concurrent ` + "`" + `gitee milestone edit` + "`" + ` calls on the
same number silently overwrite each other — the later request wins.
- view→edit gap: the state you based the edit on (e.g. "still open")
  may have changed. Re-check with ` + "`" + `gitee milestone view -n N` + "`" + `
  immediately before a state flip.
- edit→delete interleave: if another agent deletes the milestone
  first, this command 404s; if it re-creates one with the same title,
  your edit lands on a DIFFERENT milestone.
- Retrying this command with the same flags is safe (see Idempotency).

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour.

## Example
  # Rename
  gitee milestone edit -R owner/repo -n 2 -t "v1.1 (revised)"

  # Close out
  gitee milestone edit -R owner/repo 3 --state closed

  # Push out the date
  gitee milestone edit -R owner/repo -n 1 --due-on 2027-03-01

## See also
- ` + "`" + `gitee milestone view` + "`" + `   — confirm current state
- ` + "`" + `gitee milestone delete` + "`" + ` — remove (and lose history)
- ` + "`" + `gitee milestone create` + "`" + ` — add a new milestone
`