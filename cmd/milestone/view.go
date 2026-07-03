package milestone

import (
	"gitee-cli/cmd"
	"gitee-cli/internal/handler"
	"gitee-cli/internal/utils"

	"github.com/spf13/cobra"
)

var viewFlags struct {
	repo   string
	number int64
}

var viewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View a single milestone",
	Example: `  gitee milestone view --repo owner/repo --number 1
  gitee milestone view -R owner/repo 2`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(milestoneViewAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		// Accept the milestone number as either the flag value or the
		// first positional argument (gh-style "view 2").
		parsed, err := utils.ParseInt64Arg(viewFlags.number, args, "milestone")
		if err != nil {
			return err
		}
		return handler.MilestoneView(viewFlags.repo, int(parsed))
	},
}

func init() {
	f := viewCmd.Flags()
	f.StringVarP(&viewFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.Int64VarP(&viewFlags.number, "number", "n", 0, "Milestone number (required, or pass as positional)")
}

const milestoneViewAgentHelp = `# gitee milestone view — Agent Help

## Purpose
Read a single milestone's full metadata: number, title, state, open /
closed issue counts, description, due date, and URL.

## Pre-conditions
- Authenticated
- Milestone exists in the repo (returns 404 otherwise)

## Required flags
  --number / -n   Milestone number (or pass as positional: ` + "`" + `gitee milestone view 2` + "`" + `)

## Optional flags
  --repo / -R   owner/repo

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None.

## Post-conditions (on success)
- Stdout shows the milestone's full details

## Output (stdout, table mode)
  ─ Milestone Details (number=2)
    Title:  v1.1
    State:  open
    Open:   4
    Closed: 0
    Desc:   Second minor release
    Due:    2026-12-15
    URL:    https://gitee.com/owner/repo/milestones/2

## Common errors
- 404 Not Found: milestone does not exist in this repo
- 401 Unauthorized: token missing or expired

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour.

## Example
  gitee milestone view -R owner/repo -n 2

## See also
- ` + "`" + `gitee milestone list` + "`" + `   — enumerate milestones
- ` + "`" + `gitee milestone edit` + "`" + `   — change title, description, due date, or state
`