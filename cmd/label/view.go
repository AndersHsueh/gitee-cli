package label

import (
	"fmt"
	"os"

	"gitee-cli/cmd"
	"gitee-cli/internal/handler"
	"github.com/spf13/cobra"
)

var viewFlags struct {
	repo string
	name string
}

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View a single label",
	Example: `  gitee label view --repo owner/repo --name bug
  gitee label view -R owner/repo -n enhancement`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(labelViewAgentHelp)
			os.Exit(0)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		return handler.LabelView(viewFlags.repo, viewFlags.name)
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
