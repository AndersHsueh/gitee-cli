package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Per the P1-8 spec, the commits subcommand group lives entirely in this
// one file (`cmd/repo/commits.go`) and registers two verbs under the
// `gitee repo commits` umbrella:
//
//   list — GET /repos/{owner}/{repo}/commits
//   view — GET /repos/{owner}/{repo}/commits/{sha}
//
// Both share the --repo / -R flag pattern; list additionally accepts
// --sha/--path/--author/--since/--until to filter the response, view
// takes a positional <sha> argument (a full 40-char SHA, a short SHA
// prefix, a branch name, or a tag name — Gitee accepts all four).
//
// `gitee repo compare` lives in its own file (compare.go) per the spec.

// commitsFlags is shared across list/view so the user can invoke
// `gitee repo commits list --repo owner/repo --author alice` (Cobra binds
// by pointer). Per-verb fields (path, author, since, until) are read by
// the list RunE only when set; view ignores them.
var commitsFlags struct {
	repo   string
	sha    string
	path   string
	author string
	since  string
	until  string
}

// commitsCmd is the umbrella command. Running `gitee repo commits` with
// no subcommand shows help so users discover list/view.
var commitsCmd = &cobra.Command{
	Use:   "commits <subcommand>",
	Short: "List commits or view a single commit on a repository",
	Long: `Inspect commit history of a Gitee repository.

  gitee repo commits list            — list commits, optionally filtered
  gitee repo commits view <sha>      — view a single commit + its diff stat

Filtering on ` + "`" + `list` + "`" + ` follows Gitee v5 query params: --sha narrows the
branch/SHA to walk, --path restricts to commits touching a path,
--author filters by Gitee username, and --since / --until accept
ISO-8601 timestamps (e.g. ` + "`" + `2026-01-01T00:00:00Z` + "`" + `).`,
	Example: `  gitee repo commits list --repo owner/repo
  gitee repo commits list --repo owner/repo --author alice --since 2026-01-01
  gitee repo commits view --repo owner/repo a1b2c3d4
  gitee repo commits view --repo owner/repo main`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoCommitsAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		// No subcommand given — show help so users discover list/view.
		return c.Help()
	},
}

var commitsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List commits on a repository (default branch or --sha)",
	Example: `  gitee repo commits list --repo owner/repo
  gitee repo commits list -R owner/repo --author alice
  gitee repo commits list --repo owner/repo --path src/main.go --since 2026-01-01`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoCommitsListAgentHelp)
		}
		return nil
	},
	RunE: runCommitsList,
}

var commitsViewCmd = &cobra.Command{
	Use:   "view <sha>",
	Short: "View a single commit (SHA, short SHA, branch, or tag)",
	Example: `  gitee repo commits view --repo owner/repo a1b2c3d4
  gitee repo commits view --repo owner/repo main
  gitee repo commits view -R owner/repo v1.0.0`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoCommitsViewAgentHelp)
		}
		return nil
	},
	RunE: runCommitsView,
}

func init() {
	// list flags
	lf := commitsListCmd.Flags()
	lf.StringVarP(&commitsFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	lf.StringVar(&commitsFlags.sha, "sha", "", "Branch, tag, or SHA to start walking from (defaults to the default branch)")
	lf.StringVar(&commitsFlags.path, "path", "", "Only include commits touching this path (e.g. src/main.go)")
	lf.StringVar(&commitsFlags.author, "author", "", "Filter by author Gitee username")
	lf.StringVar(&commitsFlags.since, "since", "", "Only commits after this ISO-8601 timestamp")
	lf.StringVar(&commitsFlags.until, "until", "", "Only commits before this ISO-8601 timestamp")

	// view flags
	vf := commitsViewCmd.Flags()
	vf.StringVarP(&commitsFlags.repo, "repo", "R", "", "Repository (owner/repo)")

	commitsCmd.AddCommand(commitsListCmd)
	commitsCmd.AddCommand(commitsViewCmd)
}

// runCommitsList handles `gitee repo commits list`. It applies the
// filter flags as query params (Gitee ignores unknown ones, so we only
// forward what was actually set). Pagination is delegated to the global
// --paginate / --limit pair: when --paginate is set, FetchAllPages
// walks pages until --limit (or all pages when --limit 0).
func runCommitsList(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	owner, repoName, err := resolveRepoFromFlags(commitsFlags.repo)
	if err != nil {
		return err
	}

	qv := url.Values{}
	if commitsFlags.sha != "" {
		qv.Set("sha", commitsFlags.sha)
	}
	if commitsFlags.path != "" {
		qv.Set("path", commitsFlags.path)
	}
	if commitsFlags.author != "" {
		qv.Set("author", commitsFlags.author)
	}
	if commitsFlags.since != "" {
		qv.Set("since", commitsFlags.since)
	}
	if commitsFlags.until != "" {
		qv.Set("until", commitsFlags.until)
	}

	basePath := api.ReposCommitsList(owner, repoName)
	if encoded := qv.Encode(); encoded != "" {
		basePath = basePath + "?" + encoded
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	// FetchListOrAll handles perPage clamp (1-100), --paginate dispatch, and
	// treats a 404 as an empty result (unified with issue list / pr list).
	commits, err := api.FetchListOrAll[types.Commit](ctx, client, basePath,
		rt.Paginate, rt.PerPage, rt.Limit, rt.Page,
		"listing commits")
	if err != nil {
		return err
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(commits, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(commits) == 0 {
		fmt.Printf("No commits found in %s/%s\n", owner, repoName)
		return nil
	}

	w := output.NewTableWriter()
	fmt.Fprintln(w, "SHA\tAUTHOR\tDATE\tMESSAGE")
	for _, cm := range commits {
		cc := cm
		short := shortSHA(cc.SHA)
		author := authorName(&cc)
		date := commitDate(&cc)
		msg := firstLine(commitMessage(&cc))
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", short, author, date, msg)
	}
	return w.Flush()
}

// runCommitsView handles `gitee repo commits view <sha>`. The single-
// commit endpoint returns the same shape as list (Commit) plus a
// `stats` and `files` block. We surface the summary in plaintext and
// the per-file diff table.
func runCommitsView(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("commits view requires a <sha> argument (full SHA, short SHA, branch, or tag)")
	}
	sha := args[0]

	owner, repoName, err := resolveRepoFromFlags(commitsFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.RepoCommitGet(owner, repoName, sha)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var cm types.Commit
	if err := client.Get(ctx, path, &cm); err != nil {
		return fmt.Errorf("viewing commit: %w", err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(cm, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	printCommitSummary(owner, repoName, &cm)
	if cm.Stats != nil {
		fmt.Printf("  Changes: +%d -%d (total %d)\n",
			cm.Stats.Additions, cm.Stats.Deletions, cm.Stats.Total)
	}
	if len(cm.Files) > 0 {
		printCommitFiles(cm.Files)
	}
	if cm.HTMLURL != "" {
		fmt.Printf("  URL:   %s\n", color.CyanString(cm.HTMLURL))
	}
	return nil
}

// resolveRepoFromFlags is a thin wrapper around resolveRepoArg that
// also unpacks the [2]string into owner / repoName. Used by both
// runCommitsList and runCommitsView so the lookup pattern stays in one
// place.
func resolveRepoFromFlags(local string) (string, string, error) {
	parts, err := resolveRepoArg(local)
	if err != nil {
		return "", "", err
	}
	return parts[0], parts[1], nil
}

// printCommitSummary renders a single commit's metadata (SHA, author,
// committer, date, message) to stdout in plaintext.
func printCommitSummary(owner, repoName string, cm *types.Commit) {
	fmt.Printf("Commit %s in %s/%s\n", shortSHA(cm.SHA), owner, repoName)
	if cm.Commit != nil {
		if name := cm.Commit.Author.Name; name != "" {
			fmt.Printf("  Author:    %s <%s>\n", name, cm.Commit.Author.Email)
		}
		if name := cm.Commit.Committer.Name; name != "" {
			fmt.Printf("  Committer: %s <%s>\n", name, cm.Commit.Committer.Email)
		}
		if cm.Commit.Author.Date != "" {
			fmt.Printf("  Date:      %s\n", cm.Commit.Author.Date)
		}
		if msg := commitMessage(cm); msg != "" {
			fmt.Println("  Message:")
			for _, line := range strings.Split(strings.TrimRight(msg, "\n"), "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
	}
	if len(cm.Parents) > 0 {
		parents := make([]string, 0, len(cm.Parents))
		for _, p := range cm.Parents {
			parents = append(parents, shortSHA(p.SHA))
		}
		fmt.Printf("  Parents:   %s\n", strings.Join(parents, ", "))
	}
}

// printCommitFiles renders the per-file diff table for a commit detail
// response. Each row shows the status (added/modified/removed/renamed),
// the filename, and the +/- counts.
func printCommitFiles(files []types.CommitFile) {
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

// shortSHA trims a 40-char SHA to its first 7 characters, mirroring the
// convention used by `git log --oneline`. SHAs shorter than 8 chars are
// returned unchanged (so a short-SHA input still prints verbatim).
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// authorName picks the most useful author display label for the table:
// the Gitee username if present, otherwise the git author name, otherwise "-".
func authorName(cm *types.Commit) string {
	if cm == nil {
		return "-"
	}
	if cm.Author != nil && cm.Author.Login != "" {
		return cm.Author.Login
	}
	if cm.Commit != nil {
		if cm.Commit.Author.Username != "" {
			return cm.Commit.Author.Username
		}
		if cm.Commit.Author.Name != "" {
			return cm.Commit.Author.Name
		}
	}
	return "-"
}

// commitDate returns the author date (YYYY-MM-DD) when present, else "".
func commitDate(cm *types.Commit) string {
	if cm == nil || cm.Commit == nil {
		return ""
	}
	d := cm.Commit.Author.Date
	if len(d) < 10 {
		return d
	}
	return d[:10]
}

// commitMessage prefers the top-level Message field, falling back to
// the nested commit.message. Gitee usually populates both, but a few
// payloads omit the top-level field.
func commitMessage(cm *types.Commit) string {
	if cm == nil {
		return ""
	}
	if cm.Message != "" {
		return cm.Message
	}
	if cm.Commit != nil {
		return cm.Commit.Message
	}
	return ""
}

// firstLine returns the first line of a possibly multi-line commit
// message, used for the table view on `commits list`.
func firstLine(msg string) string {
	if i := strings.IndexAny(msg, "\r\n"); i >= 0 {
		return msg[:i]
	}
	return msg
}

const repoCommitsAgentHelp = `# gitee repo commits — Agent Help

## Purpose
Top-level umbrella for inspecting a repository's commit history. Use
these verbs to enumerate commits (optionally filtered by path, author,
or date) or fetch the full detail of a single commit including its
diff stat and per-file changes.

## Subcommands
  list            List commits with optional filtering.
                  See ` + "`" + `gitee repo commits list --agent-help` + "`" + `.
  view <sha>      View one commit in detail.
                  See ` + "`" + `gitee repo commits view --agent-help` + "`" + `.

## Pre-conditions
- Authenticated (gitee auth status)
- --repo / -R is set in the form ` + "`" + `owner/repo` + "`" + `, or
  ` + "`" + `--repo` + "`" + ` is set globally, or the local git remote resolves it
- The repo is accessible to the token (public, or the token has read access)

## Idempotency
- list:    YES. Read-only; calling repeatedly yields the same result
           (modulo upstream commits between calls).
- view:    YES. Read-only.

## Atomicity
Each verb is a single API call. No partial states.

## Side effects
None.

## Output (stdout)
- list default:  table with SHA, AUTHOR, DATE, MESSAGE
- list -o json:  []types.Commit
- view default:  metadata + message + diff stat + per-file table
- view -o json:  types.Commit (includes stats + files)

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour authenticated. Use ` + "`" + `--paginate` + "`" + `
for walks that span multiple pages.

## See also
- ` + "`" + `gitee repo commits list` + "`" + `  — filtered commit history
- ` + "`" + `gitee repo commits view` + "`" + `  — single commit + diff stat
- ` + "`" + `gitee repo compare` + "`" + `      — diff two refs
- ` + "`" + `gitee search commits` + "`" + `   — search commits across repos
- ` + "`" + `gitee --agent-help` + "`" + `      — overview of all commands
`

const repoCommitsListAgentHelp = `# gitee repo commits list — Agent Help

## Purpose
Enumerate commits on a repository, optionally filtered by branch/SHA,
path, author, or date range. The default branch is walked when no
` + "`" + `--sha` + "`" + ` is supplied.

## Pre-conditions
- Authenticated with read access to the repo
- --repo / -R is set
- The optional filters, if set, must be well-formed:
    --author   Gitee username (exact match, no wildcards)
    --since    ISO-8601 timestamp, e.g. 2026-01-01T00:00:00Z
    --until    ISO-8601 timestamp; non-inclusive on Gitee v5
    --path     repo-relative path, no leading slash

## Optional flags
  --repo / -R    Repository in owner/repo form.
  --sha          Branch, tag, or SHA to start walking from.
  --path         Only commits touching this path.
  --author       Filter by Gitee username.
  --since        Earliest commit timestamp (ISO-8601).
  --until        Latest commit timestamp (ISO-8601).
  --paginate     Walk all pages; --limit 0 for "everything".
  --limit / -L   (global) cap on total commits returned.
  --output / -o  (global) json | table (default table).

## Idempotency
YES. Read-only; calling repeatedly yields the same result (modulo
upstream commits between calls).

## Atomicity
Atomic per request. Each API call returns one page; --paginate stitches
multiple pages into a single response.

## Side effects
None.

## Post-conditions (on success)
- Stdout contains the commit table (default) or JSON array (-o json)
- Empty result prints a friendly "No commits found in owner/repo" line

## Output (stdout, table mode)
  SHA       AUTHOR   DATE        MESSAGE
  a1b2c3d   alice    2026-06-01  Fix login bug
  b2c3d4e   bob      2026-05-29  Add tests

## Output (stdout, json mode)
  [
    {"sha":"a1b2c3d...","commit":{"message":"...","author":{...},"tree":{...}},
     "author":{"login":"alice"}, "parents":[{"sha":"..."}], ...},
    ...
  ]

## Common errors
- 401 Unauthorized:   token missing or expired
- 403 Forbidden:      repo is private and the token cannot read it
- 404 Not Found:      repo does not exist
- 422 Unprocessable:  malformed --since/--until timestamp

## Race condition warning ⚠️
The shared mutable state is the branch ref being walked: --sha
` + "`" + `main` + "`" + ` (or the default branch when --sha is omitted) resolves to
whatever HEAD is AT REQUEST TIME. A push landing between pages of a
--paginate walk can shift the window — the same commit may appear
twice, or a commit may be skipped entirely, because page N+1 is
fetched against a moved ref. Mitigation: for a stable snapshot, pin
--sha to a full 40-char commit SHA (immutable) instead of a branch
name; or re-run and diff if exactness matters. The list→act gap also
applies: a commit you select here as a PR base or tag target can be
force-pushed away before the follow-up call — re-verify with
` + "`" + `gitee repo commits view --repo owner/repo <sha>` + "`" + ` immediately before
acting on it. Retrying THIS command is always safe (read-only).

## Next steps (orchestration hints)
After listing commits, an agent typically wants to:
1. View a specific commit in detail:
   ` + "`" + `gitee repo commits view --repo owner/repo <sha>` + "`" + `
2. Diff two refs:
   ` + "`" + `gitee repo compare --repo owner/repo main...feature/x` + "`" + `
3. Search commit messages across repos:
   ` + "`" + `gitee search commits <query>` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour authenticated. --paginate issues one request per page.

## Example
  # Latest commits on the default branch
  gitee repo commits list --repo myorg/myrepo

  # Only alice's commits since the new year
  gitee repo commits list --repo myorg/myrepo --author alice --since 2026-01-01

  # Everything touching a specific file
  gitee repo commits list --repo myorg/myrepo --path src/main.go --paginate

  # JSON for jq filtering
  gitee repo commits list --repo myorg/myrepo -o json \\
    | jq -r '.[] | "\(.sha[0:7])\t\(.commit.message)"'

## See also
- ` + "`" + `gitee repo commits view` + "`" + `  — single commit + diff stat
- ` + "`" + `gitee repo compare` + "`" + `      — diff two refs
- ` + "`" + `gitee search commits` + "`" + `   — search across repos
`

const repoCommitsViewAgentHelp = `# gitee repo commits view — Agent Help

## Purpose
Fetch the full detail of a single commit — author, committer, date,
message, parent SHAs, the +/- diff stat, and the per-file change list.
The <sha> argument accepts a full 40-char SHA, a short SHA prefix, a
branch name, or a tag name; Gitee resolves the input server-side.

## Pre-conditions
- Authenticated with read access to the repo
- --repo / -R is set
- <sha> is provided and resolves to exactly one commit

## Required arguments
  <sha>        Full SHA, short SHA, branch, or tag. Disambiguating
               between a short SHA and a branch name with the same
               prefix is Gitee's responsibility; if ambiguous the API
               returns 422.

## Optional flags
  --repo / -R   Repository in owner/repo form.
  --output / -o (global) json | table (default table).

## Idempotency
YES. Read-only.

## Atomicity
Atomic per request. One API call returns the full detail.

## Side effects
None.

## Post-conditions (on success)
- Stdout contains the commit summary + diff stat + per-file table
  (default) or a single Commit JSON object (-o json)

## Output (stdout, table mode)
  Commit a1b2c3d in owner/repo
    Author:    Alice <alice@example.com>
    Committer: Alice <alice@example.com>
    Date:      2026-06-01T12:34:56Z
    Message:
      Fix login bug
    Parents:   b2c3d4e, c3d4e5f
    Changes:   +12 -3 (total 15)
    Files:
      STATUS   ADD  DEL  FILE
      MODIFIED +8   -2   src/auth.go
      ADDED    +4   -1   test/auth_test.go
    URL:       https://gitee.com/owner/repo/commit/a1b2c3d

## Output (stdout, json mode)
  {
    "sha": "a1b2c3d...",
    "commit": {
      "author":   {"name":"Alice","email":"alice@example.com","date":"..."},
      "committer":{"name":"Alice","email":"alice@example.com","date":"..."},
      "message":  "Fix login bug",
      "tree":     {"sha":"..."},
      "comment_count": 0
    },
    "author":     {"login":"alice", ...},
    "committer":  {"login":"alice", ...},
    "parents":    [{"sha":"b2c3d4e..."}, {"sha":"c3d4e5f..."}],
    "stats":      {"additions":12,"deletions":3,"total":15},
    "files":      [{"filename":"src/auth.go","additions":8,"deletions":2,"status":"modified",...}, ...],
    "html_url":   "https://gitee.com/owner/repo/commit/a1b2c3d"
  }

## Common errors
- 401 Unauthorized:   token missing or expired
- 403 Forbidden:      repo is private and the token cannot read it
- 404 Not Found:      <sha> does not resolve to any commit
- 422 Unprocessable:  <sha> is ambiguous (e.g. short SHA matches multiple)

## Race condition warning ⚠️
Full SHAs identify immutable commits, so a view-by-SHA result never
goes stale. The race is in the MOVABLE ref forms: when <sha> is a
branch name or tag, Gitee resolves it to the current tip at request
time — a push or tag move between your call and a downstream action
means you inspected commit X but the branch now points at commit Y.
Classic failure: you view ` + "`" + `main` + "`" + `, decide it is safe to tag or
release, and by the time ` + "`" + `gitee repo tag create --refs main` + "`" + ` runs,
main has moved and the tag lands on the wrong commit. Mitigation:
capture the full ` + "`" + `sha` + "`" + ` field from THIS output and pass that (not
the branch name) to the follow-up write. A short-SHA prefix can also
BECOME ambiguous later (new commits with the same prefix → 422);
prefer the full SHA everywhere. Retrying THIS command is safe.

## Next steps (orchestration hints)
After viewing a commit, an agent typically wants to:
1. See the diff between this commit and its first parent:
   ` + "`" + `gitee repo compare --repo owner/repo <sha>~1...<sha>` + "`" + `
2. Find the PR that introduced this commit (if any):
   ` + "`" + `gitee search issues <sha>` + "`" + ` (commit messages are indexed)
3. Walk back through history: feed the parent SHA into ` + "`" + `commits view` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
1000 req/hour authenticated. One request per call.

## Example
  # Plain view of a short SHA
  gitee repo commits view --repo myorg/myrepo a1b2c3d

  # View the latest commit on main
  gitee repo commits view --repo myorg/myrepo main

  # View a tagged release commit
  gitee repo commits view --repo myorg/myrepo v1.0.0

  # JSON for parsing
  gitee repo commits view --repo myorg/myrepo a1b2c3d -o json \\
    | jq '.files[] | select(.additions > 0) | .filename'

## See also
- ` + "`" + `gitee repo commits list` + "`" + `  — find a SHA to view
- ` + "`" + `gitee repo compare` + "`" + `      — diff two SHAs/branches/tags
- ` + "`" + `gitee pr diff` + "`" + `          — diff two branches via PR number
`
