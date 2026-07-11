package repo

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var deleteFlags struct {
	owner string
	repo  string
	yes   bool
}

var deleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>]",
	Short: "Delete a repository",
	Example: `  gitee repo delete owner/repo
  gitee repo delete --repo owner/repo --yes`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoDeleteAgentHelp)
		}
		return nil
	},
	RunE: runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.StringVar(&deleteFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&deleteFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.BoolVar(&deleteFlags.yes, "yes", false, "Skip confirmation")
}

func runDelete(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	owner, repoName, err := resolveRepo(deleteFlags.owner, deleteFlags.repo, args)
	if err != nil {
		return err
	}

	if !deleteFlags.yes {
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Delete repository %s/%s?", color.YellowString("?"), owner, repoName),
			false,
		)
		if err != nil {
			return err
		}
		if !yes {
			fmt.Println("Cancelled")
			return nil
		}
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReposDelete(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("deleting repository: %w", err)
	}

	fmt.Printf("Repository %s/%s deleted\n", owner, repoName)
	return nil
}

const repoDeleteAgentHelp = `# gitee repo delete — Agent Help

## Purpose
Permanently delete a repository. This is a DESTRUCTIVE write operation:
the repo, including all code, issues, PRs, wiki, releases, comments,
and history, is erased. The repository name is freed for re-use after
deletion.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Token has admin (owner) permission on the target repo
- One of the following must resolve to owner/repo:
    - positional argument:        ` + "`" + `gitee repo delete owner/repo` + "`" + `
    - --repo / -R flag:           ` + "`" + `--repo owner/repo` + "`" + `
    - split --owner + --repo:     ` + "`" + `--owner owner --repo name` + "`" + `
- For interactive runs (no --yes): the user must type ` + "`" + `y` + "`" + `, ` + "`" + `Y` + "`" + `, or ` + "`" + `yes` + "`" + `
  at the prompt. Anything else (including EOF / non-TTY) cancels.
- For automation, pass ` + "`" + `--yes` + "`" + ` to skip the prompt.

## Optional flags
  --owner         Repository owner (when --repo is just the bare name)
  --repo / -R     Repository name, OR owner/name; overrides positional arg
  --yes           Skip the y/N confirmation prompt (use in scripts/agents)

## Idempotency
PARTIAL. Calling on an already-deleted repo returns 404, which is treated
as an error by this command (you'll see "deleting repository: ..."
in stderr). There is no successful "no-op" path.

## Atomicity
Atomic. The DELETE call is single-shot — either the repo is erased, or
nothing changes. There is no partial state visible to subsequent calls.

## Side effects (IRREVERSIBLE — read carefully)
- The repository record is removed from Gitee.
- ALL of the following are permanently lost:
    * git history (every branch, tag, reflog)
    * issues, issue comments, and labels
    * pull requests, PR comments, and review records
    * wiki pages and attachments
    * releases and release assets
    * webhooks configured on the repo
    * deploy keys, collaborator permissions, branch protections
- Forks of the repo are NOT deleted; only the upstream is removed
  (forks will simply become orphaned).
- The original namespace (e.g. ` + "`" + `owner/repo` + "`" + `) becomes available
  for any user to recreate the same name.
- THIS ACTION CANNOT BE UNDONE. Gitee support cannot restore a deleted
  repo. If you only want to hide the repo from listings without losing
  data, use ` + "`" + `gitee repo archive owner/repo` + "`" + ` instead — archive
  is fully reversible via ` + "`" + `gitee repo archive ... --unarchive` + "`" + `.
- Email notifications may go to the repo owner.

## Post-conditions (on success)
- Stdout reports "Repository owner/name deleted"
- The repo returns 404 on subsequent ` + "`" + `gitee repo view` + "`" + `
- The git remote URL no longer resolves
- The namespace is freed for a future ` + "`" + `gitee repo create` + "`" + `

## Common errors
- 401 Unauthorized:  token expired or invalid             →  gitee auth login
- 403 Forbidden:     token lacks admin (owner) permission
- 404 Not Found:     repo doesn't exist (or already deleted) — verify before re-trying
- Confirmation cancelled: nothing was deleted; safe to retry
- Network/timeout:   transient; safer to retry than to repeat blindly because
                     a partial server-side state could leave the repo half-deleted
                     (in practice the API is atomic, but be cautious)

## Next steps (orchestration hints)
Destructive verbs have NO direct follow-up on the deleted repo. Typical
post-deletion next steps for an agent:

1. Verify it is gone: ` + "`" + `gitee repo view owner/repo` + "`" + ` should now fail
2. If you wanted to keep history, you goofed — recreate with
   ` + "`" + `gitee repo create --name name --auto-init=false` + "`" + ` and re-push
   from a local clone (` + "`" + `git push --mirror origin` + "`" + `).
3. If you only wanted to hide it, the right verb was ` + "`" + `gitee repo archive` + "`" + `.
   Archived repos are fully restorable with ` + "`" + `--unarchive` + "`" + `.
4. Clean up CI/CD config and downstream mirrors that pointed at this URL.

## OAuth scope required
projects (admin) — repo owner only.

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Interactive (default): prompts the user
  gitee repo delete myorg/throwaway

  # Non-interactive (CI / agent): skip the prompt
  gitee repo delete --repo myorg/throwaway --yes

  # Safer pattern: archive FIRST, delete LATER if still unwanted
  gitee repo archive --repo myorg/legacy --force
  sleep 7d   # wait, confirm no one complains
  gitee repo delete --repo myorg/legacy --yes

## See also
- ` + "`" + `gitee repo view` + "`" + `             — confirm what you're about to delete
- ` + "`" + `gitee repo archive` + "`" + `         — reversible alternative (recommended for "soft" removes)
- ` + "`" + `gitee repo list` + "`" + `            — discover repo names before deletion
- ` + "`" + `gitee --agent-help` + "`" + `          — overview of all commands with agent help
`
