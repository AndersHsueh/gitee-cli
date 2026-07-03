package milestone

import (
	"gitee-cli/cmd"
	"gitee-cli/internal/handler"
	"gitee-cli/internal/utils"

	"github.com/spf13/cobra"
)

var deleteFlags struct {
	repo   string
	number int64
	force  bool
}

var deleteCmd = &cobra.Command{
	Use:   "delete <number>",
	Short: "Delete a milestone",
	Example: `  gitee milestone delete --repo owner/repo --number 1
  gitee milestone delete -R owner/repo 3 --force`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(milestoneDeleteAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		parsed, err := utils.ParseInt64Arg(deleteFlags.number, args, "milestone")
		if err != nil {
			return err
		}
		confirm := ""
		if deleteFlags.force {
			confirm = "y"
		}
		return handler.MilestoneDelete(deleteFlags.repo, int(parsed), confirm)
	},
}

func init() {
	f := deleteCmd.Flags()
	f.StringVarP(&deleteFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.Int64VarP(&deleteFlags.number, "number", "n", 0, "Milestone number to delete (required, or pass as positional)")
	f.BoolVarP(&deleteFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

const milestoneDeleteAgentHelp = `# gitee milestone delete — Agent Help

## Purpose
Permanently remove a milestone from a repository. Issues and PRs that
referenced the milestone are detached (their ` + "`" + `milestone` + "`" + ` field is cleared);
the issues themselves are NOT deleted.

## Pre-conditions
- Authenticated with admin access
- The milestone --number exists in the repo

## Required flags
  --number / -n   Milestone number (or pass as positional)

## Optional flags
  --repo / -R    owner/repo
  --force / -f   Skip the y/N prompt (use in scripts / agents)

## Idempotency
NO. Re-running on a deleted milestone returns 404. There is no
"create or update" path on delete; use ` + "`" + `gitee milestone create` + "`" + ` to
add a fresh milestone (with a new number).

## Atomicity
Atomic.

## Side effects
- The milestone is removed from the repo
- Issues/PRs that had this milestone lose their ` + "`" + `milestone` + "`" + ` field
  (they are NOT deleted)
- Webhooks subscribed to milestone-deleted events may fire
- This action is NOT recoverable from the API (audit logs only)

## Post-conditions (on success)
- The milestone no longer exists in the repo
- All issue/PR references are detached

## Output (stdout)
  ✓ Milestone #3 deleted

## Common errors
- 404 Not Found: --number doesn't exist
- 403 Forbidden:  token lacks admin scope

## OAuth scope required
projects (admin)

## Rate limit
1000 req/hour.

## Example
  gitee milestone delete -R owner/repo -n 3 --force

## See also
- ` + "`" + `gitee milestone list` + "`" + `   — see what exists before delete
- ` + "`" + `gitee milestone view` + "`" + `   — confirm details
`