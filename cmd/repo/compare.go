package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Per the P1-8 spec, `gitee repo compare` lives in its own file
// (`cmd/repo/compare.go`) because it is a single verb (not a subcommand
// group with multiple inner verbs) — keeping it separate from commits.go
// keeps each file focused on one user-facing command.
//
// Endpoint: GET /repos/{owner}/{repo}/compare/{base}...{head}
//   - base / head are refs: SHA, short SHA, branch, or tag
//   - response is a CompareResult with status, ahead/behind counts,
//     the commit list, and the aggregated file diff

// compareFlags holds the per-invocation state. The base and head refs
// are positional arguments; --repo is a shared flag with the rest of
// the repo command family.
var compareFlags struct {
	repo string
}

// compareCmd is a single verb, not a group, so it has its own --help
// and no umbrella subcommands.
var compareCmd = &cobra.Command{
	Use:   "compare <base>...<head>",
	Short: "Compare two refs (branches, tags, or SHAs) on a repository",
	Long: `Compare two refs on a Gitee repository and show the diff.

The positional arguments follow the Gitee v5 convention:

  gitee repo compare --repo owner/repo <base>...<head>

` + "`" + `<base>` + "`" + ` and ` + "`" + `<head>` + "`" + ` may each be a full 40-char SHA, a short SHA prefix,
a branch name, or a tag name. The three dots between them are literal
— pass them as a single string ` + "`" + `<base>...<head>` + "`" + ` so the CLI splits
them for you.`,
	Example: `  gitee repo compare --repo owner/repo main...feature/x
  gitee repo compare --repo owner/repo v1.0.0...v2.0.0
  gitee repo compare --repo owner/repo a1b2c3d...main`,
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoCompareAgentHelp)
		}
		return nil
	},
	RunE: runCompare,
}

func init() {
	f := compareCmd.Flags()
	f.StringVarP(&compareFlags.repo, "repo", "R", "", "Repository (owner/repo)")

	compareCmd.AddCommand() // no-op: compareCmd has no subcommands
}

func runCompare(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("compare requires a single <base>...<head> argument (e.g. main...feature/x)")
	}

	// The CLI accepts "<base>...<head>" as one positional argument so the
	// user does not have to quote-escape the dots. Split on the literal
	// three-dot separator; if it is missing, fall back to a two-argument
	// invocation (<base> <head>) for symmetry with the other list verbs.
	base, head, err := splitCompareArgs(args)
	if err != nil {
		return err
	}

	owner, repoName, err := resolveRepoFromFlags(compareFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.ReposCompare(owner, repoName, base, head)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var result types.CompareResult
	if err := client.Get(ctx, path, &result); err != nil {
		return fmt.Errorf("comparing %s...%s: %w", base, head, err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	printCompareSummary(owner, repoName, base, head, &result)
	if len(result.Commits) > 0 {
		printCompareCommits(result.Commits)
	}
	if len(result.Files) > 0 {
		printCompareFiles(result.Files)
	}
	return nil
}

// splitCompareArgs accepts the positional arguments as either:
//
//   - one argument "<base>...<head>" (the documented form), or
//   - two arguments "<base>" "<head>" (convenience for users who
//     prefer the explicit two-token form)
//
// and returns the resolved base / head strings. An empty result is
// rejected with a clear error so the user gets actionable feedback
// instead of an opaque 404 from Gitee.
func splitCompareArgs(args []string) (string, string, error) {
	if len(args) == 1 {
		raw := args[0]
		// Split on the literal "..." separator. We deliberately do not
		// accept ".." or ".." with extra dots because that would
		// collide with filesystem path syntax; users with branch names
		// containing dots can use the two-argument form.
		const sep = "..."
		i := strings.Index(raw, sep)
		if i < 0 {
			return "", "", fmt.Errorf("compare requires <base>...<head> as a single argument (got %q without the ... separator); pass two refs separately as <base> <head> instead", raw)
		}
		base := strings.TrimSpace(raw[:i])
		head := strings.TrimSpace(raw[i+len(sep):])
		if base == "" || head == "" {
			return "", "", fmt.Errorf("compare requires both <base> and <head>; got base=%q head=%q", base, head)
		}
		return base, head, nil
	}
	if len(args) >= 2 {
		base := strings.TrimSpace(args[0])
		head := strings.TrimSpace(args[1])
		if base == "" || head == "" {
			return "", "", fmt.Errorf("compare requires both <base> and <head>; got base=%q head=%q", base, head)
		}
		return base, head, nil
	}
	return "", "", fmt.Errorf("compare requires a <base>...<head> argument")
}

// printCompareSummary renders the top-level compare metadata to stdout:
// status, ahead/behind counts, total commits, and a couple of useful
// links when present. Mirrors the Gitee web UI's compare header.
func printCompareSummary(owner, repoName, base, head string, r *types.CompareResult) {
	status := strings.ToUpper(r.Status)
	if status == "" {
		status = "UNKNOWN"
	}
	fmt.Printf("Compare %s/%s: %s...%s\n", owner, repoName, base, head)
	fmt.Printf("  Status:       %s\n", status)
	fmt.Printf("  Ahead by:     %d commit(s)\n", r.AheadBy)
	fmt.Printf("  Behind by:    %d commit(s)\n", r.BehindBy)
	fmt.Printf("  Total commits: %d\n", r.TotalCommits)
	if r.HTMLURL != "" {
		fmt.Printf("  URL:          %s\n", color.CyanString(r.HTMLURL))
	}
}

// printCompareCommits renders the commit list inside the compare result.
// Shape matches the `commits list` table so agents and humans can move
// between the two commands without relearning the layout.
func printCompareCommits(commits []types.CompareCommit) {
	fmt.Println("  Commits:")
	w := output.NewTableWriter()
	fmt.Fprintln(w, "    SHA\tAUTHOR\tDATE\tMESSAGE")
	for _, cm := range commits {
		short := shortSHA(cm.SHA)
		author := "-"
		date := ""
		msg := ""
		if cm.Commit != nil {
			if cm.Commit.Author.Username != "" {
				author = cm.Commit.Author.Username
			} else if cm.Commit.Author.Name != "" {
				author = cm.Commit.Author.Name
			}
			if len(cm.Commit.Author.Date) >= 10 {
				date = cm.Commit.Author.Date[:10]
			}
			msg = firstLine(cm.Commit.Message)
		}
		if cm.Author != nil && cm.Author.Login != "" {
			author = cm.Author.Login
		}
		fmt.Fprintf(w, "    %s\t%s\t%s\t%s\n", short, author, date, msg)
	}
	_ = w.Flush()
}

// printCompareFiles renders the aggregated file diff (across all
// commits in the compare window). Same shape as the per-commit files
// table in `commits view`.
func printCompareFiles(files []types.CommitFile) {
	fmt.Println("  Files:")
	w := output.NewTableWriter()
	fmt.Fprintln(w, "    STATUS\tADD\tDEL\tFILE")
	for _, f := range files {
		status := strings.ToUpper(f.Status)
		if status == "" {
			status = "MODIFIED"
		}
		name := f.Filename
		if f.Previous != "" && f.Previous != f.Filename {
			name = f.Previous + " → " + f.Filename
		}
		fmt.Fprintf(w, "    %s\t+%d\t-%d\t%s\n", status, f.Additions, f.Deletions, name)
	}
	_ = w.Flush()
}

const repoCompareAgentHelp = `# gitee repo compare — Agent Help

## Purpose
Diff two refs on a Gitee repository and report the resulting commit
list and file changes. The refs may be branches, tags, full 40-char
SHAs, or short SHA prefixes — Gitee resolves them server-side.

The output mirrors the Gitee web UI's compare view: a top-line status
(diverged / ahead / behind / identical), an ahead-by / behind-by count,
the list of commits unique to ` + "`" + `head` + "`" + `, and the aggregated per-file diff.

## Pre-conditions
- Authenticated with read access to the repo
- --repo / -R is set
- <base> and <head> resolve to valid refs (Gitee returns 422 on
  ambiguous SHAs and 404 on missing refs)

## Required arguments
  <base>...<head>   Two refs separated by literal three dots.
                    Branches with dots in their names can be passed
                    as two separate positional arguments instead:
                      gitee repo compare --repo owner/repo <base> <head>

## Optional flags
  --repo / -R    Repository in owner/repo form.
  --output / -o  (global) json | table (default table).

## Idempotency
YES. Read-only; calling repeatedly yields the same result (modulo
upstream commits between calls).

## Atomicity
Atomic per request. One API call returns the full compare result.

## Side effects
None.

## Post-conditions (on success)
- Stdout contains the compare summary + commit list + per-file table
  (default) or a single CompareResult JSON object (-o json)

## Output (stdout, table mode)
  Compare owner/repo: main...feature/x
    Status:        DIVERGED
    Ahead by:      3 commit(s)
    Behind by:     0 commit(s)
    Total commits: 3
    URL:           https://gitee.com/owner/repo/compare/main...feature/x
    Commits:
      SHA       AUTHOR   DATE        MESSAGE
      a1b2c3d   alice    2026-06-01  Add pagination
      b2c3d4e   alice    2026-06-02  Fix tests
      c3d4e5f   alice    2026-06-03  Update docs
    Files:
      STATUS   ADD  DEL  FILE
      ADDED    +50  -0   src/pagination.go
      MODIFIED +3   -1   README.md

## Output (stdout, json mode)
  {
    "status": "diverged",
    "ahead_by": 3,
    "behind_by": 0,
    "total_commits": 3,
    "base_commit":      {"sha":"...","commit":{...}, ...},
    "merge_base_commit":{"sha":"...","commit":{...}, ...},
    "commits": [{"sha":"...","commit":{...}, ...}, ...],
    "files":   [{"filename":"...","additions":50,"deletions":0,"status":"added"}, ...],
    "html_url":"https://gitee.com/owner/repo/compare/main...feature/x"
  }

## Common errors
- 401 Unauthorized:   token missing or expired
- 403 Forbidden:      repo is private and the token cannot read it
- 404 Not Found:      repo, <base>, or <head> does not resolve
- 422 Unprocessable:  <base> / <head> is ambiguous (e.g. short SHA
                     matches multiple commits)

## Race condition warning ⚠️
The shared mutable state is the pair of refs being compared. Branch
names and tags resolve server-side AT REQUEST TIME, so the result is a
snapshot: a push to <head> (or a force-push to <base>) landing right
after this call changes ahead_by/behind_by and the file diff, while
your output still shows the old window. Classic failure: you see
"ahead by 3, behind by 0", decide the branch is merge-ready, and in
the gap someone pushes a breaking commit to <head> — your ` + "`" + `gitee pr
create` + "`" + ` / merge then carries code you never reviewed. Mitigation:
for decisions that must be exact, resolve both refs to full SHAs first
(` + "`" + `gitee repo commits view --repo owner/repo <ref>` + "`" + `) and compare
SHA...SHA — immutable inputs give a stable answer; and re-run compare
immediately before acting if the refs are branches. Retrying THIS
command is always safe (read-only).

## Next steps (orchestration hints)
After comparing, an agent typically wants to:
1. Open the Gitee web compare view for the same refs:
   ` + "`" + `gitee repo browse --repo owner/repo/compare/main...feature/x` + "`" + `
2. View a specific commit in detail:
   ` + "`" + `gitee repo commits view --repo owner/repo <sha>` + "`" + `
3. Open a PR if the diff should be merged:
   ` + "`" + `gitee pr create --repo owner/repo --base main --head feature/x` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour authenticated. One request per call.

## Example
  # Compare a feature branch against main
  gitee repo compare --repo myorg/myrepo main...feature/x

  # Compare two release tags
  gitee repo compare --repo myorg/myrepo v1.0.0...v2.0.0

  # Compare a SHA against main
  gitee repo compare --repo myorg/myrepo a1b2c3d...main

  # JSON for parsing
  gitee repo compare --repo myorg/myrepo main...feature/x -o json \\
    | jq '.commits | length'

  # Two-argument form for branches containing dots
  gitee repo compare --repo myorg/myrepo release/1.0 release/2.0

## See also
- ` + "`" + `gitee repo commits list` + "`" + `  — enumerate commits on a ref
- ` + "`" + `gitee repo commits view` + "`" + `  — detail of one commit
- ` + "`" + `gitee pr diff` + "`" + `          — diff via a PR number
- ` + "`" + `gitee repo browse` + "`" + `      — open the repo in a browser
`
