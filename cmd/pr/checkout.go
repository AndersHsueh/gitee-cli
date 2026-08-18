package pr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var checkoutFlags struct {
	repo    string
	branch  string
	remote  string
	force   bool
	noFetch bool
}

var checkoutCmd = &cobra.Command{
	Use:   "checkout <number>",
	Short: "Check out a pull request's head ref locally",
	Long: `Fetch the pull request's head branch into a local working copy.

This runs:
  git fetch <remote> pull/<number>/head:<local-branch>
  git checkout <local-branch>

The local branch name defaults to "pr-<number>" but can be overridden
with --branch. The remote defaults to "origin" but can be overridden
with --remote. The PR's source repo is auto-detected from the Gitee
API response.`,
	Example: `  gitee pr checkout 42 --repo owner/repo
  gitee pr checkout 42 -R owner/repo --branch feature/foo
  gitee pr checkout 42 -R owner/repo --remote upstream`,
	Aliases: []string{"co"},
	// PreRunE fires BEFORE ValidateRequiredFlags so --agent-help works.
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(prCheckoutAgentHelp)
		}
		return nil
	},
	RunE: runCheckout,
}

func init() {
	f := checkoutCmd.Flags()
	f.StringVarP(&checkoutFlags.repo, "repo", "R", "", "Repository (owner/repo) where the PR lives")
	f.StringVarP(&checkoutFlags.branch, "branch", "b", "", "Local branch name (default: pr-<number>)")
	f.StringVar(&checkoutFlags.remote, "remote", "origin", "Git remote to fetch from")
	f.BoolVar(&checkoutFlags.force, "force", false, "Overwrite existing local branch if it exists")
	f.BoolVar(&checkoutFlags.noFetch, "no-fetch", false, "Skip the git fetch (assume refs/pull/N/head is already local)")

	PRCmd.AddCommand(checkoutCmd)
}

func runCheckout(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	if len(args) < 1 {
		return fmt.Errorf("PR number is required (pass it as the first argument)")
	}
	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: must be an integer", args[0])
	}

	owner, repoName, err := resolvePRRepo("", checkoutFlags.repo)
	if err != nil {
		return err
	}

	// Default local branch name is "pr-<number>".
	localBranch := checkoutFlags.branch
	if localBranch == "" {
		localBranch = fmt.Sprintf("pr-%d", prNum)
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	// We need the PR's head ref (and source repo, for cross-fork PRs).
	// Always fetch metadata, even with --no-fetch, because the refspec
	// `pull/<N>/head` is computed from the PR number — we don't need
	// source branch info unless the user wants to know.
	// However we DO need to know if the local branch already exists.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	prFetchErr := client.Get(ctx, api.PRsGet(owner, repoName, prNum), &pr)
	if prFetchErr != nil && !checkoutFlags.noFetch {
		// With --no-fetch the user has explicitly opted out of API-dependent
		// behavior; the GET failure should not gate a local-only operation.
		return fmt.Errorf("getting PR #%d: %w", prNum, prFetchErr)
	}

	// Check whether the local branch already exists. With --force we will
	// delete it below, so this guard is skipped.
	if !checkoutFlags.force {
		if exists, err := gitBranchExists(localBranch); err != nil {
			return fmt.Errorf("checking local branch: %w", err)
		} else if exists {
			return fmt.Errorf("local branch %q already exists; use --force to overwrite or --branch to choose a new name", localBranch)
		}
	}

	// --force: delete the local branch (if it exists) so the subsequent
	// fetch + checkout can overwrite it cleanly. Best-effort: ignore "branch
	// doesn't exist" errors, which `git branch -D` returns as exit 1.
	if checkoutFlags.force {
		_ = gitRunSilent("branch", "-D", localBranch)
	}

	if !checkoutFlags.noFetch {
		// Refspec: pull/<N>/head:<local-branch>
		// This works for both same-repo and cross-fork PRs.
		refspec := fmt.Sprintf("pull/%d/head:%s", prNum, localBranch)
		if err := gitRun("fetch", checkoutFlags.remote, refspec); err != nil {
			return fmt.Errorf("git fetch failed: %w", err)
		}
	}

	if err := gitRun("checkout", localBranch); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s Checked out %s (PR #%d: %s)\n", green("✓"), bold(localBranch), prNum, pr.Title)
	if pr.HTMLURL != "" {
		fmt.Printf("  URL:  %s\n", cyan(pr.HTMLURL))
	}
	if pr.Head != nil && pr.Head.Ref != "" {
		fmt.Printf("  Head: %s (from %s)\n", pr.Head.Ref, pr.Head.Repo.FullName)
	}
	return nil
}

// gitBranchExists returns true if a local branch named name exists.
func gitBranchExists(name string) (bool, error) {
	out, err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+name).CombinedOutput()
	if err == nil {
		return true, nil
	}
	// exit 1 means ref doesn't exist; anything else is an error.
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
}

// gitRun runs `git <subcommand> <args...>` and streams stdout/stderr to
// the parent process so users see git's output directly.
func gitRun(subcommand string, args ...string) error {
	fullArgs := append([]string{subcommand}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// gitRunSilent runs a git command and discards all output. Used for
// best-effort cleanup like "git branch -D".
func gitRunSilent(args ...string) error {
	return exec.Command("git", args...).Run()
}

const prCheckoutAgentHelp = `# gitee pr checkout — Agent Help

## Purpose
Fetch and check out a pull request's head ref as a local git branch.
Equivalent to ` + "`" + `gh pr checkout` + "`" + `. The head ref is fetched as
` + "`" + `refs/pull/<N>/head` + "`" + ` from the configured remote and bound to a
local branch (default: ` + "`" + `pr-<N>` + "`" + `).

## Pre-conditions
- Authenticated (only to look up the PR title; the fetch itself uses
  the git remote, not the API token)
- A git working copy on disk
- The configured --remote exists and has the PR

## Required arguments
  <number>     PR number (positional, required)

## Optional flags
  --repo / -R    owner/repo where the PR lives
  --branch / -b  Local branch name (default: pr-<number>)
  --remote       Git remote to fetch from (default: origin)
  --force        Overwrite an existing local branch with the same name
  --no-fetch     Skip the git fetch (assume refs/pull/N/head already local)

## Aliases
  gitee co <number>   — same as ` + "`" + `gitee pr checkout` + "`" + `

## Idempotency
YES (without --force). Re-running after the first checkout is a no-op
(existing local branch is detected and the command exits cleanly with
an error suggesting --force).

With --force, the local branch is forcibly reset to the current
remote head. Repeatable.

## Atomicity
NOT atomic. If ` + "`" + `git fetch` + "`" + ` succeeds but ` + "`" + `git checkout` + "`" + ` fails (e.g.
uncommitted local changes blocking the switch), the fetched refs
remain in your local repo. Re-run the command after resolving the
conflict to complete the checkout.

## Side effects
- Creates a local git branch (e.g. pr-42)
- May modify FETCH_HEAD and the remote-tracking namespace
- Does NOT touch the Gitee API beyond the read-only PR lookup

## Post-conditions (on success)
- You are on a local branch whose tip matches refs/pull/<N>/head
- Stdout reports the branch name and the PR's title

## Output (stdout)
  ✓ Checked out pr-42 (PR #42: Add dark mode toggle)
    URL:  https://gitee.com/owner/repo/pulls/42
    Head: feature/dark-mode (from fork-owner/fork-repo)

## Common errors
- 404 Not Found: PR doesn't exist
- "local branch already exists": use --force or --branch
- "git fetch failed": wrong --remote, or the PR is from a fork and
  the remote doesn't have it
- "git not in PATH": install git

## OAuth scope required
projects (read) — only used to fetch the PR title for the success line

## Rate limit
1000 req/hour for the API call; the git fetch is rate-limited by your
git remote, not the Gitee API.

## Race condition warning ⚠️
The fetched head is a SNAPSHOT at fetch time. If the PR is updated
after you check it out, your local branch will be stale. To refresh:
run ` + "`" + `gitee pr checkout <N> --force` + "`" + `.

## Next steps (orchestration hints)
1. After checkout, you are on the PR's branch locally.
2. Run your tests: ` + "`" + `go test ./...` + "`" + `, ` + "`" + `npm test` + "`" + `, etc.
3. Make changes and commit.
4. Push back to the PR: ` + "`" + `git push <remote> HEAD` + "`" + `
5. Review/approve via ` + "`" + `gitee pr review <N>` + "`" + `

## Example (closing the loop)
  # 1. Check out the PR to investigate a bug
  gitee pr checkout 42 -R owner/repo
  go test ./...

  # 2. Apply a fix on the local branch
  vim internal/foo.go
  git commit -am "fix: handle nil pointer"

  # 3. Push back so the PR is updated
  git push origin HEAD

## See also
- ` + "`" + `gitee pr view` + "`" + `         — see PR metadata without checking out
- ` + "`" + `gitee pr diff` + "`" + `         — see what files changed
- ` + "`" + `gitee pr review` + "`" + `       — approve / request changes
- ` + "`" + `gitee co` + "`" + `             — alias of this command
`
