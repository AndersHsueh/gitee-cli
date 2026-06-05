package status

// statusAgentHelp is the agent-targeted help printed when the user runs
// `gitee status --agent-help`. It is intentionally longer and more
// structured than the default `--help` output: agents read carefully,
// humans skim.
//
// The 14 sections match the convention established by P1b/P1c commands
// (gitee webhook, gitee pr edit, gitee pr checks, etc.) so agents can
// rely on a stable document shape.
const statusAgentHelp = `# gitee status — Agent Help

## Purpose
Show a focused, at-a-glance view of what's currently waiting on the
authenticated user in a Gitee repository, in two sections:

  1. Issues assigned to you  (server-side filter: ?assignee=<me>)
  2. PRs awaiting your review (client-side filter: open && not authored by you)

` + "`" + `gitee status` + "`" + ` mirrors ` + "`" + `gh status` + "`" + ` for the Gitee platform, with the
caveats listed in "Known limitations" below.

## Pre-conditions
- Authenticated with read access to the target repo
  (env var GITEE_TOKEN, or 'gitee auth login' must have been run)
- The 'origin' git remote either points to a Gitee host (HTTPS or SSH
  form), or the user passes --repo owner/repo explicitly
- CWD is inside a git working tree (auto-detect path) OR --repo is
  supplied (explicit path)

## Required arguments
  (none)

## Optional flags
  --repo / -R     owner/repo (overrides git auto-detection)
  --output        table (default) or json
  --paginate      Fetch all pages (vs default single page)
  --limit N       Cap total items when --paginate is set
  --per-page N    Page size (max 100, default 30)

## Idempotency
YES. Read-only — no API state is mutated.

## Atomicity
Atomic (one /user call + one /issues call + one /pulls call, all reads).

## Side effects
None. Pure GETs.

## Post-conditions (on success)
- Stdout contains 2 sections (table mode) or 1 JSON object (json mode)
- The repo and the current branch (if any) are printed in the header
- The current user's login is printed under the header
- Empty sections are rendered as "none" (table) or count: 0 (json) — not
  an error

## Output (stdout, table mode)
  owner/repo  on branch feature/x
  current user: andershsueh

  Issues assigned to you  (2)
  #123  Add unit tests for parser
  #127  Fix race in cache invalidation

  PRs awaiting your review  (3)
  #41  feat: add webhook filter      feature/cool-thing
  #42  fix: nil pointer in run.go    fix/bug-42
  #43  docs: update README           docs/readme-tweak

## Output (stdout, json mode)
  {
    "repo": "owner/repo",
    "user": "andershsueh",
    "assigned_to_you": {
      "title": "Issues assigned to you",
      "count": 2,
      "items": [{"number": "123", "title": "...", "url": "..."}]
    },
    "awaiting_review": {
      "title": "PRs awaiting your review",
      "count": 3,
      "items": [{"number": 41, "title": "...", "url": "...", "branch": "feature/x"}]
    },
    "open_prs_total": 12
  }

  Note: issue numbers are STRINGS in the Gitee v5 issue API but INTS
  in the PR API. The JSON preserves the source types — do not assume
  uniform number typing when parsing.

## Common errors
- 401 Unauthorized:  token missing or expired. Run 'gitee auth login'.
- 403 Forbidden:     read scope is sufficient; write isn't required.
- 404 Not Found:     either the repo doesn't exist (private, renamed,
                     deleted) or there are no matching issues/PRs (in
                     which case the section is empty, not an error).
- "Not inside a git working tree": you ran this in a non-git dir. Use
  --repo owner/repo to override.
- "Origin points to ... not a Gitee repository": your 'origin' remote
  is on a non-Gitee host (e.g. github.com). Use --repo owner/repo.

## Known limitations

### 1. "Awaiting my review" is client-side filtered
Gitee v5's /repos/{o}/{r}/pulls endpoint does NOT support
?reviewer=<me> (unlike GitHub's /repos/{o}/{r}/pulls?reviewer=).
The CLI fetches all open PRs and drops the ones where the current
user is the author.

Consequence: a PR where you ALREADY reviewed and the author pushed a
new commit will still appear in "awaiting review" (the list endpoint
doesn't expose the user's review status). This is the same
approximation ` + "`" + `gh status` + "`" + ` makes.

If the agent needs the precise "unreviewed by me" set, use the
specific PR's reviews endpoint (not yet exposed in gitee-cli —
see common-todos.md P2 roadmap).

### 2. "Assigned to me" requires a server-side match
The /repos/{o}/{r}/issues endpoint with ?assignee=<me> works for
direct assignment only. Issues where the user is a WATCHER or
@MENTIONED but not the assignee do not appear. This matches ` + "`" + `gh status` + "`" + `
behavior.

### 3. Auto-detection is single-remote
We read ONLY the 'origin' remote. If your fork workflow uses 'upstream'
or a non-standard remote name, pass --repo explicitly.

## Next steps (orchestration hints)
1. Run ` + "`" + `gitee status` + "`" + ` in your repo dir.
2. For an issue in "assigned to you", inspect with
   ` + "`" + `gitee issue view <n> --repo owner/repo` + "`" + `.
3. For a PR in "awaiting review", inspect with
   ` + "`" + `gitee pr view <n> --repo owner/repo` + "`" + ` and review with
   ` + "`" + `gitee pr review <n> --repo owner/repo --event APPROVE` + "`" + `.

## OAuth scope required
projects (read). 'user_info' is also used for the cached username
fallback path.

## Rate limit
Gitee API v5: 1000 req/hour. Each ` + "`" + `gitee status` + "`" + ` call = 3 requests
(/user + /issues + /pulls), so up to ~333 status calls per hour.

## Example (closing the loop)
  # 1. In your gitee-cli working dir, run status
  cd ~/src/gitee-cli
  gitee status

  # 2. If it says "not in a Gitee repository", you probably
  #    cloned from a mirror or a github.com fork. Override:
  gitee status --repo andershsueh/gitee-cli

  # 3. JSON output for an agent pipeline
  gitee status --repo owner/repo --output json | jq '.awaiting_review.items[].number'

## See also
- ` + "`" + `gitee issue list --assignee <user>` + "`" + `   — raw query, no auto-detect
- ` + "`" + `gitee pr list --state open` + "`" + `             — raw query, no auto-detect
- ` + "`" + `gitee auth status` + "`" + `                       — confirm the username used here
- ` + "`" + `common-todos.md` + "`" + `                          — P2 roadmap (status, alias, extension)
`

// printStatusAgentHelpImpl writes the const to stdout. Called from
// PreRunE in status.go. The indirection through a function (rather than
// inlining `fmt.Print(statusAgentHelp)` in status.go) keeps the agent
// help text in its own file, paralleling the P1b/P1c convention.
func printStatusAgentHelpImpl() {
	fmtPrint(statusAgentHelp)
}
