// Package status implements the `gitee status` command, which provides a
// at-a-glance view of issues assigned to the current user and PRs awaiting
// their review in the current Gitee repository.
//
// The command mirrors `gh status`'s sectioned output but with two important
// Gitee-v5-specific adaptations:
//
//   - "Awaiting my review" is computed CLIENT-SIDE (fetch all open PRs, drop
//     ones where the current user is the author). Gitee v5 has no
//     `?reviewer=<me>` query parameter on /repos/{}/{}/pulls, so the
//     server-side filter doesn't exist.
//
//   - The "current repo" is auto-detected from `git remote get-url origin`
//     when invoked inside a git working tree. The user can override with
//     --repo (the global flag) for the non-git-dir case.
package status

import (
	"fmt"
	"os"

	"gitee-cli/cmd"
	"github.com/spf13/cobra"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show issues assigned to you and PRs awaiting your review",
	Long: `Show a focused, at-a-glance view of what's currently waiting on you
in a Gitee repository: issues assigned to you, and open pull requests that
are awaiting your review.

The "current repository" is auto-detected from the git working tree
(reads the 'origin' remote URL). When run outside a git working copy,
or when the origin remote points to a non-Gitee host, use --repo to
specify a repository explicitly.`,
	Example: `  # In a git repo on gitee.com
  gitee status

  # Explicit repo (works from any directory)
  gitee status --repo owner/repo

  # JSON output
  gitee status --repo owner/repo --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(statusAgentHelp)
			os.Exit(0) // short-circuit before cobra's flag validation
		}
		return nil
	},
	RunE: runStatus,
}
