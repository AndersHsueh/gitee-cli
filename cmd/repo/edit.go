package repo

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var editFlags struct {
	owner         string
	repo          string
	name          string
	description   string
	privateValue  bool
	defaultBranch string
}

var editCmd = &cobra.Command{
	Use:   "edit [<owner>/<repo>]",
	Short: "Edit repository settings",
	Example: `  gitee repo edit owner/repo --description "New description"
  gitee repo edit --repo owner/repo --private`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoEditAgentHelp)
		}
		return nil
	},
	RunE: runEdit,
}

func init() {
	f := editCmd.Flags()
	f.StringVar(&editFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&editFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVar(&editFlags.name, "name", "", "New repository name")
	f.StringVarP(&editFlags.description, "description", "d", "", "Repository description")
	f.BoolVar(&editFlags.privateValue, "private", false, "Make repository private")
	f.StringVar(&editFlags.defaultBranch, "default-branch", "", "Default branch name")
}

func runEdit(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	owner, repoName, err := resolveRepo(editFlags.owner, editFlags.repo, args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.RepositoryEditRequest{
		Name:          editFlags.name,
		Description:   editFlags.description,
		DefaultBranch: editFlags.defaultBranch,
	}
	if cobraCmd.Flags().Changed("private") {
		req.Private = &editFlags.privateValue
	}

	path := api.ReposEdit(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Patch(ctx, path, req, &repo); err != nil {
		return fmt.Errorf("editing repository: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Repository %s updated\n", green("✓"), repo.FullName)
	return nil
}

const repoEditAgentHelp = `# gitee repo edit — Agent Help

## Purpose
Patch a repository's settings: rename, change description, toggle
visibility (public/private), or change the default branch. Use this when
an agent needs to update metadata on an existing repo without going
through the web UI.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- Token has admin (write) permission on the target repo
- One of the following must resolve to owner/repo:
    - positional argument:        ` + "`" + `gitee repo edit owner/repo` + "`" + `
    - --repo / -R flag:           ` + "`" + `--repo owner/repo` + "`" + `
    - split --owner + --repo:     ` + "`" + `--owner owner --repo name` + "`" + `
- At least one mutable flag is provided (--name, --description,
  --private, --default-branch); otherwise the request is a no-op

## Optional flags
  --owner             Repository owner (when --repo is just the bare name)
  --repo / -R         Repository name, OR owner/name; overrides positional arg
  --name              New repository name (rename)
  --description / -d  New repository description
  --private           Make the repository private (true) or public (false);
                      only applied if the flag was explicitly set
  --default-branch    New default branch name (branch must already exist)

## Idempotency
YES, per-field. Re-running with the same values produces the same final
state. PATCH semantics mean unspecified fields are left unchanged, so
calling repeatedly with the same input is safe.

## Atomicity
Atomic per request. Either all changed fields update, or none do.

## Side effects
- Repo name, description, visibility, or default branch is updated on Gitee
- If --default-branch points to a branch that does NOT exist on the
  server, the API returns 422 — no fallback
- If --private is toggled: webhooks subscribed to repo events may fire;
  collaborators may lose access (re-evaluate ACL after toggling)
- Renaming (--name) changes the HTML URL and git remote URL. Existing
  clones need: ` + "`" + `git remote set-url origin <new-url>` + "`" + `
- Webhook callbacks subscribed to "Repository" events may fire

## Post-conditions (on success)
- Stdout reports "✓ Repository owner/name updated"
- All provided fields reflect new values on the server
- Omitted fields are unchanged (PATCH semantics)

## Common errors
- 401 Unauthorized:  token expired or invalid   →  gitee auth login
- 403 Forbidden:     token lacks admin scope, or visibility downgrade is blocked
- 404 Not Found:     owner/name doesn't exist
- 422 Unprocessable: --default-branch doesn't exist; name conflict (already taken);
                     or invalid characters in --name
- Network/timeout:   transient; safe to retry (PATCH is idempotent for the same payload)

## Next steps (orchestration hints)
After a successful edit, an agent typically wants to:
1. Verify the change: ` + "`" + `gitee repo view owner/name -o json | jq` + "`" + `
2. After a rename: update local git remotes:
   ` + "`" + `git remote set-url origin https://gitee.com/<owner>/<new-name>.git` + "`" + `
3. After a private toggle: refresh collaborator tokens / CI caches
4. To change the default branch's HEAD commit, use a separate workflow
   (push a commit, or ` + "`" + `gitee repo ...` + "`" + ` does not move refs)
5. To fully remove the repo: ` + "`" + `gitee repo delete owner/name --yes` + "`" + `

## OAuth scope required
projects (admin) — for --private and --default-branch
projects (write) — for --name and --description

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Change description
  gitee repo edit myorg/svc --description "Auth gateway (Go 1.23)"

  # Rename a repo (downstream clones need a remote URL update)
  gitee repo edit myorg/old-name --name new-name

  # Toggle to private (must explicitly set the flag)
  gitee repo edit myorg/svc --private

  # Switch default branch (branch must already exist)
  gitee repo edit myorg/svc --default-branch main

  # Verify the patch
  gitee repo view --repo myorg/new-name -o json | jq '.full_name,.private'

## See also
- ` + "`" + `gitee repo view` + "`" + `             — inspect the repo before editing
- ` + "`" + `gitee repo rename` + "`" + `          — dedicated rename verb (alias of edit --name)
- ` + "`" + `gitee repo archive` + "`" + `         — flip the archived flag
- ` + "`" + `gitee repo delete` + "`" + `          — remove the repo entirely
- ` + "`" + `gitee --agent-help` + "`" + `          — overview of all commands with agent help
`
