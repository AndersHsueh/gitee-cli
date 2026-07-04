package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var commentsFlags struct {
	owner string
	repo  string
	limit int
}

var commentsCmd = &cobra.Command{
	Use:   "comments <number>",
	Short: "List comments on an issue",
	Example: `  gitee issue comments 42 --repo owner/repo
  gitee issue comments 42 -R owner/repo --limit 50
  gitee issue comments 42 -R owner/repo --output json`,
	Args: cmd.AgentHelpAwareArgs(cobra.ExactArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(issueCommentsAgentHelp)
		}
		return nil
	},
	RunE: runComments,
}

func init() {
	f := commentsCmd.Flags()
	f.StringVar(&commentsFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&commentsFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.IntVarP(&commentsFlags.limit, "limit", "L", 30, "Maximum number of comments to return")
}

func runComments(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveRepo(commentsFlags.owner, commentsFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.IssueCommentsList(owner, repoName, args[0])
	q := url.Values{}
	q.Set("per_page", fmt.Sprintf("%d", commentsFlags.limit))
	fullPath := path + "?" + q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var comments []types.IssueComment
	if err := client.Get(ctx, fullPath, &comments); err != nil {
		return fmt.Errorf("listing issue comments: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(comments, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(comments) == 0 {
		fmt.Printf("No comments on issue #%s in %s/%s\n", args[0], owner, repoName)
		return nil
	}

	fmt.Printf("Comments on issue #%s in %s/%s:\n\n", args[0], owner, repoName)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tAUTHOR\tCREATED\tBODY")
	for _, c := range comments {
		author := c.User.Login
		if author == "" {
			author = "(unknown)"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", c.ID, author, c.CreatedAt.Format("2006-01-02 15:04"), utils.Truncate(c.Body, 60))
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d comment(s)\n", len(comments))
	return nil
}

const issueCommentsAgentHelp = `# gitee issue comments — Agent Help

## Purpose
List every comment on a Gitee issue. Use this to read the conversation
history, audit who said what, or extract the latest comment for
chaining into another command (e.g. parsing the most recent status
update).

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `)
- Token has ` + "`" + `projects` + "`" + ` scope (read is sufficient)
- --repo resolves to an existing repo
- <number> is a valid issue number (use ` + "`" + `gitee issue list` + "`" + ` to discover)

## Required arguments
  <number>   Issue number (the #N displayed in ` + "`" + `gitee issue list` + "`" + `)

## Optional flags
  --repo / -R    owner/repo
  --limit / -L   Maximum number of comments (default 30, max 100)
  --output       "table" (default) or "json"

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET.

## Post-conditions (on success)
- Stdout contains a table of comments (ID, AUTHOR, CREATED, BODY)
- Repo state is unchanged

## Output (stdout, table mode)
  ID    AUTHOR     CREATED              BODY
  9001  octocat    2026-01-15 09:23     Looking into this now.
  9002  octocat    2026-01-16 14:05     Confirmed the bug, fix coming in #43.

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 403 Forbidden:    no read access
- 404 Not Found:    issue number doesn't exist
- 422 Unprocessable: --limit is out of range (Gitee max per_page is 100)

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  # Get the most recent comment body
  gitee issue comments 42 -R owner/repo --limit 1 --output json | jq -r '.[0].body'

## See also
- ` + "`" + `gitee issue view` + "`" + `   — issue body + metadata (without comments)
- ` + "`" + `gitee issue comment` + "`" + ` — add a new comment (singular)
- ` + "`" + `gitee pr comments` + "`" + `  — same for pull requests
`
