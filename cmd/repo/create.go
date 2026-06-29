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

var createFlags struct {
	name        string
	description string
	private     bool
	autoInit    bool
	gitignore   string
	license     string
	org         string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new repository",
	Example: `  gitee repo create --name my-repo --description "My new repo"
  gitee repo create --name my-repo --private --org my-org
  gitee repo create --name my-repo --gitignore Go --license MIT`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoCreateAgentHelp)
		}
		return nil
	},
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVarP(&createFlags.name, "name", "n", "", "Repository name (required)")
	f.StringVarP(&createFlags.description, "description", "d", "", "Repository description")
	f.BoolVarP(&createFlags.private, "private", "p", false, "Make repository private")
	f.BoolVar(&createFlags.autoInit, "auto-init", true, "Initialize with README")
	f.StringVar(&createFlags.gitignore, "gitignore", "", "Gitignore template name")
	f.StringVar(&createFlags.license, "license", "", "License template name")
	f.StringVar(&createFlags.org, "org", "", "Organization to create repo under")
	_ = createCmd.MarkFlagRequired("name")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.RepositoryCreateRequest{
		Name:              createFlags.name,
		Description:       createFlags.description,
		Private:           createFlags.private,
		AutoInit:          createFlags.autoInit,
		GitignoreTemplate: createFlags.gitignore,
		LicenseTemplate:   createFlags.license,
		Organization:      createFlags.org,
	}

	var path string
	if createFlags.org != "" {
		path = api.ReposCreateForOrg(createFlags.org)
	} else {
		path = api.ReposCreate()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Post(ctx, path, req, &repo); err != nil {
		return fmt.Errorf("creating repository: %w", err)
	}

	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Repository %s created\n", green("✓"), bold(repo.FullName))
	fmt.Printf("  URL: %s\n", color.CyanString(repo.HTMLURL))
	return nil
}

const repoCreateAgentHelp = `# gitee repo create — Agent Help

## Purpose
Create a new repository on Gitee under the authenticated user
or under an organization. Use this when an agent needs to spin up
a fresh project on behalf of the user (e.g. after a scaffolding step
or for an isolated worktree experiment).

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- --name is non-empty and unique within the target scope
  (enforced by Cobra MarkFlagRequired)
- If --org is set: the org exists and the token has create-repo permission

## Required flags
  --name / -n    Repository name (required, non-empty, must be unique in scope)

## Optional flags
  --description / -d  Repository description
  --private / -p      Make the repository private (default: public)
  --auto-init         Initialize with a README.md (default: true)
  --gitignore         Gitignore template name (e.g. "Go", "Node")
  --license           License template name (e.g. "MIT", "Apache-2.0")
  --org               Organization login to create the repo under (default: own account)

## Idempotency
NOT idempotent. Re-running with the same --name in the same scope
returns 422 ("name has already been taken"). The Gitee API has no
"create or update" semantic. To rename an existing repo use
` + "`" + `gitee repo rename` + "`" + ` or ` + "`" + `gitee repo edit --name ...` + "`" + `.

## Atomicity
Atomic. The repo is created in one POST; either it appears or it doesn't.
There is no partial state.

## Side effects
- A new repository is created on Gitee under the token owner (or --org)
- If --auto-init is true: an initial commit with README.md is created,
  and a default branch (usually ` + "`" + `master` + "`" + `) is set
- Webhooks subscribed to "Repository" events on the org may fire
- The new repo is immediately visible at its HTML URL
- Email/in-app notification may go to watchers of the scope

## Post-conditions (on success)
- Stdout reports "✓ Repository owner/name created" plus the HTML URL
- The repo exists; one git remote URL is ` + "`" + `https://gitee.com/owner/name.git` + "`" + `
- Initial empty branch exists if --auto-init was true (else empty repo)

## Common errors
- 401 Unauthorized:  token expired or invalid   →  gitee auth login
- 403 Forbidden:     token lacks create scope, or --org forbidden
- 404 Not Found:     --org does not exist or is not visible to token
- 422 Unprocessable: name already taken, invalid characters, etc.
- Network/timeout:   transient; safe to retry (but check the URL didn't fire twice)

## Next steps (orchestration hints)
After a successful repo create, an agent typically wants to:
1. Record the FullName and URL from stdout
2. Local working setup: ` + "`" + `gitee repo clone owner/name` + "`" + ` to pull
   the initial commit (or fetch an empty repo on --auto-init=false)
3. Inspect details: ` + "`" + `gitee repo view owner/name` + "`" + ` for verification
4. Tune settings: ` + "`" + `gitee repo edit owner/name --description "..."` + "`" + `
5. Push the first real commit:
   ` + "`" + `git -C name remote add origin <html_url>` + "`" + ` then ` + "`" + `git push -u origin main` + "`" + `
6. Create the first issue tracker: ` + "`" + `gitee issue create --repo owner/name ...` + "`" + `

## OAuth scope required
projects (write) — plus organization membership with create permission
if --org is used.

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Minimal: a fresh public repo with a README
  gitee repo create --name sandbox --description "Throwaway test"

  # Private repo inside an org, with Go gitignore and MIT license
  gitee repo create --name billing-svc --private \\
    --org acme --gitignore Go --license MIT \\
    --description "Acme billing service (Go)"

  # Empty repo (no README) for an import from another host
  gitee repo create --name migrated --auto-init=false

## See also
- ` + "`" + `gitee repo view` + "`" + `             — inspect the new repo
- ` + "`" + `gitee repo edit` + "`" + `             — change settings post-create
- ` + "`" + `gitee repo rename` + "`" + `          — rename an existing repo
- ` + "`" + `gitee repo clone` + "`" + `            — clone the new repo locally
- ` + "`" + `gitee repo delete` + "`" + `           — remove an unwanted repo
- ` + "`" + `gitee --agent-help` + "`" + `          — overview of all commands with agent help
`
