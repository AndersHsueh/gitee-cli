package label

import (
	"gitee-cli/cmd"
	"gitee-cli/internal/handler"

	"github.com/spf13/cobra"
)

var listFlags struct {
	repo   string
	limit  int
	filter string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List labels in a repository",
	Example: `  gitee label list --repo owner/repo
  gitee label list -R owner/repo --limit 50`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(labelListAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		return handler.LabelList(listFlags.repo, listFlags.filter, listFlags.limit)
	},
}

func init() {
	f := listCmd.Flags()
	f.StringVarP(&listFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.IntVarP(&listFlags.limit, "limit", "L", 100, "Maximum number of labels to fetch")
	f.StringVar(&listFlags.filter, "color", "", "Filter labels by hex color (without #)")
}

const labelListAgentHelp = `# gitee label list — Agent Help

## Purpose
List all labels defined in a repository. Use this to discover the
label taxonomy, check whether a label exists, or enumerate options
before bulk operations.

## Pre-conditions
- Authenticated (gitee auth status)
- Repository is readable

## Required flags
  --repo / -R   owner/repo (falls back to global --repo)

## Optional flags
  --limit / -L   Maximum number of labels (default 100, max 100 per page)
  --color        Filter by exact hex color (without #)

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None.

## Post-conditions (on success)
- Stdout contains a table of labels (NAME, COLOR, URL)
- Repo state is unchanged

## Output (stdout, table mode)
  NAME        COLOR     URL
  bug         #ff0000   https://gitee.com/owner/repo/labels/bug
  enhancement #00ff00   https://gitee.com/owner/repo/labels/enhancement

## Common errors
- 404 Not Found: repo does not exist or is private
- 401 Unauthorized: token missing or expired

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour authenticated.

## Example
  gitee label list -R myorg/myrepo --color ff0000

## See also
- ` + "`" + `gitee label create` + "`" + `    — add a new label
- ` + "`" + `gitee issue label list` + "`" + ` — same behavior via the legacy path
`
