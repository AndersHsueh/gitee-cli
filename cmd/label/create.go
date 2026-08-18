package label

import (
	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var createFlags struct {
	repo       string
	name       string
	labelColor string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new label",
	Example: `  gitee label create --repo owner/repo --name bug --color ff0000
  gitee label create -R owner/repo -n enhancement -c 00ff00`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(labelCreateAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		dc, err := cmd.NewDomainClient()
		if err != nil {
			return err
		}
		return dc.LabelCreate(createFlags.repo, createFlags.name, createFlags.labelColor)
	},
}

func init() {
	f := createCmd.Flags()
	f.StringVarP(&createFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&createFlags.name, "name", "n", "", "Label name (required)")
	f.StringVarP(&createFlags.labelColor, "color", "c", "gray", "Label color (6 hex digits, e.g. ff0000)")
}

const labelCreateAgentHelp = `# gitee label create — Agent Help

## Purpose
Add a new label to a repository. Labels are scoped to a single repo
and must be created before they can be applied to issues or PRs.

## Pre-conditions
- Authenticated with write access to the repo
- The label name does NOT already exist (returns 422 on conflict)

## Required flags
  --name / -n   Label name (required, non-empty)
  --color / -c  6 hex digits (with or without leading #). Default: gray

## Optional flags
  --repo / -R   owner/repo

## Idempotency
NOT idempotent. Re-running with the same name returns 422 (label exists).
Use ` + "`" + `gitee label edit` + "`" + ` to change color or rename.

## Atomicity
Atomic. Either the label is created, or no change happens.

## Side effects
- The new label is immediately available in the issue/PR creation UI
- Webhooks subscribed to "Label" events may fire

## Post-conditions (on success)
- A new label with the given name + color exists in the repo
- Stdout shows success + URL

## Output (stdout, table mode)
  ✓ Label created successfully
    Name:  bug
    Color: #ff0000
    URL:   https://gitee.com/owner/repo/labels/bug

## Common errors
- 422 Unprocessable: label name already exists
- 401 Unauthorized: token expired
- 403 Forbidden:    token lacks write scope

## Race condition warning ⚠️
The shared mutable state is the repo's label set, keyed by name.
- Two agents creating the same name concurrently: one wins, the other
  gets 422 (label exists). On 422, fall back to ` + "`" + `gitee label edit` + "`" + `
  if your intent was to update rather than create.
- A ` + "`" + `gitee label list` + "`" + ` snapshot may miss a label created by another
  agent in the list→create gap — never treat list output as proof of
  absence before creating.
- The new label can take 1-2 seconds to appear in list output. Do NOT
  re-list to verify creation; the success output of THIS command is
  authoritative.
Blind retry of a request that actually succeeded surfaces as 422, not
as a duplicate label — inspect the error before retrying.

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour.

## Example
  gitee label create -R myorg/myrepo -n "priority/high" -c ff0000

## See also
- ` + "`" + `gitee label list` + "`" + ` — see what already exists
- ` + "`" + `gitee label edit` + "`" + ` — modify an existing label
- ` + "`" + `gitee label clone` + "`" + ` — bulk-copy a label set from another repo
`
