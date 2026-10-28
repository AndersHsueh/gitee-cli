package repo

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

var tagFlags struct {
	repo    string
	name    string
	refs    string
	message string
}

var tagCmd = &cobra.Command{
	Use:   "tag <subcommand>",
	Short: "List or create tags in a repository",
	Long: `Manage Git tags in a Gitee repository.

  gitee repo tag list    — list existing tags
  gitee repo tag create  — create a tag at a specific commit

Tag deletion is NOT exposed by Gitee v5; use native git over SSH to
remove a tag once it exists:

  git push origin :refs/tags/<tag_name>`,
	Example: `  gitee repo tag list --repo owner/repo
  gitee repo tag create --repo owner/repo --name v1.0.0 --refs main`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoTagAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		// No subcommand given — show help so users discover list/create.
		return c.Help()
	},
}

var tagListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List tags in a repository",
	Example: `  gitee repo tag list --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoTagListAgentHelp)
		}
		return nil
	},
	RunE: runTagList,
}

var tagCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a tag pointing at a commit",
	Example: `  gitee repo tag create --repo owner/repo --name v1.0.0 --refs main
  gitee repo tag create --repo owner/repo --name v1.0.0 --refs <sha> --message "release notes"`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoTagCreateAgentHelp)
		}
		return nil
	},
	RunE: runTagCreate,
}

func init() {
	// list flags
	lf := tagListCmd.Flags()
	lf.StringVarP(&tagFlags.repo, "repo", "R", "", "Repository (owner/repo)")

	// create flags
	cf := tagCreateCmd.Flags()
	cf.StringVarP(&tagFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	cf.StringVarP(&tagFlags.name, "name", "n", "", "Tag name (e.g. v1.0.0)")
	cf.StringVarP(&tagFlags.refs, "refs", "r", "", "Commit SHA or branch to tag (e.g. main or a 40-char SHA)")
	cf.StringVarP(&tagFlags.message, "message", "m", "", "Tag annotation message (optional; omit for a lightweight tag)")
	_ = tagCreateCmd.MarkFlagRequired("name")
	_ = tagCreateCmd.MarkFlagRequired("refs")

	tagCmd.AddCommand(tagListCmd)
	tagCmd.AddCommand(tagCreateCmd)
}

func runTagList(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveRepoArg(tagFlags.repo)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.ReposTagsList(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var tags []types.Tag
	if err := client.Get(ctx, path, &tags); err != nil {
		return fmt.Errorf("listing tags: %w", err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(tags, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Tags in %s/%s:\n", owner, repoName)
	if len(tags) == 0 {
		fmt.Println("  (none)")
		return nil
	}
	for _, tg := range tags {
		fmt.Printf("  %s", tg.Name)
		if tg.Commit.SHA != "" {
			short := tg.Commit.SHA
			if len(short) > 7 {
				short = short[:7]
			}
			fmt.Printf("  -> %s", short)
		}
		fmt.Println()
	}
	return nil
}

func runTagCreate(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveRepoArg(tagFlags.repo)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	payload := types.TagCreateRequest{
		Refs:       tagFlags.refs,
		TagName:    tagFlags.name,
		TagMessage: tagFlags.message,
	}

	path := api.ReposTagsCreate(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var created types.Tag
	if err := client.Post(ctx, path, payload, &created); err != nil {
		return fmt.Errorf("creating tag: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Tag %s created in %s/%s\n", green("✓"), bold(created.Name), owner, repoName)
	if created.Commit.SHA != "" {
		short := created.Commit.SHA
		if len(short) > 7 {
			short = short[:7]
		}
		fmt.Printf("  Commit: %s\n", short)
	}
	return nil
}

// resolveRepoArg resolves --repo from the local flag (set by Cobra) or the
// global --repo flag (cmd.GlobalRepo). Returns owner, repoName on success.
// Mirrors the inline pattern in branches.go so the verb is self-contained.
func resolveRepoArg(local string) ([2]string, error) {
	repo := local
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	if repo == "" {
		return [2]string{}, fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return [2]string{}, fmt.Errorf("invalid --repo format, expected owner/repo")
	}
	return [2]string{parts[0], parts[1]}, nil
}

const repoTagAgentHelp = `# gitee repo tag — Agent Help

## Purpose
Top-level umbrella for tag operations: list tags that exist in a
Gitee repository, and create new tags pointing at existing commits.
Use this before any release workflow (tags back releases), before
noting canonical version markers, or before invoking native git to
push to a remote.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The target repo exists and is visible to the token
- For tag create: the token must have write permission on the repo
  (member, collaborator with push, or owner)
- --repo is supplied in the form ` + "`" + `owner/repo` + "`" + ` (or ` + "`" + `--repo` + "`" + `
  is set globally; falls back to the local git remote). Validated locally;
  missing or invalid format returns an error before any API call.

## Subcommands
  list     List existing tags. See ` + "`" + `gitee repo tag list --agent-help` + "`" + `.
  create   Create a tag at a specific commit or branch. See
           ` + "`" + `gitee repo tag create --agent-help` + "`" + `.

Note: Gitee v5 does NOT expose a tag-delete endpoint. To remove a tag,
use native git:
  ` + "`" + `git push origin :refs/tags/<tag_name>` + "`" + `
(or the ` + "`" + `:tag` + "`" + ` syntax). This tool does not invent a verb for it.

## Idempotency
- list:    YES. Read-only.
- create:  NOT idempotent. Re-running with the same name returns 422
           ("tag already exists"). Use list first to check existence.

## Atomicity
- list:    Atomic. Single API call returns the full tag list.
- create:  Atomic. Either the tag exists after the call or nothing changed.

## Side effects
- list:    None. Read-only.
- create:  A new tag reference is created in the repo. Subscribers to
           repo events may be notified (e.g. CI on tag push triggers).
           If a release was already configured against this tag, the
           release is unaffected — only the underlying ref is new.

## Output (stdout)
- list default:  plaintext list of tag names (and shortened SHAs when available)
- list -o json:  JSON array of full tag objects
- create:        success line with tag name and short SHA

## OAuth scope required
- list:    projects (read)
- create:  projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call counts as 1 request.

## See also
- ` + "`" + `gitee repo tag list` + "`" + `    — enumerate tags
- ` + "`" + `gitee repo tag create` + "`" + `  — create a tag
- ` + "`" + `gitee repo branches` + "`" + `   — choose a source branch (e.g. main)
- ` + "`" + `gitee release list` + "`" + `   — tags back releases
- ` + "`" + `gitee --agent-help` + "`" + `    — overview of all commands with agent help
`

const repoTagListAgentHelp = `# gitee repo tag list — Agent Help

## Purpose
List every tag in a Gitee repository. Use this as a pre-flight for
release flows, for verifying whether a tag exists, or for orchestrating
downstream actions keyed on tag names.

## Pre-conditions
- Authenticated (gitee auth status)
- The target repo exists and is visible to the token
- --repo is supplied in the form ` + "`" + `owner/repo` + "`" + ` (or ` + "`" + `--repo` + "`" + `
  is set globally; falls back to the local git remote).

## Optional flags
  --repo / -R     Repository in ` + "`" + `owner/repo` + "`" + ` form (falls back to ` + "`" + `--repo` + "`" + `
                  on the global flag, then to git remote).

## Idempotency
YES. Read-only; calling repeatedly yields the same list (assuming no
upstream tag create/delete between calls). Safe to retry.

## Atomicity
Atomic per request. A single API call returns the full tag list
(server may paginate internally; this verb returns all of them).

## Side effects
None. Read-only.

## Post-conditions
- Stdout is a plaintext list (default) or JSON array (-o json)
- Process exits 0 even when the repo has zero tags (prints "(none)")
- No state change on the server, no local cache write

## Output (stdout, table mode)
  Tags in owner/repo:
    v1.0.0   -> abc1234
    v1.1.0   -> deadbee
    v2.0.0-rc.1 -> 1234567

The "->" trailing value is the first 7 chars of the commit SHA that
the tag points at (full SHA is in JSON mode).

## Output (stdout, json mode)
  [
    {
      "name": "v1.0.0",
      "message": "release notes for 1.0.0",
      "commit": {
        "sha": "abc1234...",
        "url": "https://gitee.com/owner/repo/commit/abc1234..."
      },
      "tagger": null
    }
  ]

Field names follow the Gitee API v5 /repos/{owner}/{repo}/tags
response. Use ` + "`" + `.name` + "`" + ` as input to other commands (release view, etc.).

## Common errors
- 401 Unauthorized:  token expired or invalid   →  gitee auth login
- 403 Forbidden:     repo is private and token cannot read it
- 404 Not Found:     owner/name doesn't exist; check spelling
- "invalid --repo format": must contain exactly one ` + "`" + `/` + "`" + `
- Network/timeout:   transient; safe to retry

## Race condition warning ⚠️
The shared mutable state is the repo's tag set. This listing is a
point-in-time snapshot: another agent's ` + "`" + `gitee repo tag create` + "`" + ` or
a ` + "`" + `git push :refs/tags/<name>` + "`" + ` can add/remove tags in the gap
before you act. Classic failures: (1) you confirm "v1.2.0 is free"
from this list, then ` + "`" + `gitee repo tag create --name v1.2.0` + "`" + ` fails
422 because a concurrent release job claimed it — treat the 422 as
the real availability check and pick the next version; (2) you take
the tag→commit SHA mapping from this output and base a release or
diff on it, but a force-push MOVED the tag (native git can move tags;
Gitee will follow) so your release points at the wrong commit.
Mitigation: re-run THIS command immediately before acting on a
specific tag, and prefer pinning the full commit SHA from the JSON
output over trusting the tag name downstream. Retrying is safe.

## Next steps (orchestration hints)
After listing tags, an agent typically wants to:
1. Pick a tag of interest (e.g. ` + "`" + `v1.0.0` + "`" + `).
2. Inspect the matching release:
   ` + "`" + `gitee release list --repo owner/repo` + "`" + ` (filter by tag_name in JSON)
3. Create a new tag at HEAD or a specific branch:
   ` + "`" + `gitee repo tag create --repo owner/repo --name v1.2.0 --refs main` + "`" + `
4. Delete a tag (Gitee v5 has no API; use native git):
   ` + "`" + `git push origin :refs/tags/<tag_name>` + "`" + `

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Plainlist
  gitee repo tag list --repo myorg/billing-svc

  # JSON for jq filtering
  gitee repo tag list --repo myorg/billing-svc -o json \\
    | jq -r '.[].name'

  # Pick the latest semver tag and view its release
  TAG=$(gitee repo tag list --repo myorg/billing-svc -o json \\
    | jq -r '.[0].name')
  gitee release view --repo myorg/billing-svc "$TAG"

## See also
- ` + "`" + `gitee repo tag create` + "`" + ` — add a new tag
- ` + "`" + `gitee repo branches` + "`" + `  — see source branches
- ` + "`" + `gitee release list` + "`" + `  — see releases (one per tag)
- ` + "`" + `gitee --agent-help` + "`" + `   — overview of all commands with agent help
`

const repoTagCreateAgentHelp = `# gitee repo tag create — Agent Help

## Purpose
Create a Git tag in a Gitee repository, pointing at an existing commit
or branch. Tags are immutable named pointers — they back releases and
are the canonical way to mark a snapshot as "this is version X".

## Pre-conditions
- Authenticated with projects (write) scope on the target repo
- The source commit (refs) MUST already exist on the Gitee remote:
  - branch name (e.g. ` + "`" + `main` + "`" + `) is resolved to its current HEAD SHA, OR
  - full 40-char commit SHA (e.g. ` + "`" + `a1b2c3d4...` + "`" + `)
- --repo is supplied in the form ` + "`" + `owner/repo` + "`" + ` (or ` + "`" + `--repo` + "`" + `
  is set globally; falls back to the local git remote).
- --name is non-empty and unique within the repo
  (enforced by Cobra MarkFlagRequired)
- --refs is non-empty
  (enforced by Cobra MarkFlagRequired)

## Required flags
  --repo / -R     Repository in ` + "`" + `owner/repo` + "`" + ` form (falls back to ` + "`" + `--repo` + "`" + `
                  on the global flag, then to git remote).
  --name / -n    Tag name. Gitee tag names follow Git ref-name rules:
                 alphanumerics, dots, slashes, dashes, underscores.
                 Common pattern: ` + "`" + `v<major>.<minor>.<patch>` + "`" + ` (e.g. v1.0.0).
  --refs / -r    Source ref to tag. Either a branch name (e.g. ` + "`" + `main` + "`" + `)
                 or a full 40-char commit SHA.

## Optional flags
  --message / -m   Tag annotation message. With a message, the tag is
                   "annotated" and shows up in ` + "`" + `git show <tag>` + "`" + ` with
                   its own metadata. Without a message, the tag is
                   "lightweight" and just points at the commit.

## Idempotency
NOT idempotent. Re-running with the same --name returns 422
("tag already exists"). To move a tag to a different commit, delete
it first with ` + "`" + `git push origin :refs/tags/<tag_name>` + "`" + ` and re-create.

## Atomicity
Atomic. Either the tag exists after the call or nothing changed.
There is no partial state.

## Side effects
- A new tag reference is created in the repo
- Subscribers to repo events (e.g. CI on tag push) may be triggered
  immediately
- Webhooks subscribed to "Push" events on the repo will fire
- If a release was already configured to be auto-created on this tag,
  the matching release may also be created (depends on repo settings)

## Post-conditions (on success)
- Stdout prints "✓ Tag <name> created in owner/repo" plus the short SHA
- The tag is immediately visible via ` + "`" + `gitee repo tag list` + "`" + `

## Output (stdout, table mode)
  ✓ Tag v1.0.0 created in owner/repo
    Commit: a1b2c3d

## Common errors
- 401 Unauthorized:  token expired or invalid   →  gitee auth login
- 403 Forbidden:     token lacks write scope on this repo
- 404 Not Found:     repo does not exist, or --refs SHA/branch not found
- 422 Unprocessable: tag name already exists, or invalid characters
- "invalid --repo format": must contain exactly one ` + "`" + `/` + "`" + `

## Race condition warning ⚠️
If another agent or human is concurrently creating tags on this repo,
the second --name will fail with 422. Re-list and pick a fresh name.
There is no implicit monotonic version allocator — callers must
manage that themselves.

## Next steps (orchestration hints)
After a successful tag create, an agent typically wants to:
1. Verify: ` + "`" + `gitee repo tag list --repo owner/repo` + "`" + `
2. Attach a release:
   ` + "`" + `gitee release create --repo owner/repo --tag v1.0.0 --name "v1.0.0" --target main` + "`" + `
3. Or update an existing release that targets this tag:
   ` + "`" + `gitee release edit --repo owner/repo v1.0.0 --body "..."` + "`" + `
4. Push to a local mirror so other CI / clones see it:
   ` + "`" + `git fetch origin tag v1.0.0` + "`" + `

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Lightweight tag at the tip of main
  gitee repo tag create --repo myorg/billing-svc \\
    --name v1.0.0 --refs main

  # Annotated tag at a specific commit SHA
  gitee repo tag create --repo myorg/billing-svc \\
    --name v1.0.0 --refs a1b2c3d4e5f67890... \\
    --message "Release 1.0.0: initial GA"

  # Pipeline: tag then release
  gitee repo tag create --repo myorg/billing-svc \\
    --name v1.1.0 --refs main
  gitee release create --repo myorg/billing-svc \\
    --tag v1.1.0 --name "v1.1.0" \\
    --body "$(cat CHANGELOG.md)"

## See also
- ` + "`" + `gitee repo tag list` + "`" + `    — verify the tag exists
- ` + "`" + `gitee release create` + "`" + `  — attach a release to the new tag
- ` + "`" + `gitee repo branches` + "`" + `   — pick a source branch for --refs
- ` + "`" + `gitee --agent-help` + "`" + `    — overview of all commands with agent help
`
