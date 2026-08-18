package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var viewFlags struct {
	owner string
	repo  string
}

var viewCmd = &cobra.Command{
	Use:     "view <number>",
	Short:   "View details of an issue",
	Args:    cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	Example: `  gitee issue view 42 --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(issueViewAgentHelp)
		}
		return nil
	},
	RunE: runView,
}

func init() {
	f := viewCmd.Flags()
	f.StringVar(&viewFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&viewFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	owner, repoName, err := resolveRepo(viewFlags.owner, viewFlags.repo)
	if err != nil {
		return err
	}
	number := args[0]

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.IssuesGet(owner, repoName, number)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var issue types.Issue
	if err := client.Get(ctx, path, &issue); err != nil {
		return fmt.Errorf("viewing issue: %w", err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(issue, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s  #%s\n\n", bold(issue.Title), issue.Number)
	fmt.Printf("  State  : %s\n", issue.State)
	fmt.Printf("  Author : %s\n", issue.User.Login)
	if issue.Assignee != nil {
		fmt.Printf("  Assignee: %s\n", issue.Assignee.Login)
	}
	if len(issue.Labels) > 0 {
		names := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			names[i] = l.Name
		}
		fmt.Printf("  Labels : %s\n", strings.Join(names, ", "))
	}
	fmt.Printf("  URL    : %s\n", cyan(issue.HTMLURL))
	fmt.Printf("  Created: %s\n", issue.CreatedAt.Format("2006-01-02 15:04"))
	if issue.Body != "" {
		fmt.Printf("\n%s\n", issue.Body)
	}
	return nil
}

// issueViewAgentHelp is the agent-targeted help for `gitee issue view`.
// It is intentionally long, structured, and includes machine-relevant
// details (idempotency, side effects, orchestration hints) that the
// default --help omits because they're noise for humans.
//
// Maintainer note: keep this in sync with the actual command behavior.
// When flags change, update the relevant section here.
const issueViewAgentHelp = `# gitee issue view — Agent Help

## Purpose
Fetch and display the full details of a single Gitee issue — title,
state, author, assignees, labels, body, and timestamps. Use this to
inspect an issue that was just discovered via ` + "`" + `gitee issue list` + "`" + `,
or to read an issue referenced by number from a PR / commit message.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Repository exists: --repo (or --owner + --repo) points to an accessible repo
- Read access: token must have read permission on the target repo
  (private repos require projects or higher)
- Issue number is valid: the issue must exist in the target repo

## Required arguments
  <number>  Issue number (integer, e.g. 42). Must be a positional arg
            after the verb. Use ` + "`" + `gitee issue list` + "`" + ` to discover numbers.

## Optional flags
  --repo / -R       owner/repo format (falls back to global --repo, then git remote)
  --output / -o     text (default) | json — json prints the full types.Issue
                    struct with all fields (title, body, state, labels,
                    assignees, milestones, html_url, created_at, updated_at,
                    comments, etc.) for downstream pipelines

## Idempotency
Idempotent (read-only). Re-running this command does not change any
remote state. The output may differ between calls if the issue has
been edited between them (e.g. title changed, labels added).

## Atomicity
Atomic. Either the full issue is returned, or an error is raised.
There is no partial-success state.

## Side effects
None. This is a pure read. No notifications, no webhooks, no
mutations on the issue, and no "viewed" event is fired.

## Post-conditions (on success)
- Stdout contains the issue details (text or JSON) — see Output sections
- No state change on the issue (no view counter, no last-read tracking)
- The data returned reflects the API at the time of the call

## Output (stdout, text mode)
  <title>  #<number>

    State   : open | closed
    Author  : <username>
    Assignee: <username>          (only if assigned)
    Labels  : <name>, <name>      (only if any)
    URL     : https://gitee.com/owner/repo/issues/<number>
    Created : 2026-06-28 14:32

  <body markdown, if any>

## Output (stdout, json mode)
  -o json prints the full types.Issue struct (all fields) to stdout,
  one object. Suitable for downstream pipelines.

## Parsing hints
  # Get the state of a specific issue:
  gitee issue view 42 --repo owner/repo -o json | jq -r '.state'

  # Get the title:
  gitee issue view 42 --repo owner/repo -o json | jq -r '.title'

  # Get the body (may be empty):
  gitee issue view 42 --repo owner/repo -o json | jq -r '.body // ""'

  # Get the labels:
  gitee issue view 42 --repo owner/repo -o json | jq -r '.labels[].name'

  # Get the assignees:
  gitee issue view 42 --repo owner/repo -o json | jq -r '.assignee.login // empty'

## Common errors
- 401 Unauthorized:  token expired or invalid    →  gitee auth login
- 403 Forbidden:     token lacks read scope      →  check token permissions
- 404 Not Found:     repo or issue doesn't exist →  verify owner/repo + number
- 422 Unprocessable: issue number malformed      →  must be a positive integer

## Race condition warning ⚠️
The shared mutable state is the issue itself: state, title, labels,
assignees, and the comment count can all change the moment after
this GET returns. The classic failure is view-then-act: you read
` + "`" + `state: open` + "`" + `, decide to close, but another agent closed (or
reopened) the issue in the gap — ` + "`" + `gitee issue close` + "`" + ` treats
"already closed" as a silent no-op, so nothing flags the flip. If
any downstream action depends on what you see here (close only if
open, comment only if unassigned), re-run ` + "`" + `gitee issue view {N}` + "`" + `
immediately BEFORE the mutating call and decide from the fresh read.
The ` + "`" + `comments` + "`" + ` field is a count snapshot; for the actual
conversation use ` + "`" + `gitee issue comments {N}` + "`" + `. Re-running this
command is always safe (read-only).

## Next steps (orchestration hints)
After viewing an issue, an agent typically wants to:
1. Edit it: ` + "`" + `gitee issue edit {N} --title '...'` + "`" + ` (or --body, --labels, etc.)
2. Comment on it: ` + "`" + `gitee issue comment {N} --body '...'` + "`" + `
3. Read all comments: ` + "`" + `gitee issue comments {N}` + "`" + `
4. Close it: ` + "`" + `gitee issue close {N}` + "`" + ` (or ` + "`" + `gitee issue reopen {N}` + "`" + `)
5. If this issue describes a bug to fix, create a PR that closes it:
   ` + "`" + `gitee pr create --body 'Closes #{N} ...'` + "`" + `
6. To see the cross-repo activity summary, ` + "`" + `gitee status` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Quick check
  gitee issue view 42 --repo owner/repo

  # Capture full state for an automated decision
  STATE=$(gitee issue view 42 --repo owner/repo -o json | jq -r '.state')
  if [ "$STATE" = "open" ]; then
    gitee issue close 42 --repo owner/repo
  fi

  # Get the author
  AUTHOR=$(gitee issue view 42 --repo owner/repo -o json | jq -r '.user.login')

## See also
- ` + "`" + `gitee issue list` + "`" + `        — discover issue numbers in a repo
- ` + "`" + `gitee issue edit N` + "`" + `     — modify an existing issue
- ` + "`" + `gitee issue close N` + "`" + `    — close an issue
- ` + "`" + `gitee issue reopen N` + "`" + `   — reopen a closed issue
- ` + "`" + `gitee issue comment N` + "`" + `  — add a comment
- ` + "`" + `gitee issue comments N` + "`" + ` — list all comments on an issue
- ` + "`" + `gitee pr create` + "`" + `        — create a PR (link via "Closes #N" in body)
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands with agent help
`
