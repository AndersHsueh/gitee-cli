package label

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var viewFlags struct {
	repo string
	name string
}

var viewCmd = &cobra.Command{
	Use:   "view",
	Args:  cobra.NoArgs,
	Short: "View a single label",
	Example: `  gitee label view --repo owner/repo --name bug
  gitee label view -R owner/repo -n enhancement`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(labelViewAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		dc, err := cmd.NewDomainClientFrom(rt)
		if err != nil {
			return err
		}
		return dc.LabelView(viewFlags.repo, viewFlags.name)
	},
}

func init() {
	f := viewCmd.Flags()
	f.StringVarP(&viewFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&viewFlags.name, "name", "n", "", "Label name (required)")
}

const labelViewAgentHelp = `# gitee label view — Agent Help

## Purpose
Read a single label's full metadata: name, color (with visual swatch),
description, and URL.

## Pre-conditions
- Authenticated
- Label exists in the repo (returns 404 otherwise)

## Required flags
  --name / -n   Label name
  --repo / -R   owner/repo

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None.

## Post-conditions (on success)
- Stdout shows the label's full details

## Output (stdout, table mode)
  ─ Label Details
    Name:  bug
    Color: #ff0000 (visual swatch)
    URL:   https://gitee.com/owner/repo/labels/bug

## Common errors
- 404 Not Found: label does not exist in this repo
- 401 Unauthorized: token missing or expired

## Race condition warning ⚠️
The shared mutable state is the single label record; this command reads
a point-in-time snapshot.
- The label can be renamed, recolored, or deleted between this read and
  any action you take based on it. If your action depends on current
  state (e.g. "only delete if the color is red"), re-view immediately
  before the write.
- After a rename by another agent, view by the old name returns 404.
  That means "renamed or deleted", not "never existed" — cross-check
  with ` + "`" + `gitee label list` + "`" + ` before concluding anything.
- For 1-2 seconds after ` + "`" + `gitee label create` + "`" + `, viewing the new name
  may briefly 404 (indexing lag); retry once rather than assuming the
  create failed.

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour.

## Example
  gitee label view -R owner/repo -n bug

## See also
- ` + "`" + `gitee label list` + "`" + `   — enumerate labels
- ` + "`" + `gitee label edit` + "`" + `   — change color or rename
`
