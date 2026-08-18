package issue

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var createFlags struct {
	title     string
	body      string
	labels    string
	assignees string
	milestone int
	repo      string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Example: `  # Create an issue in the specified repo
  gitee issue create --repo owner/repo --title "Bug: crash on startup"

  # With body and labels
  gitee issue create --repo owner/repo --title "Feature request" \
    --body "Please add dark mode" --labels "enhancement,help wanted"`,
	// PreRunE fires BEFORE ValidateRequiredFlags, so an agent can request
	// --agent-help even when required flags (e.g. --title) are missing.
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(issueCreateAgentHelp)
		}
		return nil
	},
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVarP(&createFlags.title, "title", "t", "", "Issue title (required)")
	f.StringVarP(&createFlags.body, "body", "b", "", "Issue body (supports Markdown)")
	f.StringVarP(&createFlags.labels, "labels", "l", "", "Labels (comma-separated)")
	f.StringVarP(&createFlags.assignees, "assignees", "a", "", "Assignees (comma-separated usernames)")
	f.IntVarP(&createFlags.milestone, "milestone", "m", 0, "Milestone number")
	f.StringVarP(&createFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	_ = createCmd.MarkFlagRequired("title")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	// Resolve --repo: local flag > global --repo
	repo := createFlags.repo
	if repo == "" {
		repo = rt.Repo
	}
	if repo == "" {
		return fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid --repo format %q, expected owner/repo", repo)
	}
	owner, repoName := parts[0], parts[1]

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"repo":  repoName, // Gitee v5: repo is just the name, owner is in the URL path
		"title": createFlags.title,
	}
	if createFlags.body != "" {
		payload["body"] = createFlags.body
	}
	if createFlags.labels != "" {
		payload["labels"] = createFlags.labels
	}
	if createFlags.assignees != "" {
		payload["assignees"] = createFlags.assignees
	}
	if createFlags.milestone > 0 {
		payload["milestone_number"] = createFlags.milestone
	}

	var issue types.Issue
	// Gitee v5: POST /repos/{owner}/issues (NOT /repos/{owner}/{repo}/issues)
	path := api.IssuesCreate(owner)

	fmt.Printf("Creating issue in %s/%s...\n", owner, repoName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	if err := client.Post(ctx, path, payload, &issue); err != nil {
		return fmt.Errorf("creating issue: %w", err)
	}

	// Success output
	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s Issue #%s created\n", green("✓"), bold(issue.Number))
	fmt.Printf("  Title : %s\n", issue.Title)
	fmt.Printf("  State : %s\n", issue.State)
	fmt.Printf("  URL   : %s\n", color.CyanString(issue.HTMLURL))
	return nil
}

// issueCreateAgentHelp is the agent-targeted help for `gitee issue create`.
// It is intentionally long, structured, and includes machine-relevant
// details (idempotency, side effects, orchestration hints) that the
// default --help omits because they're noise for humans.
//
// Maintainer note: keep this in sync with the actual command behavior.
// When flags change, update the relevant section here.
const issueCreateAgentHelp = `# gitee issue create — Agent Help

## Purpose
Create a new issue in a Gitee repository. Use this when an automated check
(e.g. test agent) finds a defect, or when an agent needs to file a feature
request / task on behalf of the user.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Repository exists: --repo (or --owner + --repo) points to an accessible repo
- Write access: token must have write permission on the target repo
- --title is non-empty (enforced by Cobra MarkFlagRequired)

## Required flags
  --title / -t    Issue title (required, non-empty)

## Optional flags
  --body / -b       Issue body, Markdown supported
  --labels / -l     Comma-separated label names (must already exist on the repo)
  --assignees / -a  Comma-separated usernames
  --milestone / -m  Milestone number
  --repo / -R       owner/repo format (falls back to global --repo, then git remote)

## Idempotency
NOT idempotent. Each invocation creates a NEW issue with a fresh number.
Re-running this command creates a duplicate. There is no "create or update"
semantic on the Gitee API; use ` + "`" + `gitee issue edit` + "`" + ` to modify an existing one.

## Atomicity
Atomic. Either the issue is created and a number is returned, or no
change happens. There is no partial-success state.

## Side effects
- Email notification sent to all repo watchers
- Webhooks subscribed to "Issue" events fire
- The new issue is in "open" state by default
- The issue is visible immediately in the repo's issue list (web UI)

## Post-conditions (on success)
- An issue with a new unique number exists in the repo
- Stdout includes the issue number (#N) and URL
- The issue is in "open" state
- The new issue does NOT have a label/assignee unless --labels/--assignees were passed

## Output (stdout, table mode)
  Creating issue in owner/repo...
  ✓ Issue #42 created
    Title : Bug: ...
    State : open
    URL   : https://gitee.com/owner/repo/issues/42

## Parsing the issue number
The issue number is in the success line. Robust parsing:
  OUT=$(gitee issue create --repo owner/repo --title "..." --body "...")
  ISSUE_NUM=$(echo "$OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

## Common errors
- 401 Unauthorized:  token expired or invalid  →  gitee auth login
- 403 Forbidden:     token lacks write scope   →  check token permissions
- 404 Not Found:     repo doesn't exist        →  verify owner/repo spelling
- 422 Unprocessable: validation failed         →  check title is non-empty, labels exist

## Race condition warning ⚠️
DO NOT call ` + "`" + `gitee issue list` + "`" + ` immediately after to verify creation.
Gitee API indexing can lag 1-2 seconds; a follow-up list may not return
the new issue even though it was created. Use the issue number from THIS
output directly — do not re-list to find it.

## Next steps (orchestration hints)
After a successful issue create, an agent typically wants to:
1. Record the issue number from stdout (do NOT re-list to find it)
2. If this is a test-reported bug: continue with the test report workflow
3. If you need to add details: ` + "`" + `gitee issue edit {N} --body "..."` + "`" + ` (NOT create)
4. If the issue should be closed: ` + "`" + `gitee issue close {N}` + "`" + `
5. To link a future PR to this issue: include ` + "`" + `Closes #N` + "`" + ` in the PR body

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example (agent-friendly, closing the loop)
  # 1. Create the issue
  OUT=$(gitee issue create --repo owner/repo \
    --title "Bug: crash on startup" --body "Steps to reproduce...")
  ISSUE_NUM=$(echo "$OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')

  # 2. Add labels (separate call, idempotent)
  gitee issue edit "$ISSUE_NUM" --labels "bug,priority/high"

  # 3. After fixing, create a PR that references this issue
  gitee pr create --repo owner/repo --title "fix: handle null pointer" \
    --source fix/null-pointer --target main --body "Closes #${ISSUE_NUM}"

  # 4. After PR is merged, close the issue
  gitee issue close "$ISSUE_NUM"

## See also
- ` + "`" + `gitee issue view N` + "`" + `        — read an issue
- ` + "`" + `gitee issue edit N` + "`" + `        — modify title/body/labels (idempotent)
- ` + "`" + `gitee issue close N` + "`" + `       — close an issue
- ` + "`" + `gitee issue comment N` + "`" + `     — add a comment
- ` + "`" + `gitee pr create` + "`" + `           — create a PR (link via "Closes #N" in body)
- ` + "`" + `gitee --agent-help` + "`" + `        — overview of all commands with agent help
`
