package repo

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Per the P2-12 spec, the branch-protection subcommand group lives entirely
// in this one file (`cmd/repo/protect.go`) and registers two verbs under
// the `gitee repo protect` umbrella:
//
//   protect     — PUT    /repos/{owner}/{repo}/branches/{branch}/protection
//   unprotect   — DELETE /repos/{owner}/{repo}/branches/{branch}/protection
//
// Both verbs share the --repo / -R flag pattern and take the branch
// name as a positional argument. The umbrella command also accepts
// `gitee repo unprotect <branch>` as an alias so users can run it from
// the repo root in the same way gh does.

// protectFlags is shared across protect/unprotect so the user can
// invoke `gitee repo protect --repo owner/repo main` (Cobra binds by
// pointer).
var protectFlags struct {
	repo string
}

var protectCmd = &cobra.Command{
	Use:   "protect <subcommand>",
	Short: "Enable or disable branch protection",
	Long: `Manage branch protection on a Gitee repository.

Branch protection prevents force-pushes and direct pushes from anyone
without admin rights. Use these verbs to lock a release-bearing
branch (e.g. main, master, release/*) so accidental pushes can't
break it, and to unlock it again when you intentionally need to.

  gitee repo protect <branch>     — enable branch protection
  gitee repo unprotect <branch>   — remove branch protection`,
	Example: `  gitee repo protect main --repo owner/repo
  gitee repo unprotect main --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoProtectAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		// No subcommand given — show help so users discover protect/unprotect.
		return c.Help()
	},
}

var protectProtectCmd = &cobra.Command{
	Use:   "protect <branch>",
	Short: "Enable branch protection on a repository branch",
	Example: `  gitee repo protect main --repo owner/repo
  gitee repo protect release/v1 --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoProtectProtectAgentHelp)
		}
		return nil
	},
	RunE: runRepoProtect,
}

var protectUnprotectCmd = &cobra.Command{
	Use:   "unprotect <branch>",
	Short: "Remove branch protection from a repository branch",
	Example: `  gitee repo unprotect main --repo owner/repo
  gitee repo unprotect release/v1 --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoProtectUnprotectAgentHelp)
		}
		return nil
	},
	RunE: runRepoUnprotect,
}

// repoUnprotectCmd is a top-level alias for `gitee repo protect
// unprotect <branch>` so the verb reads naturally at the repo root:
// `gitee repo unprotect main --repo owner/repo`. Mirrors how star /
// unstar live as siblings under RepoCmd rather than nesting.
var repoUnprotectCmd = &cobra.Command{
	Use:   "unprotect <branch>",
	Short: "Remove branch protection from a repository branch (alias of `gitee repo protect unprotect`)",
	Example: `  gitee repo unprotect main --repo owner/repo
  gitee repo unprotect release/v1 --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoProtectUnprotectAgentHelp)
		}
		return nil
	},
	RunE: runRepoUnprotect,
}

func init() {
	// protect flags
	pf := protectProtectCmd.Flags()
	pf.StringVarP(&protectFlags.repo, "repo", "R", "", "Repository (owner/repo); falls back to global --repo, then to the positional arg")

	// unprotect flags (under the umbrella)
	uf := protectUnprotectCmd.Flags()
	uf.StringVarP(&protectFlags.repo, "repo", "R", "", "Repository (owner/repo); falls back to global --repo, then to the positional arg")

	// top-level alias flags
	rf := repoUnprotectCmd.Flags()
	rf.StringVarP(&protectFlags.repo, "repo", "R", "", "Repository (owner/repo); falls back to global --repo, then to the positional arg")

	protectCmd.AddCommand(protectProtectCmd)
	protectCmd.AddCommand(protectUnprotectCmd)
}

// resolveProtectBranchArg returns the branch name from the positional
// argument (required) and validates it. Branch names follow Git ref-name
// rules; we leave the actual character validation to Gitee so the CLI
// stays a thin transport, but we do reject the empty string up front
// so the failure mode is "branch is required" rather than a 404 from
// Gitee.
func resolveProtectBranchArg(args []string) (string, error) {
	if len(args) == 0 || args[0] == "" {
		return "", fmt.Errorf("branch is required (e.g. `gitee repo protect main --repo owner/repo`)")
	}
	return args[0], nil
}

func runRepoProtect(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveRepoArg(protectFlags.repo)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	branch, err := resolveProtectBranchArg(args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.RepoBranchProtection(owner, repoName, branch)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	// PUT returns 204 No Content; we don't decode a body.
	if err := client.Put(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("enabling branch protection: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s Protected branch %s on %s/%s\n",
		green("✓"), yellow(branch), owner, repoName)
	return nil
}

func runRepoUnprotect(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveRepoArg(protectFlags.repo)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	branch, err := resolveProtectBranchArg(args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.RepoUnprotectBranch(owner, repoName, branch)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("removing branch protection: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s Unprotected branch %s on %s/%s\n",
		green("✓"), yellow(branch), owner, repoName)
	return nil
}

const repoProtectAgentHelp = `# gitee repo protect — Agent Help

## Purpose
Top-level umbrella for branch protection operations: enable branch
protection on a specific branch (so non-admins cannot push directly
or force-push) and remove that protection. Protection is binary at
this Gitee v5 API level — there are no granular options exposed (e.g.
"required reviewers", "dismiss stale approvals") as GitHub offers.

## Subcommands
  protect <branch>     Enable branch protection. See
                       ` + "`" + `gitee repo protect --agent-help` + "`" + `.
  unprotect <branch>   Remove branch protection. See
                       ` + "`" + `gitee repo unprotect --agent-help` + "`" + `.

## Pre-conditions
- Authenticated (gitee auth status)
- --repo owner/repo is set (or global --repo, or local git remote)
- Caller has admin permission on the repo (Gitee enforces this; 403
  otherwise)
- <branch> is the branch name as a positional argument
- The branch must already exist on the remote — protecting a
  non-existent branch returns 422 ("branch not found")

## Idempotency
- protect:    YES. Re-protecting an already-protected branch is a
              no-op on the server; the post-condition (protected) holds.
- unprotect:  YES. Re-unprotecting an already-unprotected branch returns
              404, but the post-condition (unprotected) holds. Safe to
              retry.

## Atomicity
Each verb is a single API call. Either the protection state changes
atomically or nothing changes.

## Side effects
- protect:    The branch's ` + "`" + `protected` + "`" + ` flag is set to true on the server.
              Direct pushes (and force-pushes) by non-admins are rejected
              immediately. Webhooks subscribed to "Push" events continue
              to fire for pushes by admins.
- unprotect:  The branch's ` + "`" + `protected` + "`" + ` flag is set to false. Any user
              with push access can now push directly and force-push.

## Output (stdout, table mode)
- protect:    single-line success message with branch name
- unprotect:  single-line success message with branch name

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## See also
- ` + "`" + `gitee repo branches` + "`" + `   — enumerate branches (their ` + "`" + `protected` + "`" + ` flag is the source of truth)
- ` + "`" + `gitee auth status` + "`" + `    — verify token scope before calling
- ` + "`" + `gitee --agent-help` + "`" + `    — overview of all commands with agent help
`

const repoProtectProtectAgentHelp = `# gitee repo protect — Agent Help

## Purpose
Enable branch protection on a specific branch of a Gitee repository.
After this call, the branch's ` + "`" + `protected` + "`" + ` flag is true; non-admin
pushers cannot push directly or force-push to the branch (they must
go through a pull request, which itself is subject to the repo's
review rules).

Use this to lock release-bearing branches (e.g. ` + "`" + `main` + "`" + `, ` + "`" + `master` + "`" + `,
` + "`" + `release/*` + "`" + `) so an accidental push can't break a live system.

## Pre-conditions
- Authenticated with projects (write) scope on the target repo
- Caller has admin permission on the repo (Gitee enforces this; 403
  for non-admins)
- --repo / -R is set (owner/repo)
- <branch> is the branch name as a positional argument
- The branch exists on the remote (otherwise Gitee returns 422)

## Required flags / arguments
  <branch>             Branch name to protect (e.g. ` + "`" + `main` + "`" + `, ` + "`" + `release/v1` + "`" + `).
                       Branch names follow Git ref-name rules (alphanumerics,
                       dots, slashes, dashes, underscores). Slashes are
                       allowed and URL-escaped inside the endpoint builder.

## Optional flags
  --repo / -R          Repository in ` + "`" + `owner/repo` + "`" + ` form.

## Idempotency
YES. Re-protecting an already-protected branch is a no-op on the
server; the post-condition (protected) holds. Safe to retry.

## Atomicity
Atomic. Either the branch is protected after the call or nothing
changed.

## Side effects
- The branch's ` + "`" + `protected` + "`" + ` flag is set to true immediately
- Non-admin pushers who try ` + "`" + `git push` + "`" + ` or ` + "`" + `git push --force` + "`" + ` get rejected
  with a "branch is protected" error from Gitee
- Open pull requests targeting the branch are NOT auto-closed; review
  rules (if any) continue to apply
- Webhooks subscribed to "Push" events continue to fire for admin pushes
- No email is sent (silent state change)

## Post-conditions (on success)
- Stdout prints "✓ Protected branch <branch> on <owner>/<repo>"
- A subsequent ` + "`" + `gitee repo branches --repo owner/repo` + "`" + ` shows
  ` + "`" + `protected: true` + "`" + ` for this branch

## Output (stdout, table mode)
  ✓ Protected branch main on myorg/myrepo

## Common errors
- 401 Unauthorized:  token missing or expired → gitee auth login
- 403 Forbidden:     caller is not a repo admin
- 404 Not Found:     repo does not exist
- 422 Unprocessable: branch does not exist on the remote, or the
                     branch name is invalid
- "branch is required": no positional argument was given
- "invalid --repo format": must contain exactly one '/'

## Next steps (orchestration hints)
After enabling protection, an agent typically wants to:
1. Verify: ` + "`" + `gitee repo branches --repo owner/repo -o json | jq '.[] | select(.protected)'` + "`" + `
2. Set up branch protection rules at the Gitee web UI if finer
   granularity (e.g. required reviewers) is needed — the CLI toggle
   is binary at this API level
3. Communicate the change to collaborators: pushes via PR only from
   now on
4. To undo: ` + "`" + `gitee repo unprotect <branch> --repo owner/repo` + "`" + `

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Protect main
  gitee repo protect main --repo myorg/billing-svc

  # Protect a release branch
  gitee repo protect release/v1 --repo myorg/billing-svc

  # Pipeline: protect main after CI passes on initial push
  if gitee repo branches --repo myorg/billing-svc -o json | jq -e '.[] | select(.name=="main") | select(.protected == false)' > /dev/null; then
    gitee repo protect main --repo myorg/billing-svc
  fi

## See also
- ` + "`" + `gitee repo unprotect` + "`" + `  — undo this
- ` + "`" + `gitee repo branches` + "`" + `   — verify the ` + "`" + `protected` + "`" + ` flag
- ` + "`" + `gitee pr create` + "`" + `      — the supported workflow after protection
- ` + "`" + `gitee --agent-help` + "`" + `   — overview of all commands with agent help
`

const repoProtectUnprotectAgentHelp = `# gitee repo unprotect — Agent Help

## Purpose
Remove branch protection from a specific branch of a Gitee
repository. After this call, the branch's ` + "`" + `protected` + "`" + ` flag is false;
any user with push access can push directly and force-push.

Use this when you intentionally need to bypass the normal PR-review
flow (e.g. reverting a bad merge with a force-push, or moving a tag
that points to a protected branch). Always re-protect immediately
after the exceptional operation.

## Pre-conditions
- Authenticated with projects (write) scope on the target repo
- Caller has admin permission on the repo (Gitee enforces this; 403
  for non-admins)
- --repo / -R is set (owner/repo)
- <branch> is the branch name as a positional argument

## Required flags / arguments
  <branch>             Branch name to unprotect (e.g. ` + "`" + `main` + "`" + `, ` + "`" + `release/v1` + "`" + `).

## Optional flags
  --repo / -R          Repository in ` + "`" + `owner/repo` + "`" + ` form.

## Idempotency
YES. Re-unprotecting an already-unprotected branch returns 404, but
the post-condition (unprotected) holds. Safe to retry.

## Atomicity
Atomic. Either the branch is unprotected after the call or nothing
changed.

## Side effects
- The branch's ` + "`" + `protected` + "`" + ` flag is set to false immediately
- Any user with push access can now ` + "`" + `git push` + "`" + ` and ` + "`" + `git push --force` + "`" + `
  without restriction
- No email is sent (silent state change)
- Webhooks subscribed to "Push" events continue to fire normally

## Post-conditions (on success)
- Stdout prints "✓ Unprotected branch <branch> on <owner>/<repo>"
- A subsequent ` + "`" + `gitee repo branches --repo owner/repo` + "`" + ` shows
  ` + "`" + `protected: false` + "`" + ` for this branch

## Output (stdout, table mode)
  ✓ Unprotected branch main on myorg/myrepo

## Common errors
- 401 Unauthorized:  token missing or expired → gitee auth login
- 403 Forbidden:     caller is not a repo admin
- 404 Not Found:     repo does not exist, OR the branch is not currently
                     protected (idempotent — already unprotected)
- "branch is required": no positional argument was given
- "invalid --repo format": must contain exactly one '/'

## Safety warning ⚠️
Removing protection re-enables direct pushes AND force-pushes. A
careless ` + "`" + `git push --force` + "`" + ` on ` + "`" + `main` + "`" + ` immediately after this call
will rewrite history visible to every collaborator. Either:
- Lock down who has push access to the branch before unprotecting
- Perform the exceptional operation (force-push, etc.) in the same
  orchestration step that calls unprotect, then re-protect
- Or do the operation via a temporary branch and cherry-pick back

## Next steps (orchestration hints)
After unprotecting, an agent typically wants to:
1. Perform the exceptional operation (force-push, etc.)
2. Re-protect immediately:
   ` + "`" + `gitee repo protect <branch> --repo owner/repo` + "`" + `
3. Verify the round-trip:
   ` + "`" + `gitee repo branches --repo owner/repo -o json | jq '.[] | select(.name=="<branch>")'` + "`" + `
4. Notify collaborators that the branch is unprotected again

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Plain removal (use sparingly; see safety warning above)
  gitee repo unprotect main --repo myorg/billing-svc

  # Round-trip: unprotect, force-push a fix, re-protect
  gitee repo unprotect main --repo myorg/billing-svc
  git push --force-with-lease origin main
  gitee repo protect main --repo myorg/billing-svc

## See also
- ` + "`" + `gitee repo protect` + "`" + `    — the inverse verb
- ` + "`" + `gitee repo branches` + "`" + `  — verify the ` + "`" + `protected` + "`" + ` flag
- ` + "`" + `gitee --agent-help` + "`" + `   — overview of all commands with agent help
`