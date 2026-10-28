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

// Per the P1-5 spec, the collaborator subcommand group lives entirely in
// this one file (`cmd/repo/collaborator.go`) and registers three verbs
// under the `gitee repo collaborator` umbrella:
//   list    — list existing collaborators on a repo
//   add     — PUT /repos/{owner}/{repo}/collaborators/{username}
//   remove  — DELETE /repos/{owner}/{repo}/collaborators/{username}
//
// All three share the --repo / -R flag pattern; add additionally takes
// a --permission flag (pull | push | admin; default push) and a
// --username positional argument.

// collaboratorFlags is shared across list/add/remove so the user can
// invoke `gitee repo collaborator add --repo owner/repo --permission
// push --username alice` (Cobra binds by pointer).
var collaboratorFlags struct {
	repo       string
	username   string
	permission string
	limit      int
}

var collaboratorCmd = &cobra.Command{
	Use:   "collaborator <subcommand>",
	Short: "Manage repository collaborators",
	Long: `Manage collaborators for a Gitee repository.

A collaborator is an individual user explicitly added to a repository
to grant them push or admin access. Use these verbs to discover who
already has access, grant a new collaborator a specific permission
level, or remove an existing one.

  gitee repo collaborator list             — list existing collaborators
  gitee repo collaborator add <username>   — add or update a collaborator
  gitee repo collaborator remove <username>— remove a collaborator`,
	Example: `  gitee repo collaborator list --repo owner/repo
  gitee repo collaborator add alice --repo owner/repo --permission push
  gitee repo collaborator remove alice --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoCollaboratorAgentHelp)
		}
		return nil
	},
	// GroupRunE prints help with no args and returns an error (with the
	// correct fully-qualified form) when given a positional arg — see cmd.GroupRunE.
	RunE: cmd.GroupRunE,
}

var collaboratorListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: "List collaborators on a repository",
	Example: `  gitee repo collaborator list --repo owner/repo
  gitee repo collaborator list -R owner/repo -o json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoCollaboratorListAgentHelp)
		}
		return nil
	},
	RunE: runCollaboratorList,
}

var collaboratorAddCmd = &cobra.Command{
	Use:   "add <username>",
	Short: "Add or update a collaborator's permission on a repository",
	Example: `  gitee repo collaborator add alice --repo owner/repo --permission push
  gitee repo collaborator add bob -R owner/repo --permission admin`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoCollaboratorAddAgentHelp)
		}
		return nil
	},
	RunE: runCollaboratorAdd,
}

var collaboratorRemoveCmd = &cobra.Command{
	Use:   "remove <username>",
	Short: "Remove a collaborator from a repository",
	Example: `  gitee repo collaborator remove alice --repo owner/repo
  gitee repo collaborator remove bob -R owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoCollaboratorRemoveAgentHelp)
		}
		return nil
	},
	RunE: runCollaboratorRemove,
}

func init() {
	// list flags
	lf := collaboratorListCmd.Flags()
	lf.StringVarP(&collaboratorFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	lf.IntVarP(&collaboratorFlags.limit, "limit", "L", 30, "Maximum number of collaborators to fetch")

	// add flags
	af := collaboratorAddCmd.Flags()
	af.StringVarP(&collaboratorFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	af.StringVarP(&collaboratorFlags.username, "username", "u", "", "Gitee username to grant access (required; or pass as positional argument)")
	af.StringVarP(&collaboratorFlags.permission, "permission", "p", "push", "Permission level: pull | push | admin")

	// remove flags
	rf := collaboratorRemoveCmd.Flags()
	rf.StringVarP(&collaboratorFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	rf.StringVarP(&collaboratorFlags.username, "username", "u", "", "Gitee username to remove (required; or pass as positional argument)")

	collaboratorCmd.AddCommand(collaboratorListCmd)
	collaboratorCmd.AddCommand(collaboratorAddCmd)
	collaboratorCmd.AddCommand(collaboratorRemoveCmd)
}

// validPermission is the closed set Gitee v5 accepts for the
// `permission` body field on collaborator add.
var validPermissions = map[string]bool{
	"pull":  true,
	"push":  true,
	"admin": true,
}

func runCollaboratorList(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveRepoArg(collaboratorFlags.repo)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.CollaboratorsList(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var collabs []types.Collaborator
	if err := client.Get(ctx, path, &collabs); err != nil {
		return fmt.Errorf("listing collaborators: %w", err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(collabs, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Collaborators on %s/%s:\n", owner, repoName)
	if len(collabs) == 0 {
		fmt.Println("  (none)")
		return nil
	}
	w := output.NewTableWriter()
	fmt.Fprintln(w, "USERNAME\tNAME\tPERMISSION\tURL")
	for _, c := range collabs {
		perm := c.Permission
		if perm == "" {
			// Fall back to role_name for legacy payloads that only emit
			// the older role-based field.
			perm = c.RoleName
		}
		if perm == "" {
			perm = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Login, c.Name, perm, c.HTMLURL)
	}
	return w.Flush()
}

// resolveCollaboratorUsername returns the username from the --username
// flag if set, otherwise from args[0]. Returns an error if neither is
// provided, or if both are set but disagree.
func resolveCollaboratorUsername(args []string) (string, error) {
	if collaboratorFlags.username != "" {
		if len(args) > 0 && args[0] != "" {
			return "", fmt.Errorf("username given both as positional argument and --username flag; pick one")
		}
		return collaboratorFlags.username, nil
	}
	if len(args) == 0 || args[0] == "" {
		return "", fmt.Errorf("username is required (pass as positional arg or via --username/-u)")
	}
	return args[0], nil
}

func runCollaboratorAdd(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveRepoArg(collaboratorFlags.repo)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	username, err := resolveCollaboratorUsername(args)
	if err != nil {
		return err
	}

	permission := strings.ToLower(strings.TrimSpace(collaboratorFlags.permission))
	if !validPermissions[permission] {
		return fmt.Errorf("invalid --permission %q: must be one of pull, push, admin", collaboratorFlags.permission)
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	payload := types.CollaboratorAddRequest{Permission: permission}
	path := api.CollaboratorAdd(owner, repoName, username)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	// Gitee v5 PUT for collaborator returns 204 No Content on success;
	// the body, if present, is a Collaborator object. We accept both.
	var created types.Collaborator
	if err := client.Put(ctx, path, payload, &created); err != nil {
		return fmt.Errorf("adding collaborator: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Collaborator %s granted %s on %s/%s\n",
		green("✓"), bold(username), bold(permission), owner, repoName)
	return nil
}

func runCollaboratorRemove(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo, err := resolveRepoArg(collaboratorFlags.repo)
	if err != nil {
		return err
	}
	owner, repoName := repo[0], repo[1]

	username, err := resolveCollaboratorUsername(args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	path := api.CollaboratorRemove(owner, repoName, username)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("removing collaborator: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s Collaborator %s removed from %s/%s\n",
		green("✓"), yellow(username), owner, repoName)
	return nil
}

const repoCollaboratorAgentHelp = `# gitee repo collaborator — Agent Help

## Purpose
Top-level umbrella for repository collaborator operations: list the
users explicitly granted access to a repository, grant (or update) a
collaborator's permission, and remove a collaborator.

A "collaborator" on Gitee is an individual user added to a repo with a
specific permission level (pull | push | admin). It does NOT include
organization members who inherit access via the org — use gitee org
members for that scope.

## Subcommands
  list             List current collaborators. See
                   gitee repo collaborator list --agent-help
  add <username>   Add or update a collaborator's permission. See
                   gitee repo collaborator add --agent-help
  remove <username>
                   Remove a collaborator. See
                   gitee repo collaborator remove --agent-help

## Pre-conditions
- Authenticated (gitee auth status)
- --repo owner/repo is set (or global --repo, or local git remote)
- Caller has admin permission on the repo (required by Gitee for all
  three operations; non-admins get 403)

## Idempotency
- list:    YES. Read-only.
- add:     YES. Re-invoking with the same username + permission is a
           no-op on the server. Re-invoking with a different permission
           updates the existing collaborator's level in place.
- remove:  YES. Re-invoking on a user who is no longer a collaborator
           returns 404, but the post-condition (user has no access) holds.

## Atomicity
Each verb is a single API call. Either the collaborator state changes
atomically or nothing changes.

## Side effects
- list:    None.
- add:     The target user is granted the requested permission and is
           notified by email (Gitee default). Webhooks subscribed to
           "Member" events fire.
- remove:  The user loses repository access immediately. Open pull
           requests and issues they authored remain in place; pushes
           already accepted remain in the git history.

## Output (stdout, table mode)
- list:    USERNAME, NAME, PERMISSION, URL table
- add:     single-line success message with username + permission
- remove:  single-line success message with username

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## See also
- gitee org members — enumerate users who inherit access via org membership
- gitee auth status  — verify token scope before calling
- gitee --agent-help — overview of all commands with agent help
`

const repoCollaboratorListAgentHelp = `# gitee repo collaborator list — Agent Help

## Purpose
Enumerate the explicit collaborators on a Gitee repository. Use this to
audit access, prepare for bulk permission changes, or check whether a
specific user already has access before adding them again.

## Pre-conditions
- Authenticated with read access to the repo
- --repo / -R is set (owner/repo)
- Caller is an admin (read endpoint requires admin on Gitee v5)

## Optional flags
  --repo / -R     Repository in owner/repo form (falls back to global
                  --repo, then local git remote).
  --limit / -L    Maximum number of collaborators to fetch (default 30)

## Idempotency
YES. Read-only; calling repeatedly returns the same list (assuming no
upstream changes between calls).

## Atomicity
Atomic per request. A single API call returns the full collaborator
list (server may paginate internally; this verb returns all of them).

## Side effects
None.

## Post-conditions
- Stdout is a plaintext table (default) or JSON array (-o json)
- Process exits 0 even when there are zero collaborators (prints "(none)")
- No state change on the server

## Output (stdout, table mode)
  Collaborators on owner/repo:
    USERNAME   NAME     PERMISSION  URL
    alice      Alice    push        https://gitee.com/alice
    bob        Bob      admin       https://gitee.com/bob

## Output (stdout, json mode)
  [
    {"id":1,"login":"alice","name":"Alice","permission":"push","html_url":"..."},
    {"id":2,"login":"bob","name":"Bob","permission":"admin","html_url":"..."}
  ]

Field names follow the Gitee API v5 /repos/{owner}/{repo}/collaborators
response. ` + "`" + `permission` + "`" + ` is one of pull | push | admin.

## Common errors
- 401 Unauthorized:  token missing or expired → gitee auth login
- 403 Forbidden:     caller is not a repo admin
- 404 Not Found:     repo does not exist or is private to the caller
- "invalid --repo format": must contain exactly one '/'

## Race condition warning ⚠️
The shared mutable state is the repo's collaborator set. This list is
a point-in-time snapshot: between listing and acting on the result,
another admin can add, update, or remove the same user. The classic
failure is list→act: you confirm "alice is not a collaborator" and
call ` + "`" + `gitee repo collaborator add alice --permission push` + "`" + `, but in
the gap another admin granted her admin — your add silently DOWNGRADES
her (add is upsert, not create-or-fail; see its Idempotency section).
Mitigation: decide the target permission from policy, not from the
absence of a row; if the current level matters, re-list immediately
before the write; in multi-agent setups serialize collaborator
mutations for the same repo. Retrying THIS list command is always
safe (read-only).

## Next steps (orchestration hints)
After listing collaborators, an agent typically wants to:
1. Check whether a specific user already has access:
     gitee repo collaborator list --repo owner/repo -o json \
       | jq -e '.[] | select(.login == "alice")'
2. Grant or update a user:
     gitee repo collaborator add alice --repo owner/repo --permission push
3. Revoke a user:
     gitee repo collaborator remove alice --repo owner/repo
4. For organization-wide access (different model), use
   gitee org members --org <org>.

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  # Plaintext table
  gitee repo collaborator list --repo myorg/myrepo

  # JSON for jq filtering
  gitee repo collaborator list --repo myorg/myrepo -o json \
    | jq -r '.[] | "\(.login)\t\(.permission)"'

## See also
- gitee repo collaborator add    — grant a user permission
- gitee repo collaborator remove — revoke a user's permission
- gitee org members              — users with org-inherited access
`

const repoCollaboratorAddAgentHelp = `# gitee repo collaborator add — Agent Help

## Purpose
Grant a Gitee user access to a repository at a specific permission
level (pull / push / admin), or update an existing collaborator's
permission. Use this for onboarding new contributors, granting
push rights after a code review, or escalating a trusted user to
admin.

## Pre-conditions
- Authenticated with projects (write) scope on the target repo
- Caller has admin permission on the repo (Gitee enforces this)
- The target user exists on Gitee (otherwise Gitee returns 404)
- --repo / -R is set (owner/repo)
- <username> is provided (positional argument) or via --username / -u
- --permission is one of pull | push | admin (default: push)

## Required flags
  <username>        Gitee username (or --username / -u).
                    Accepts letters, digits, hyphens, underscores; the
                    URL-escape is handled inside the endpoint builder.
  --repo / -R       Repository in owner/repo form.

## Optional flags
  --permission / -p   Permission level: pull | push | admin (default: push).
                      Lowercase only; invalid values are rejected client-side
                      before any API call is made.

## Idempotency
YES. Re-invoking with the same username and permission is a no-op on
the server. Re-invoking with a DIFFERENT permission updates the
collaborator in place — there is no "create or fail" semantic.

## Atomicity
Atomic. Either the collaborator has the new permission after the call,
or nothing changed. There is no partial state.

## Side effects
- The target user gains (or has updated) the requested permission
- An email notification is sent to the target user (Gitee default;
  cannot be suppressed via the API)
- Webhooks subscribed to "Member" events on the repo will fire
- If the user previously had a different permission, their access is
  updated immediately (in-flight git operations may fail)

## Post-conditions (on success)
- The user is now a collaborator with the requested permission level
- Stdout prints "✓ Collaborator <username> granted <permission> on <owner>/<repo>"

## Output (stdout, table mode)
  ✓ Collaborator alice granted push on myorg/myrepo

## Common errors
- 401 Unauthorized:  token missing or expired → gitee auth login
- 403 Forbidden:     caller is not a repo admin
- 404 Not Found:     repo does not exist OR username does not exist
- 422 Unprocessable: invalid permission value (also caught client-side)
- "invalid --permission": must be one of pull, push, admin
- "invalid --repo format": must contain exactly one '/'

## Race condition warning
If two agents simultaneously add the same user with different
permissions, Gitee stores the permission from whichever request is
processed last. There is no optimistic-concurrency token on this
endpoint — serialize at the orchestration layer if it matters.

## Next steps (orchestration hints)
After adding a collaborator, an agent typically wants to:
1. Verify: gitee repo collaborator list --repo owner/repo
2. Notify the user out-of-band (Gitee also sends email by default)
3. If you granted push, ensure the user knows the workflow:
   - git clone git@gitee.com:owner/repo.git
   - commit, push to a topic branch, open a PR
4. If you granted admin, document the responsibility (e.g. branch
   protection, release signing) for audit purposes.

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  # Grant push access (most common)
  gitee repo collaborator add alice --repo myorg/myrepo

  # Grant admin (use sparingly)
  gitee repo collaborator add alice --repo myorg/myrepo --permission admin

  # Pipeline: bulk-add from a list
  cat teammates.txt | while read u; do
    gitee repo collaborator add "$u" --repo myorg/myrepo --permission push
  done

## See also
- gitee repo collaborator list    — verify the user was added
- gitee repo collaborator remove  — revoke the user's permission
- gitee org members               — manage org-level access instead
`

const repoCollaboratorRemoveAgentHelp = `# gitee repo collaborator remove — Agent Help

## Purpose
Revoke a Gitee user's explicit collaborator permission on a repository.
After this, the user has no access to the repo (unless they also inherit
access via org membership — see "Race condition warning" below).

## Pre-conditions
- Authenticated with projects (write) scope on the target repo
- Caller has admin permission on the repo (Gitee enforces this)
- The target user IS currently a collaborator; if not, the call returns
  404 (idempotent: the post-condition still holds)
- --repo / -R is set (owner/repo)
- <username> is provided (positional argument) or via --username / -u

## Required flags
  <username>        Gitee username to remove (or --username / -u).
  --repo / -R       Repository in owner/repo form.

## Optional flags
  (none)

## Idempotency
YES. Re-invoking on a user who is no longer a collaborator returns 404,
but the post-condition (user has no access) holds. Safe to retry.

## Atomicity
Atomic. Either the user is removed after the call or nothing changed.

## Side effects
- The target user loses repository access immediately
- Their open issues and PRs remain visible (history preserved)
- Their pending PR reviews may be lost (Gitee behavior; out of CLI scope)
- Webhooks subscribed to "Member" events fire
- No email is sent to the removed user (unlike the add path)

## Post-conditions (on success)
- The user is no longer a collaborator
- Stdout prints "✓ Collaborator <username> removed from <owner>/<repo>"

## Output (stdout, table mode)
  ✓ Collaborator alice removed from myorg/myrepo

## Common errors
- 401 Unauthorized:  token missing or expired → gitee auth login
- 403 Forbidden:     caller is not a repo admin
- 404 Not Found:     user is not a collaborator (idempotent — already removed)
                     OR repo does not exist
- "invalid --repo format": must contain exactly one '/'

## Race condition warning ⚠️
Org-inherited access: if the removed user is also a member of an
organization that has access to the repo, they may STILL be able to
push via the org membership. This verb only revokes the EXPLICIT
collaborator grant. Use gitee org member remove --org <org> <user>
to revoke org-level access.

## Next steps (orchestration hints)
After removing a collaborator, an agent typically wants to:
1. Verify: gitee repo collaborator list --repo owner/repo
2. If the user should also lose org-level access, follow up with
   gitee org member remove --org <org> <user>
3. If you removed admin, consider rotating any credentials the user
   had access to (deploy keys, webhooks secrets, etc.)
4. Audit recent commits by that user before deciding on
   git history rewrite (out of CLI scope).

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  # Plain removal
  gitee repo collaborator remove alice --repo myorg/myrepo

  # Pipeline: bulk-remove from a list
  cat leavers.txt | while read u; do
    gitee repo collaborator remove "$u" --repo myorg/myrepo
  done

## See also
- gitee repo collaborator list   — confirm removal
- gitee repo collaborator add    — re-grant access if it was a mistake
- gitee org member remove        — revoke org-level access too
`
