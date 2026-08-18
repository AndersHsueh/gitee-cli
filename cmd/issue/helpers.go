package issue

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

func resolveRepo(owner, repo string) (string, string, error) {
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	return utils.ResolveRepo(owner, repo, nil)
}

var editFlags struct {
	owner     string
	repo      string
	title     string
	body      string
	labels    string
	assignees string
	milestone int
}

var closeFlags struct {
	owner string
	repo  string
}

var reopenFlags struct {
	owner string
	repo  string
}

var commentFlags struct {
	owner string
	repo  string
	body  string
}

var editCmd = &cobra.Command{
	Use:     "edit <number>",
	Short:   "Edit an issue",
	Example: `  gitee issue edit 42 --repo owner/repo --title "New title"`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(issueEditAgentHelp)
		}
		return nil
	},
	RunE: runEdit,
}

var closeCmd = &cobra.Command{
	Use:     "close <number>",
	Short:   "Close an issue",
	Example: `  gitee issue close 42 --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(issueCloseAgentHelp)
		}
		return nil
	},
	RunE: runClose,
}

var reopenCmd = &cobra.Command{
	Use:     "reopen <number>",
	Short:   "Reopen an issue",
	Example: `  gitee issue reopen 42 --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(issueReopenAgentHelp)
		}
		return nil
	},
	RunE: runReopen,
}

var commentCmd = &cobra.Command{
	Use:     "comment <number>",
	Short:   "Add a comment to an issue",
	Example: `  gitee issue comment 42 --repo owner/repo --body "Comment text"`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(issueCommentAgentHelp)
		}
		return nil
	},
	RunE: runComment,
}

func init() {
	f := editCmd.Flags()
	f.StringVar(&editFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&editFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVar(&editFlags.title, "title", "", "New title")
	f.StringVar(&editFlags.body, "body", "", "New body")
	f.StringVar(&editFlags.labels, "labels", "", "Comma-separated labels")
	f.StringVar(&editFlags.assignees, "assignees", "", "Comma-separated assignees")
	f.IntVar(&editFlags.milestone, "milestone", 0, "Milestone number")

	f = closeCmd.Flags()
	f.StringVar(&closeFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&closeFlags.repo, "repo", "R", "", "Repository name or owner/repo")

	f = reopenCmd.Flags()
	f.StringVar(&reopenFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&reopenFlags.repo, "repo", "R", "", "Repository name or owner/repo")

	f = commentCmd.Flags()
	f.StringVar(&commentFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&commentFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVar(&commentFlags.body, "body", "", "Comment body (required)")
	_ = commentCmd.MarkFlagRequired("body")
}

func runEdit(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("issue number is required")
	}
	owner, repoName, err := resolveRepo(editFlags.owner, editFlags.repo)
	if err != nil {
		return err
	}
	issueNum := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("repo", repoName)
	if editFlags.title != "" {
		form.Set("title", editFlags.title)
	}
	if editFlags.body != "" {
		form.Set("body", editFlags.body)
	}
	if editFlags.labels != "" {
		form.Set("labels", editFlags.labels)
	}
	if editFlags.assignees != "" {
		form.Set("assignee", editFlags.assignees)
	}
	if editFlags.milestone > 0 {
		form.Set("milestone", fmt.Sprintf("%d", editFlags.milestone))
	}

	path := api.IssuesEdit(owner, repoName, issueNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var issue types.Issue
	// Gitee v5 issue update endpoint takes formData (NOT JSON) and requires
	// `repo` as a form field. See the IssuesEdit builder doc.
	if err := client.PatchForm(ctx, path, form, &issue); err != nil {
		return fmt.Errorf("editing issue: %w", err)
	}
	fmt.Printf("Issue #%s updated\n", issueNum)
	return nil
}

func runClose(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("issue number is required")
	}
	owner, repoName, err := resolveRepo(closeFlags.owner, closeFlags.repo)
	if err != nil {
		return err
	}
	issueNum := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("repo", repoName)
	form.Set("state", "closed")
	path := api.IssuesEdit(owner, repoName, issueNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var issue types.Issue
	// Gitee v5 issue update endpoint takes formData (NOT JSON) and requires
	// `repo` as a form field. See the IssuesEdit builder doc. The remote's
	// formatStateTransitionError still wraps the error for the (rare) case
	// where Gitee v5 returns 404 for an enterprise/project container that
	// the API cannot route to.
	if err := client.PatchForm(ctx, path, form, &issue); err != nil {
		return formatStateTransitionError(err, owner, repoName, issueNum, "close")
	}
	fmt.Printf("Issue #%s closed\n", issueNum)
	return nil
}

func runReopen(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("issue number is required")
	}
	owner, repoName, err := resolveRepo(reopenFlags.owner, reopenFlags.repo)
	if err != nil {
		return err
	}
	issueNum := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("repo", repoName)
	form.Set("state", "open")
	path := api.IssuesEdit(owner, repoName, issueNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var issue types.Issue
	// Gitee v5 issue update endpoint takes formData (NOT JSON) and requires
	// `repo` as a form field. See the IssuesEdit builder doc. The remote's
	// formatStateTransitionError still wraps the error for the (rare) case
	// where Gitee v5 returns 404 for an enterprise/project container that
	// the API cannot route to.
	if err := client.PatchForm(ctx, path, form, &issue); err != nil {
		return formatStateTransitionError(err, owner, repoName, issueNum, "reopen")
	}
	fmt.Printf("Issue #%s reopened\n", issueNum)
	return nil
}

// issueWebURL builds the canonical Gitee web URL for an issue, honoring
// the effective --hostname / config active_host so enterprise deployments
// are not silently redirected to gitee.com. Used by the 404-workaround
// hint emitted when the API rejects state transitions on issue
// containers it cannot locate (see formatStateTransitionError).
func issueWebURL(owner, repo, number string) string {
	return fmt.Sprintf("https://%s/%s/%s/issues/%s", cmd.ResolveHostname(), owner, repo, number)
}

// stateTransitionVerbs maps a state-transition action ("close", "reopen")
// to (1) the present-participle form used in the error prefix
// ("closing issue: …") and (2) the imperative form used in the
// workaround instruction ("close the issue manually at: …").
//
// Keep this in sync with runClose / runReopen, which already hard-code
// the same strings.
var stateTransitionVerbs = map[string]struct{ ing, imperative string }{
	"close":  {"closing", "close"},
	"reopen": {"reopening", "reopen"},
}

// formatStateTransitionError wraps a state-transition (close/reopen) PATCH
// failure with a human-friendly hint when the underlying cause is the
// known Gitee v5 routing limitation: issues belonging to an enterprise
// project container cannot be state-toggled via
// /repos/{owner}/{repo}/issues/{number} — the API returns 404 with a
// body containing the project-or-enterprise signature. In that case we
// surface the web UI URL so the user has a working alternative.
//
// For any other error we preserve the original "closing issue: %w" /
// "reopening issue: %w" wrapping by looking up the verb from `action`
// so the user-facing prefix matches what the runX function was doing.
func formatStateTransitionError(err error, owner, repo, number, action string) error {
	if err == nil {
		return nil
	}
	verbs, ok := stateTransitionVerbs[action]
	if !ok {
		// Defensive fallback for unknown actions — should not happen
		// because runClose/runReopen are the only callers.
		return fmt.Errorf("%s issue: %w", action, err)
	}
	if api.IsProjectOrEnterpriseNotFound(err) {
		return fmt.Errorf(
			"%s issue: %w\n"+
				"\n"+
				"Gitee v5 cannot toggle the state of this issue via PATCH on\n"+
				"/repos/{owner}/{repo}/issues/{number} (HTTP 404: project or enterprise).\n"+
				"This usually means the issue lives in an enterprise/project\n"+
				"container rather than directly under the repo. As a workaround,\n"+
				"%s the issue manually at:\n"+
				"  %s",
			verbs.ing, err, verbs.imperative, issueWebURL(owner, repo, number),
		)
	}
	return fmt.Errorf("%s issue: %w", verbs.ing, err)
}

// projectOrEnterprise404Note is the "Common errors" section snippet
// shared by the issue close and issue reopen --agent-help. Both verbs
// hit the same PATCH /repos/{owner}/{repo}/issues/{number} route and
// therefore share the same 404 / "project or enterprise" failure mode.
// Kept as a single constant so the wording stays in sync.
const projectOrEnterprise404Note = `- 404 Not Found:     body says "project or enterprise"
                  →  Gitee v5 cannot PATCH the state of an issue that
                     lives in an enterprise/project container. Workaround:
                     toggle the state at
                     ` + "`" + `https://gitee.com/{owner}/{repo}/issues/{number}` + "`" + `
                     (web UI). gitee-cli surfaces this URL in the error
                     message automatically — look for the line beginning
                     "Gitee v5 cannot toggle the state".`

func runComment(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("issue number is required")
	}
	owner, repoName, err := resolveRepo(commentFlags.owner, commentFlags.repo)
	if err != nil {
		return err
	}
	issueNum := args[0]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("body", commentFlags.body)
	path := api.IssuesComment(owner, repoName, issueNum)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// Gitee v5 issue comment endpoint takes formData (NOT JSON). Sending a
	// JSON body here is rejected with HTTP 401 before reaching the business
	// layer — see the IssuesComment builder doc. The same formData contract
	// that v0.12.6 applied to issue close/reopen/edit applies here.
	var comment types.IssueComment
	if err := client.PostForm(ctx, path, form, &comment); err != nil {
		return fmt.Errorf("commenting on issue: %w", err)
	}
	fmt.Printf("Comment added to issue #%s\n", issueNum)
	return nil
}

// issueEditAgentHelp is the agent-targeted help for `gitee issue edit`.
// It is intentionally long, structured, and includes machine-relevant
// details (idempotency, side effects, orchestration hints) that the
// default --help omits because they're noise for humans.
//
// Maintainer note: keep this in sync with the actual command behavior.
// When flags change, update the relevant section here.
const issueEditAgentHelp = `# gitee issue edit — Agent Help

## Purpose
Modify an existing Gitee issue in place — change its title, body, labels,
assignees, or milestone. Use this as the "update" half of create-then-edit
flows, or for batch-triage where you set labels/assignees on a list of
issues returned by ` + "`" + `gitee issue list` + "`" + `.

## Pre-conditions
- Authenticated: token present in config (~~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Repository exists: --repo (or --owner + --repo) points to an accessible repo
- Write access: token must have write permission on the target repo
- Issue exists: the issue number must be valid in the target repo

## Required arguments
  <number>  Issue number (integer, e.g. 42). Must be a positional arg
            after the verb. Use ` + "`" + `gitee issue list` + "`" + ` to discover numbers.

## Optional flags
  --title        New title (replaces the existing one)
  --body         New body (replaces the existing one; Markdown supported)
  --labels       Comma-separated label names (replaces the existing labels)
  --assignees    Comma-separated usernames (replaces the existing assignees)
  --milestone    Milestone number
  --owner        Repository owner
  --repo / -R    Repository name or owner/repo

Note: flags that are not passed are left UNCHANGED on the issue. This is
critical — calling ` + "`" + `gitee issue edit 42 --title '...'` + "`" + ` does NOT clear
the body or the labels. To clear a field, pass an empty value explicitly
(e.g. ` + "`" + `--body ''` + "`" + `) — but note that Gitee v5 currently treats an
empty string as "no change" for body; to clear the body, use the web UI.

## Idempotency
Idempotent for the same inputs. Re-running with the same flags results
in the same end state. However, calling it with DIFFERENT inputs each
time mutates the issue toward the most recent call — there is no
merge/3-way semantics.

## Atomicity
Atomic at the field-set level: all passed flags are applied together
in a single PATCH request. There is no partial-success state where some
fields were updated and others failed.

## Side effects
- The issue's updated_at timestamp advances
- Email notification sent to watchers if title/body/assignee/labels change
- Webhooks subscribed to "Issue" events fire (per the changed fields)
- The issue's state is preserved (this command does NOT close/reopen;
  use ` + "`" + `gitee issue close` + "`" + ` or ` + "`" + `gitee issue reopen` + "`" + ` for that)

## Post-conditions (on success)
- The named fields on the issue are updated
- Stdout: "Issue #N updated"
- The issue number is unchanged (this command does not create or delete)
- Other fields not passed are untouched

## Common errors
- 401 Unauthorized:  token expired or invalid    →  gitee auth login
- 403 Forbidden:     token lacks write scope     →  check token permissions
- 404 Not Found:     repo or issue doesn't exist →  verify owner/repo + number
- 422 Unprocessable: bad label name or assignee  →  check spelling, label must exist

## Race condition warning ⚠️
The shared mutable state is the issue itself (title, body, labels,
assignees, milestone). Edits are last-writer-wins with NO merge
semantics: if another agent or user edits the same issue between
your ` + "`" + `gitee issue view {N}` + "`" + ` and this call, your PATCH
overwrites their newer values for every flag you pass. ` + "`" + `--labels` + "`" + `
and ` + "`" + `--assignees` + "`" + ` replace the WHOLE set — two agents that each
"add one label" concurrently will silently drop one of them. If your
new value derives from the current content (append to the body, add
to the labels), re-read with ` + "`" + `gitee issue view {N}` + "`" + ` immediately
before editing and send the merged full value. Retrying with the
same inputs is safe (see Idempotency).

## Next steps (orchestration hints)
After editing an issue, an agent typically wants to:
1. Verify the change: ` + "`" + `gitee issue view {N}` + "`" + `
2. Add a note about the change: ` + "`" + `gitee issue comment {N} --body '...'` + "`" + `
3. If this is a test-reported bug now triaged, create a fix PR:
   ` + "`" + `gitee pr create --body 'Closes #{N} ...'` + "`" + `
4. If the issue is resolved (fix merged, won't fix, etc.), close it:
   ` + "`" + `gitee issue close {N}` + "`" + `
5. To set the issue's state to closed in a single call, use
   ` + "`" + `gitee issue close` + "`" + ` instead — it has the same auth and
   pre-conditions but is more semantically clear.

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Implementation notes (Gitee v5 quirk)
Gitee v5's issue update endpoint is shaped DIFFERENTLY from the read
endpoint. The READ path is ` + "`" + `/repos/{owner}/{repo}/issues/{number}` + "`" + `
(matches GitHub), but the WRITE path drops {repo} from the URL and
instead accepts ` + "`" + `repo` + "`" + ` as a formData parameter alongside the
fields being changed. All update fields (title, body, state, assignee,
collaborators, milestone, labels, ...) are also formData, NOT JSON body.

Concretely, this command sends:

  PATCH /repos/{owner}/issues/{number}                <-- no {repo} in path
  Content-Type: application/x-www-form-urlencoded
  repo=<repo>&title=...&body=...&assignee=...&milestone=...&labels=...

If you ever see ` + "`" + `HTTP 400 "state does not have a valid value"` + "`" + ` or
` + "`" + `HTTP 404 "project or enterprise"` + "`" + ` after this command, the most
likely cause is a regression that re-introduced the JSON body / wrong
path — re-check ` + "`" + `internal/api/client.go` + "`" + ` (PatchForm) and
` + "`" + `internal/api/endpoints.go` + "`" + ` (IssuesEdit) together.

## Example
  # Triage: add the "bug" label and assign to alice
  gitee issue edit 42 --repo owner/repo --labels "bug,priority/high" \
    --assignees alice

  # Rewrite the title and body
  gitee issue edit 42 --repo owner/repo \
    --title "Bug: crash on startup (regression in v0.10.0)" \
    --body "Steps to reproduce..."

  # Attach to a milestone
  gitee issue edit 42 --repo owner/repo --milestone 7

## See also
- ` + "`" + `gitee issue view N` + "`" + `       — read full details for one issue
- ` + "`" + `gitee issue list` + "`" + `        — discover issue numbers
- ` + "`" + `gitee issue close N` + "`" + `    — close the issue (state change)
- ` + "`" + `gitee issue reopen N` + "`" + `   — reopen the issue
- ` + "`" + `gitee issue comment N` + "`" + `  — add a comment (use for notes/audit trail)
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands with agent help
`

// issueCloseAgentHelp is the agent-targeted help for `gitee issue close`.
// It is intentionally long, structured, and includes machine-relevant
// details (idempotency, side effects, orchestration hints) that the
// default --help omits because they're noise for humans.
//
// Maintainer note: keep this in sync with the actual command behavior.
// When flags change, update the relevant section here.
const issueCloseAgentHelp = `# gitee issue close — Agent Help

## Purpose
Close an existing Gitee issue by transitioning its state to "closed".
Use this when an issue is resolved (fix merged), won't fix, duplicate,
or otherwise no longer actionable.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Repository exists: --repo (or --owner + --repo) points to an accessible repo
- Write access: token must have write permission on the target repo
- Issue exists: the issue number must be valid in the target repo
- The action is REVERSIBLE — you can re-open with ` + "`" + `gitee issue reopen {N}` + "`" + `

## Required arguments
  <number>  Issue number (integer, e.g. 42). Must be a positional arg
            after the verb. Use ` + "`" + `gitee issue list` + "`" + ` to discover numbers.

## Optional flags
  --owner        Repository owner
  --repo / -R    Repository name or owner/repo

## Idempotency
Idempotent in effect (not in result). Re-running this command on an
already-closed issue is a no-op server-side and still returns success,
but the closed_at timestamp does NOT advance. The Gitee API treats
"close a closed issue" as a no-op rather than an error.

## Atomicity
Atomic. Either the issue transitions to closed (with closed_at set),
or no change happens. There is no partial-success state.

## Side effects
- The issue's state is set to "closed" and closed_at is set to "now"
- Email notification sent to watchers
- Webhooks subscribed to "Issue" events fire (state: closed)
- If the closing issue is referenced by an open PR via "Closes #N",
  the PR is NOT auto-closed by Gitee — you must merge the PR separately
- The close action is REVERSIBLE: ` + "`" + `gitee issue reopen {N}` + "`" + `
  re-opens the issue and clears closed_at
- The issue is removed from the default "open" issue list and moves
  to the "closed" filter

## Post-conditions (on success)
- The issue's state is "closed"
- Stdout: "Issue #N closed"
- The issue's number is unchanged
- closed_at timestamp is set to "now"

## Common errors
- 401 Unauthorized:  token expired or invalid    →  gitee auth login
- 403 Forbidden:     token lacks write scope     →  check token permissions
- 404 Not Found:     repo or issue doesn't exist →  verify owner/repo + number
` + projectOrEnterprise404Note + `

## Race condition warning ⚠️
The shared mutable state is the issue's ` + "`" + `state` + "`" + ` field on the
server. Between your last ` + "`" + `gitee issue view {N}` + "`" + ` / ` + "`" + `gitee issue list` + "`" + `
and this close call, another agent or user may have already closed
(or reopened) the issue. Closing an already-closed issue is a silent
no-op server-side (see Idempotency) — the exit code will NOT tell
you the state had already flipped. If the close decision depends on
the current state (e.g. "close only if still open"), re-check with
` + "`" + `gitee issue view {N}` + "`" + ` immediately before this call. Retrying a
failed close is safe. In multi-agent triage, serialize close/reopen
transitions per issue number to avoid ping-pong.

## Reversibility
This command is REVERSIBLE. To re-open a closed issue, run
` + "`" + `gitee issue reopen {N} --repo owner/repo` + "`" + `. The reopen restores
the issue to "open" state and clears closed_at. Issue comments and
history are preserved across close/reopen cycles.

## Next steps (orchestration hints)
After closing an issue, an agent typically wants to:
1. Verify the state: ` + "`" + `gitee issue view {N}` + "`" + ` (state should be "closed")
2. If you closed by mistake, immediately: ` + "`" + `gitee issue reopen {N}` + "`" + `
3. To record why it was closed, add a final comment BEFORE closing:
   ` + "`" + `gitee issue comment {N} --body 'Closing: fixed in PR #M'` + "`" + `
4. To clean up labels, run ` + "`" + `gitee issue edit {N} --labels ''` + "`" + ` first
   (Gitee treats an empty labels string as "no change" — to remove
   all labels you must use the web UI)
5. For the inverse (open a new issue): ` + "`" + `gitee issue create` + "`" + `

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Implementation notes (Gitee v5 quirk)
The wire shape of this command is non-obvious. Gitee v5's issue update
endpoint is shaped DIFFERENTLY from the read endpoint:

  PATCH /repos/{owner}/issues/{number}                <-- no {repo} in path
  Content-Type: application/x-www-form-urlencoded
  repo=<repo>&state=closed

The READ path is ` + "`" + `/repos/{owner}/{repo}/issues/{number}` + "`" + ` (GitHub-style),
but the WRITE path drops {repo} from the URL and instead accepts ` + "`" + `repo` + "`" + `
as a formData parameter. All update fields (including ` + "`" + `state` + "`" + `) are
formData, NOT JSON body.

If you ever see ` + "`" + `HTTP 400 "state does not have a valid value"` + "`" + ` or
` + "`" + `HTTP 404 "project or enterprise"` + "`" + ` after this command, the most
likely cause is a regression that re-introduced the JSON body / wrong
path — re-check ` + "`" + `internal/api/client.go` + "`" + ` (PatchForm) and
` + "`" + `internal/api/endpoints.go` + "`" + ` (IssuesEdit) together.

The same wiring is used by ` + "`" + `gitee issue reopen` + "`" + ` and ` + "`" + `gitee issue edit` + "`" + `;
all three share the v0.12.6 fix.

## Example
  # Close issue 42
  gitee issue close 42 --repo owner/repo

  # Atomic close loop: comment, then close
  gitee issue comment 42 --repo owner/repo --body "Closing as duplicate of #41"
  gitee issue close 42 --repo owner/repo

  # Close after a fix PR is merged
  PR=$(gitee pr list --repo owner/repo --state merged -o json | jq -r '.[0].number')
  ISSUE=$(gitee pr view "$PR" --repo owner/repo -o json | jq -r '.body' | grep -oE 'Closes #[0-9]+' | head -1 | tr -d 'Closes #')
  if [ -n "$ISSUE" ]; then
    gitee issue close "$ISSUE" --repo owner/repo
  fi

## See also
- ` + "`" + `gitee issue view N` + "`" + `       — read full details for one issue
- ` + "`" + `gitee issue list` + "`" + `        — discover issue numbers
- ` + "`" + `gitee issue reopen N` + "`" + `   — inverse of this command
- ` + "`" + `gitee issue edit N` + "`" + `     — change title/body/labels (does NOT close)
- ` + "`" + `gitee issue comment N` + "`" + `  — add a comment (recommended before close)
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands with agent help
`

// issueReopenAgentHelp is the agent-targeted help for `gitee issue reopen`.
// It is intentionally long, structured, and includes machine-relevant
// details (idempotency, side effects, orchestration hints) that the
// default --help omits because they're noise for humans.
//
// Maintainer note: keep this in sync with the actual command behavior.
// When flags change, update the relevant section here.
const issueReopenAgentHelp = `# gitee issue reopen — Agent Help

## Purpose
Re-open a previously-closed Gitee issue by transitioning its state
back to "open". Use this when an issue was closed prematurely, was
regressed, or needs further work.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Repository exists: --repo (or --owner + --repo) points to an accessible repo
- Write access: token must have write permission on the target repo
- Issue exists: the issue number must be valid in the target repo
- The action is REVERSIBLE — you can close again with
  ` + "`" + `gitee issue close {N}` + "`" + `

## Required arguments
  <number>  Issue number (integer, e.g. 42). Must be a positional arg
            after the verb. Use ` + "`" + `gitee issue list` + "`" + ` to discover numbers.

## Optional flags
  --owner        Repository owner
  --repo / -R    Repository name or owner/repo

## Idempotency
Idempotent in effect (not in result). Re-running this command on an
already-open issue is a no-op server-side and still returns success.
The Gitee API treats "reopen an open issue" as a no-op rather than
an error.

## Atomicity
Atomic. Either the issue transitions to open (with closed_at cleared),
or no change happens. There is no partial-success state.

## Side effects
- The issue's state is set to "open" and closed_at is cleared
- Email notification sent to watchers
- Webhooks subscribed to "Issue" events fire (state: reopened)
- The issue reappears in the default "open" issue list
- The action is REVERSIBLE: ` + "`" + `gitee issue close {N}` + "`" + `
  closes the issue again and resets closed_at

## Post-conditions (on success)
- The issue's state is "open"
- Stdout: "Issue #N reopened"
- The issue's number is unchanged
- closed_at is cleared
- All previous comments and history are preserved

## Common errors
- 401 Unauthorized:  token expired or invalid    →  gitee auth login
- 403 Forbidden:     token lacks write scope     →  check token permissions
- 404 Not Found:     repo or issue doesn't exist →  verify owner/repo + number
` + projectOrEnterprise404Note + `

## Race condition warning ⚠️
The shared mutable state is the issue's ` + "`" + `state` + "`" + ` field on the
server. Between your last ` + "`" + `gitee issue view {N}` + "`" + ` and this reopen
call, another agent or user may have reopened the issue already —
or closed it again. Reopening an already-open issue is a silent
no-op server-side (see Idempotency), so a successful exit does NOT
prove your call caused the transition. If a follow-up action depends
on the issue being open (e.g. "reopen, then assign"), verify with
` + "`" + `gitee issue view {N}` + "`" + ` right after reopening. Retrying a failed
reopen is safe. In multi-agent triage, serialize close/reopen
transitions per issue number to avoid ping-pong.

## Reversibility
This command is REVERSIBLE. To close the issue again, run
` + "`" + `gitee issue close {N} --repo owner/repo` + "`" + `. Comments and history
are preserved across close/reopen cycles — the audit trail is not
erased.

## Next steps (orchestration hints)
After reopening an issue, an agent typically wants to:
1. Verify the state: ` + "`" + `gitee issue view {N}` + "`" + ` (state should be "open")
2. Comment to record why it was reopened:
   ` + "`" + `gitee issue comment {N} --body 'Reopening: regression in v0.10.1'` + "`" + `
3. If the issue is back in scope, add a fix label and assignee:
   ` + "`" + `gitee issue edit {N} --labels bug --assignees alice` + "`" + `
4. If you reopened by mistake, immediately: ` + "`" + `gitee issue close {N}` + "`" + `
5. For the inverse (close a still-open issue): ` + "`" + `gitee issue close` + "`" + `

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Implementation notes (Gitee v5 quirk)
The wire shape of this command is non-obvious. Gitee v5's issue update
endpoint is shaped DIFFERENTLY from the read endpoint:

  PATCH /repos/{owner}/issues/{number}                <-- no {repo} in path
  Content-Type: application/x-www-form-urlencoded
  repo=<repo>&state=open

The READ path is ` + "`" + `/repos/{owner}/{repo}/issues/{number}` + "`" + ` (GitHub-style),
but the WRITE path drops {repo} from the URL and instead accepts ` + "`" + `repo` + "`" + `
as a formData parameter. All update fields (including ` + "`" + `state` + "`" + `) are
formData, NOT JSON body.

If you ever see ` + "`" + `HTTP 400 "state does not have a valid value"` + "`" + ` or
` + "`" + `HTTP 404 "project or enterprise"` + "`" + ` after this command, the most
likely cause is a regression that re-introduced the JSON body / wrong
path — re-check ` + "`" + `internal/api/client.go` + "`" + ` (PatchForm) and
` + "`" + `internal/api/endpoints.go` + "`" + ` (IssuesEdit) together.

The same wiring is used by ` + "`" + `gitee issue close` + "`" + ` and ` + "`" + `gitee issue edit` + "`" + `;
all three share the v0.12.6 fix.

## Example
  # Reopen a prematurely-closed issue
  gitee issue reopen 42 --repo owner/repo

  # Reopen and immediately re-assign
  gitee issue reopen 42 --repo owner/repo
  gitee issue edit 42 --repo owner/repo --assignees alice --labels "bug,regression"

## See also
- ` + "`" + `gitee issue view N` + "`" + `       — read full details for one issue
- ` + "`" + `gitee issue list` + "`" + `        — discover issue numbers
- ` + "`" + `gitee issue close N` + "`" + `    — inverse of this command
- ` + "`" + `gitee issue edit N` + "`" + `     — change title/body/labels
- ` + "`" + `gitee issue comment N` + "`" + `  — add a comment (recommended after reopen)
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands with agent help
`

// issueCommentAgentHelp is the agent-targeted help for `gitee issue comment`.
// It is intentionally long, structured, and includes machine-relevant
// details (idempotency, side effects, orchestration hints) that the
// default --help omits because they're noise for humans.
//
// Maintainer note: keep this in sync with the actual command behavior.
// When flags change, update the relevant section here.
const issueCommentAgentHelp = `# gitee issue comment — Agent Help

## Purpose
Add a comment to an existing Gitee issue. Use this for triage notes,
status updates, audit trails, or to record automated decisions (e.g.
a CI agent noting "failing on master, see run #1234").

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Repository exists: --repo (or --owner + --repo) points to an accessible repo
- Write access: token must have write permission on the target repo
  (you need permission to comment, not just read)
- Issue exists: the issue number must be valid in the target repo
- --body is non-empty (enforced by Cobra MarkFlagRequired)

## Required flags
  --body         Comment body, Markdown supported (required, non-empty)

## Required arguments
  <number>  Issue number (integer, e.g. 42). Must be a positional arg
            after the verb. Use ` + "`" + `gitee issue list` + "`" + ` to discover numbers.

## Optional flags
  --owner        Repository owner
  --repo / -R    Repository name or owner/repo

## Idempotency
NOT idempotent. Each invocation creates a NEW comment with a fresh
ID. Re-running this command creates a duplicate comment. To update
an existing comment, use the Gitee web UI (gitee-cli does not yet
support ` + "`" + `gitee issue comment edit` + "`" + `).

## Atomicity
Atomic. Either the comment is created and an ID is returned, or no
change happens. There is no partial-success state.

## Side effects
- Email notification sent to all issue subscribers (watchers, the
  author, assignees, and anyone @-mentioned in the body)
- Webhooks subscribed to "Issue comment" events fire
- The comment appears immediately in the issue's comment list (web UI)
- If the body contains @username, that user gets a direct mention ping
- The new comment is in chronological order at the bottom of the
  existing comment list

## Post-conditions (on success)
- A new comment is appended to the issue
- Stdout: "Comment added to issue #N"
- The issue itself is unchanged (title, state, labels, etc. preserved)
- The new comment does NOT have any reactions by default

## Common errors
- 401 Unauthorized:  token expired or invalid    →  gitee auth login
- 403 Forbidden:     token lacks write scope     →  check token permissions
- 404 Not Found:     repo or issue doesn't exist →  verify owner/repo + number
- 422 Unprocessable: --body is empty             →  provide non-empty body

## Race condition warning ⚠️
If you are about to comment on an issue as part of a multi-step
orchestration, the issue state may have changed between your last
` + "`" + `gitee issue view` + "`" + ` and this comment call. If your action depends
on the state (e.g. "only comment if open"), re-check with
` + "`" + `gitee issue view {N}` + "`" + ` immediately before this call.

## Next steps (orchestration hints)
After adding a comment, an agent typically wants to:
1. Verify the comment landed: ` + "`" + `gitee issue comments {N}` + "`" + `
2. If the comment resolved a discussion, close the issue:
   ` + "`" + `gitee issue close {N}` + "`" + `
3. If the issue is still actionable, leave it open and add a label
   for tracking: ` + "`" + `gitee issue edit {N} --labels '...'` + "`" + `
4. To add a follow-up comment later, run this command again
   (each call appends a new comment — there is no edit-in-place)
5. If you need to mention a specific user, include @username in body

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Add a triage note
  gitee issue comment 42 --repo owner/repo --body "Reproduced on master (commit abc123). Investigating."

  # Comment then close (common pattern)
  gitee issue comment 42 --repo owner/repo --body "Closing as fixed in PR #99"
  gitee issue close 42 --repo owner/repo

  # Mention a teammate
  gitee issue comment 42 --repo owner/repo --body "@alice can you take a look at the stack trace?"

## See also
- ` + "`" + `gitee issue view N` + "`" + `       — read full details for one issue
- ` + "`" + `gitee issue comments N` + "`" + ` — list all comments on an issue
- ` + "`" + `gitee issue list` + "`" + `        — discover issue numbers
- ` + "`" + `gitee issue close N` + "`" + `    — close the issue (often follows a comment)
- ` + "`" + `gitee issue edit N` + "`" + `     — change title/body/labels (does NOT comment)
- ` + "`" + `gitee --agent-help` + "`" + `     — overview of all commands with agent help
`
