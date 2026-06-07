package label

import (
	"gitee-cli/cmd"
	"gitee-cli/internal/handler"

	"github.com/spf13/cobra"
)

var deleteFlags struct {
	repo  string
	name  string
	force bool
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a label",
	Example: `  gitee label delete --repo owner/repo --name bug
  gitee label delete -R owner/repo -n wontfix --force`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(labelDeleteAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		confirm := ""
		if deleteFlags.force {
			confirm = "y"
		}
		return handler.LabelDelete(deleteFlags.repo, deleteFlags.name, confirm)
	},
}

func init() {
	f := deleteCmd.Flags()
	f.StringVarP(&deleteFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&deleteFlags.name, "name", "n", "", "Label name to delete (required)")
	f.BoolVarP(&deleteFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

const labelDeleteAgentHelp = `# gitee label delete — Agent Help

## Purpose
Permanently remove a label from a repository. The label is detached
from any issues/PRs that referenced it; the issues themselves are
NOT deleted.

## Pre-conditions
- Authenticated with admin access
- The label --name exists in the repo

## Required flags
  --name / -n   Label name to delete

## Optional flags
  --repo / -R    owner/repo
  --force / -f   Skip the y/N prompt (use in scripts / agents)

## Idempotency
NO. Re-running on a deleted label returns 404. There is no "create
or update" path on delete; use ` + "`" + `gitee label create` + "`" + ` to add it back.

## Atomicity
Atomic.

## Side effects
- The label is removed from the repo
- Issues/PRs that had this label are unlabeled (NOT deleted)
- Webhooks subscribed to "Label deleted" events may fire
- This action is NOT recoverable from the API (audit logs only)

## Post-conditions (on success)
- The label no longer exists in the repo
- All references in issues/PRs are removed

## Output (stdout)
  ✓ Label "bug" deleted successfully

## Common errors
- 404 Not Found: --name doesn't exist
- 403 Forbidden:  token lacks admin scope

## OAuth scope required
projects (admin)

## Rate limit
1000 req/hour.

## Example
  gitee label delete -R owner/repo -n wontfix --force

## See also
- ` + "`" + `gitee label list` + "`" + `   — see what exists before delete
- ` + "`" + `gitee label view` + "`" + `   — confirm details
`
