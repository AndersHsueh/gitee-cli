package pr

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var editFlags struct {
	owner string
	repo  string
	id    int64
	title string
	body  string
	state string
	draft bool
}

var editCmd = &cobra.Command{
	Use:   "edit <number>",
	Short: "Edit a pull request (title, body, state, or draft flag)",
	Example: `  # Rename a PR
  gitee pr edit 42 -R owner/repo --title "Better title"

  # Close a PR
  gitee pr edit 42 -R owner/repo --state closed

  # Reopen (set state to open)
  gitee pr edit 42 -R owner/repo --state open

  # Convert to draft
  gitee pr edit 42 -R owner/repo --draft

  # Mark ready (use 'gitee pr ready' for this — same effect)
  gitee pr edit 42 -R owner/repo --no-draft`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prEditAgentHelp)
		}
		return nil
	},
	RunE: runEdit,
}

func init() {
	f := editCmd.Flags()
	f.StringVar(&editFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&editFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.Int64VarP(&editFlags.id, "id", "i", 0, "PR number (alternative to positional)")
	f.StringVarP(&editFlags.title, "title", "t", "", "New title (empty = leave unchanged)")
	f.StringVarP(&editFlags.body, "body", "b", "", "New body (empty = leave unchanged)")
	f.StringVarP(&editFlags.state, "state", "s", "", "New state: 'open' or 'closed' (empty = leave unchanged)")
	f.BoolVar(&editFlags.draft, "draft", false, "Mark as draft (use --no-draft to mark ready)")
}

func runEdit(cobraCmd *cobra.Command, args []string) error {
	prNum, err := utils.ParseInt64Arg(editFlags.id, args, "PR")
	if err != nil {
		return err
	}
	if prNum <= 0 {
		return fmt.Errorf("PR number is required (positional or --id)")
	}

	// At least one updatable field must be explicitly set. We use cobra's
	// Flags().Changed() to distinguish "user passed the flag" from "user
	// didn't pass it" — the same pattern as gitee webhook edit.
	if !cobraCmd.Flags().Changed("title") &&
		!cobraCmd.Flags().Changed("body") &&
		!cobraCmd.Flags().Changed("state") &&
		!cobraCmd.Flags().Changed("draft") {
		return fmt.Errorf("at least one of --title / --body / --state / --draft is required")
	}

	owner, repoName, err := resolvePRRepo(editFlags.owner, editFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.PREditRequest{
		Title: utils.StringPtrIfChanged(cobraCmd, "title", editFlags.title),
		Body:  utils.StringPtrIfChanged(cobraCmd, "body", editFlags.body),
		State: utils.StringPtrIfChanged(cobraCmd, "state", editFlags.state),
		Draft: utils.BoolPtrIfChanged(cobraCmd, "draft", editFlags.draft),
	}
	// Validate --state if set
	if req.State != nil {
		s := *req.State
		if s != "open" && s != "closed" {
			return fmt.Errorf("--state must be 'open' or 'closed' (got %q)", s)
		}
	}

	path := api.PRsEdit(owner, repoName, int(prNum))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Patch(ctx, path, req, &pr); err != nil {
		return fmt.Errorf("editing PR: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("\n%s PR #%s updated\n", green("✓"), bold(fmt.Sprintf("%d", prNum)))
	fmt.Printf("  Title: %s\n", pr.Title)
	fmt.Printf("  State: %s\n", pr.State)
	if pr.Draft {
		fmt.Printf("  Draft: yes\n")
	}
	if pr.HTMLURL != "" {
		fmt.Printf("  URL:   %s\n", pr.HTMLURL)
	}
	fmt.Println()
	return nil
}

// stringPtrIfChanged and boolPtrIfChanged moved to internal/utils/utils.go
// (utils.StringPtrIfChanged, utils.BoolPtrIfChanged) to share with
// cmd/webhook/edit.go. See utils for the doc.

const prEditAgentHelp = `# gitee pr edit — Agent Help

## Purpose
Modify an existing pull request's title, body, state, or draft flag
in a single PATCH call. Use this instead of close+reopen when you
want to keep the PR id stable. For the common "draft → ready" flip,
` + "`" + `gitee pr ready` + "`" + ` is a one-line shortcut for ` + "`" + `--no-draft` + "`" + `.

## Pre-conditions
- Authenticated with write access to the target repo
- <number> is an existing PR id
- At least one of --title / --body / --state / --draft is supplied
  (Cobra + handler validation; running with none returns an error
  before any API call)

## Required arguments
  <number>   PR number

## Required flags (at least one)
  --title / -t    New title
  --body / -b     New body
  --state / -s    New state ("open" or "closed")
  --draft         Mark as draft (--no-draft to mark ready)

## Optional flags
  --repo / -R    owner/repo
  --id / -i      Alternative to positional

## Idempotency
YES (kind of). Re-running with the same flag values is a no-op on the
server. NOT a "create or update" — the PR must already exist.

## Atomicity
Atomic. The PATCH is one API call; either all changed fields are
updated or none are.

## Side effects
- Subscribers may receive a notification for the title/body change
  (Gitee suppresses notifications for identical updates)
- State change (open ↔ closed) fires the corresponding webhooks
- Draft flag change marks/unmarks the PR as ready for review (see
  ` + "`" + `gitee pr ready` + "`" + ` for the dedicated flow)

## Post-conditions (on success)
- The PR's title / body / state / draft reflect the new values
- The PR id is unchanged
- Stdout shows the new title + state + URL

## Output (stdout)
  ✓ PR #42 updated
    Title: Better title
    State: open
    URL:   https://gitee.com/owner/repo/pulls/42

## Common errors
- 401 Unauthorized: token missing or expired
- 403 Forbidden:    not a repo collaborator
- 404 Not Found:    PR number doesn't exist
- 422 Unprocessable: --state is something other than "open"/"closed",
  PR is in a state that prevents editing (e.g. merged)

## Next steps (orchestration hints)
1. Fetch the current PR: ` + "`" + `gitee pr view 42 -R owner/repo` + "`" + `
2. Apply changes: ` + "`" + `gitee pr edit 42 -R owner/repo --title "..." --state closed` + "`" + `
3. Verify: ` + "`" + `gitee pr view 42 -R owner/repo` + "`" + ` (check Title / State)

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour.

## Example (closing the loop)
  # 1. Close a stale PR (no other field changes)
  gitee pr edit 42 -R owner/repo --state closed

  # 2. Reopen with a refined title
  gitee pr edit 42 -R owner/repo --state open --title "[WIP] retry: 2nd attempt"

## See also
- ` + "`" + `gitee pr view` + "`" + `   — inspect current state before editing
- ` + "`" + `gitee pr close` + "`" + `  — shortcut for ` + "`" + `--state closed` + "`" + `
- ` + "`" + `gitee pr reopen` + "`" + ` — shortcut for ` + "`" + `--state open` + "`" + `
- ` + "`" + `gitee pr ready` + "`" + `  — shortcut for ` + "`" + `--no-draft` + "`" + `
`
