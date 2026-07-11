package pr

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

var listFlags struct {
	owner string
	repo  string
	state string
	head  string
	base  string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Example: `  gitee pr list --repo owner/repo
  gitee pr list --repo owner/repo --state closed
  gitee pr list --repo owner/repo --head user:feature
  gitee pr list --repo owner/repo --paginate             # fetch ALL PRs
  gitee pr list --repo owner/repo --paginate --limit 0   # same, explicit`,
	// PreRunE fires BEFORE ValidateRequiredFlags, so an agent can request
	// --agent-help even when --repo is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(prListAgentHelp)
		}
		return nil
	},
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.StringVar(&listFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&listFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVar(&listFlags.state, "state", "open", "Filter by state: open|closed|merged|all")
	f.StringVar(&listFlags.head, "head", "", "Filter by head branch (user:ref)")
	f.StringVar(&listFlags.base, "base", "", "Filter by base branch")
	// --limit / --per-page / --paginate are global persistent flags.
	// See cmd/root.go for their definitions. --limit is a TOTAL cap
	// across pages when --paginate is set (was a per-page size, which
	// was a bug — see v0.4.1 changelog).
}

func runList(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolvePRRepo(listFlags.owner, listFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	// Build base query (without page/per_page — FetchAllPages handles those)
	qv := url.Values{}
	qv.Set("state", listFlags.state)
	if listFlags.head != "" {
		qv.Set("head", listFlags.head)
	}
	if listFlags.base != "" {
		qv.Set("base", listFlags.base)
	}
	basePath := api.PRsList(owner, repoName) + "?" + qv.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// FetchListOrAll handles perPage clamp (1-100), --paginate dispatch, and
	// treats a 404 as an empty result (unified with issue list / commits).
	prs, err := api.FetchListOrAll[types.PullRequest](ctx, client, basePath,
		cmd.GlobalPaginate, cmd.GlobalPerPage, cmd.GlobalLimit, cmd.GlobalPage,
		"listing PRs")
	if err != nil {
		return err
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(prs, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(prs) == 0 {
		fmt.Println("No pull requests found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tTITLE\tSTATE\tHEAD→BASE\tAUTHOR\tUPDATED")
	for _, pr := range prs {
		headToBase := ""
		if pr.Head != nil && pr.Base != nil {
			headToBase = fmt.Sprintf("%s→%s", pr.Head.Ref, pr.Base.Ref)
		}
		state := pr.State
		if pr.Merged {
			state = "merged"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			pr.Number, pr.Title, state, headToBase, pr.User.Login, utils.TruncateDate(pr.UpdatedAt))
	}
	return w.Flush()
}

func resolvePRRepo(owner, repo string) (string, string, error) {
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	return utils.ResolveRepo(owner, repo, nil)
}

// prListAgentHelp is the agent-targeted help for `gitee pr list`.
// Mirrors the structure of issueCreateAgentHelp so an agent can rely
// on every `list` command exposing the same sections.
const prListAgentHelp = `# gitee pr list — Agent Help

## Purpose
List pull requests in a Gitee repository, with optional state / head /
base filters. Use this to enumerate PRs for review, audit, or batch
workflows. For a single PR's full metadata, follow up with
` + "`" + `gitee pr view <N>` + "`" + `.

## Pre-conditions
- Authenticated: token in config, env, or --token (verify with ` + "`" + `gitee auth status` + "`" + `)
- --repo (or --owner) points to an existing, accessible repository
- --state, if given, is one of: open, closed, merged, all
- --head format: ` + "`" + `user:branch` + "`" + ` (e.g. ` + "`" + `alice:feature-x` + "`" + `)
- --base is a branch name on the target repo

## Required flags
  --repo / -R     owner/repo format (or pass --owner + --repo, or rely on global --repo / git remote)

## Optional flags
  --owner            Repository owner (overrides the owner part of --repo)
  --state            open | closed | merged | all  (default: open)
  --head             Filter by head branch as ` + "`" + `user:ref` + "`" + `
  --base             Filter by base branch name
  --paginate         Walk all pages of results (use with --limit)
  --limit N          Total cap across pages when --paginate is set; 0 = unlimited
  --page N           Page number for single-page mode (default 1)
  --per-page N       Items per page (default 30, max 100)
  --output json      Emit a JSON array of PR objects instead of a table

## Idempotency
YES. Read-only; calling it twice yields the same list (modulo
concurrent PR activity). Safe to retry on transient errors.

## Atomicity
Single API call (or sequence of paged calls when --paginate is set).
Each call either succeeds and returns the current snapshot or fails —
no partial-success state on the client.

## Side effects
- None on the Gitee repo. No notifications, no webhooks fire.
- The token's read access is recorded in Gitee's access log.
- Rate-limit counter advances by 1 (or N pages when --paginate is set).

## Post-conditions (on success)
- Stdout contains a table (default) or a JSON array (with --output json)
  of PR objects: number, title, state, head→base, author, updated_at
- Exit status 0
- No file or network state mutated

## Output (stdout, table mode)
  #      TITLE                       STATE    HEAD→BASE          AUTHOR   UPDATED
  42     Add dark mode               open     alice:dark→main    alice    2026-06-28
  41     Fix off-by-one              merged   bob:fix→main      bob      2026-06-25
  40     Drop legacy auth            closed   carol:old→main    carol    2026-06-20

  When the repo has no matching PRs: ` + "`" + `No pull requests found` + "`" + `

## Output (stdout, json mode)
  [
    {
      "number": 42,
      "title": "Add dark mode",
      "state": "open",
      "head": { "ref": "dark", "...": "..." },
      "base": { "ref": "main", "...": "..." },
      "user":  { "login": "alice", "...": "..." },
      "updated_at": "2026-06-28T10:11:12Z",
      "...": "..."
    }
  ]

## Common errors
- 401 Unauthorized:  token missing or expired  →  ` + "`" + `gitee auth login` + "`" + `
- 403 Forbidden:     no read access to the repo  →  ask the owner to add you as a collaborator
- 404 Not Found:     repo does not exist (or you can't see it)  →  verify --repo spelling
- 422 Unprocessable: invalid --state value  →  use open|closed|merged|all
- Network timeouts:  bump ` + "`" + `--timeout` + "`" + ` (global flag) or retry

## Next steps (orchestration hints)
After enumerating PRs, an agent typically wants to:
1. Pick PR numbers from the table or JSON output
2. For full metadata on a single PR: ` + "`" + `gitee pr view <N>` + "`" + `
3. To check CI / review status: ` + "`" + `gitee pr checks <N>` + "`" + ` and ` + "`" + `gitee pr review <N>` + "`" + `
4. To check out a PR branch locally: ` + "`" + `gitee pr checkout <N>` + "`" + ` (or ` + "`" + `gitee co <N>` + "`" + `)
5. To merge after approvals: ` + "`" + `gitee pr merge <N>` + "`" + ` (only if all required checks pass)

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
With --paginate and --per-page 100, fetching 1000 PRs costs 10 calls.

## Example (agent-friendly)
  # 1. List every open PR as JSON
  gitee pr list --repo owner/repo --state open --output json > open-prs.json

  # 2. Extract numbers with jq
  jq -r '.[].number' open-prs.json

  # 3. View the first one
  N=$(jq -r '.[0].number' open-prs.json)
  gitee pr view "$N" --repo owner/repo

  # 4. Fetch the full set across all pages (capped at 500)
  gitee pr list --repo owner/repo --state all --paginate --limit 500

## See also
- ` + "`" + `gitee pr view <N>` + "`" + `        — full metadata for a single PR
- ` + "`" + `gitee pr checks <N>` + "`" + `      — CI status for a PR
- ` + "`" + `gitee pr checkout <N>` + "`" + `   — fetch the PR branch locally
- ` + "`" + `gitee pr merge <N>` + "`" + `      — merge a PR (write)
- ` + "`" + `gitee pr close <N>` + "`" + `      — close a PR (write, reversible via reopen)
- ` + "`" + `gitee --agent-help` + "`" + `      — overview of all commands with agent help
`
