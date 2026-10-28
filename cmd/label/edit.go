package label

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var editFlags struct {
	repo       string
	name       string
	newName    string
	labelColor string
}

var editCmd = &cobra.Command{
	Use:   "edit",
	Args:  cobra.NoArgs,
	Short: "Edit an existing label (rename or change color)",
	Example: `  gitee label edit --repo owner/repo --name bug --color 00ff00
  gitee label edit -R owner/repo -n bug --new-name fixed-bug`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(labelEditAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		dc, err := cmd.NewDomainClientFrom(rt)
		if err != nil {
			return err
		}
		return dc.LabelEdit(editFlags.repo, editFlags.name, editFlags.newName, editFlags.labelColor)
	},
}

func init() {
	f := editCmd.Flags()
	f.StringVarP(&editFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&editFlags.name, "name", "n", "", "Current label name (required)")
	f.StringVar(&editFlags.newName, "new-name", "", "New label name")
	f.StringVarP(&editFlags.labelColor, "color", "c", "", "New label color (6 hex digits)")
}

const labelEditAgentHelp = `# gitee label edit — Agent Help

## Purpose
Rename a label or change its color WITHOUT deleting and re-creating it.
This preserves the label's history (which issues/PRs used it).

## Pre-conditions
- Authenticated with write access
- The label --name currently exists in the repo

## Required flags
  --name / -n   Current label name

## Optional flags
  --repo / -R     owner/repo
  --new-name      Rename target (must NOT collide with an existing label)
  --color / -c    New color (6 hex digits, with or without #)

## Idempotency
YES. Re-running with the same values is a server-side no-op and returns
the same object.

## Atomicity
Atomic. Either the rename + color change happen together, or no change.

## Side effects
- Issues/PRs that referenced the label by its old name are auto-updated
- Webhooks subscribed to "Label edited" events may fire

## Post-conditions (on success)
- The label's name and/or color reflect the new values
- All prior issue/PR references continue to resolve

## Output (stdout, table mode)
  ✓ Label updated successfully
    Name:  fixed-bug
    Color: #00ff00
    URL:   https://gitee.com/owner/repo/labels/fixed-bug

## Common errors
- 404 Not Found: --name doesn't exist
- 422 Unprocessable: --new-name already taken
- 403 Forbidden:  token lacks write scope

## Race condition warning ⚠️
The shared mutable state is the label's name + color record.
- Two agents editing the same label concurrently: last writer wins per
  field; there is no optimistic-locking token or ETag.
- A rename invalidates the old name immediately — any agent still
  holding the old name gets 404 on view/edit/delete. After a rename,
  use the --new-name value for ALL follow-up calls.
- A concurrent create of your --new-name target between check and PATCH
  surfaces as 422 (name already taken); pick another name or edit the
  colliding label instead.
- ` + "`" + `gitee label list` + "`" + ` output captured before your edit is stale for
  both name and color — re-list if you must display post-edit state.
Safe to retry with the same values (server-side no-op). A retry after a
successful rename must pass the NEW name as --name.

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour.

## Example
  # Just change color
  gitee label edit -R owner/repo -n bug -c ffff00

  # Just rename
  gitee label edit -R owner/repo -n bug --new-name confirmed-bug

  # Both
  gitee label edit -R owner/repo -n bug --new-name confirmed-bug -c 00ff00

## See also
- ` + "`" + `gitee label view` + "`" + ` — confirm current state
- ` + "`" + `gitee label delete` + "`" + ` — remove (and lose history)
- ` + "`" + `gitee label create` + "`" + ` — add a new one
`
