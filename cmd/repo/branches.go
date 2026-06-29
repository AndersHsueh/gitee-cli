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

	"github.com/spf13/cobra"
)

var branchesFlags struct {
	repo string
}

var branchesCmd = &cobra.Command{
	Use:     "branches",
	Short:   "List branches in a repository",
	Example: `  gitee repo branches --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoBranchesAgentHelp)
		}
		return nil
	},
	RunE: runBranches,
}

func init() {
	f := branchesCmd.Flags()
	f.StringVarP(&branchesFlags.repo, "repo", "R", "", "Repository (owner/repo)")
}

func runBranches(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo := branchesFlags.repo
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	if repo == "" {
		return fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid --repo format, expected owner/repo")
	}
	owner, repoName := parts[0], parts[1]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReposBranches(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var branches []types.Branch
	if err := client.Get(ctx, path, &branches); err != nil {
		return fmt.Errorf("listing branches: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(branches, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Branches in %s/%s:\n", owner, repoName)
	for _, b := range branches {
		fmt.Printf("  %s", b.Name)
		if b.Protected {
			fmt.Printf(" (protected)")
		}
		fmt.Println()
	}
	return nil
}

const repoBranchesAgentHelp = `# gitee repo branches — Agent Help

## Purpose
List all branches in a Gitee repository. Use this as a pre-flight for
any operation that requires picking a source/target branch, such as
opening a PR or switching local checkouts. The listing is read from
the Gitee API rather than a local clone, so it reflects the canonical
remote state.

## Pre-conditions
- Authenticated: token present in config (~/.gitee-cli/settings.json),
  env var (GITEE_TOKEN), or --token flag. Verify with: gitee auth status
- The target repo exists and is visible to the token
- --repo is supplied in the form ` + "`" + `owner/repo` + "`" + ` (or ` + "`" + `--repo` + "`" + `
  is set globally; falls back to the local git remote). Validated locally;
  missing or invalid format returns an error before any API call.

## Optional flags
  --repo / -R     Repository in ` + "`" + `owner/repo` + "`" + ` form (falls back to ` + "`" + `--repo` + "`" + `
                  on the global flag, then to git remote).

## Idempotency
YES. Read-only; calling repeatedly yields the same list (assuming no
upstream branch create/delete between calls). Safe to retry.

## Atomicity
Atomic per request. A single API call returns the full branch list
(server may paginate internally; this verb returns all of them).

## Side effects
None. Read-only.

## Post-conditions
- Stdout is a plaintext list (default) or JSON array (-o json)
- Process exits 0 even when the repo has zero branches
  (prints "Branches in owner/repo:" with no entries)
- No state change on the server, no local cache write

## Output (stdout, table mode)
  Branches in owner/repo:
    main
    develop
    feature/login (protected)
    release/1.0

A trailing " (protected)" annotation marks branches the API reports
as protected; the rest of the row is the bare branch name.

## Output (stdout, json mode)
  [
    {
      "name": "main",
      "protected": false
    },
    {
      "name": "feature/login",
      "protected": true
    }
  ]

Field names follow the Gitee API v5 /repos/{owner}/{repo}/branches
response. Use ` + "`" + `.name` + "`" + ` as input to other commands (PR create,
local checkout, etc.).

## Common errors
- 401 Unauthorized:  token expired or invalid   →  gitee auth login
- 403 Forbidden:     repo is private and token cannot read it
- 404 Not Found:     owner/name doesn't exist; check spelling
- "invalid --repo format": must contain exactly one ` + "`" + `/` + "`" + `
- Network/timeout:   transient; safe to retry

## Next steps (orchestration hints)
After listing branches, an agent typically wants to:
1. Pick a source/target pair (e.g. ` + "`" + `feature/login` + "`" + ` → ` + "`" + `main` + "`" + `)
2. Open a PR between them:
   ` + "`" + `gitee pr create --repo owner/repo --source feature/login --target main ...` + "`" + `
3. To see a specific branch's commits:
   ` + "`" + `gitee pr list --repo owner/repo --head <branch>` + "`" + `
   or ` + "`" + `git -C <local-clone> log <branch>` + "`" + ` for local history
4. If ` + "`" + `Protected` + "`" + ` is true: PRs into that branch must satisfy the
   repo's branch-protection rules (e.g. review approvals). Inspect via
   ` + "`" + `gitee repo view owner/repo -o json` + "`" + ` for canonical settings.
5. To operate on a branch locally without cloning:
   re-fetch with ` + "`" + `gitee repo clone owner/repo` + "`" + ` first.

## OAuth scope required
projects (read)

## Rate limit
Gitee API v5: 1000 requests/hour for authenticated users.
Each call to this command counts as 1 request.

## Example
  # Plainlist
  gitee repo branches --repo myorg/billing-svc

  # JSON for jq filtering
  gitee repo branches --repo myorg/billing-svc -o json \\
    | jq -r '.[] | select(.protected).name'

  # Pick a source branch and open a PR
  HEAD=$(gitee repo branches --repo myorg/billing-svc -o json \\
    | jq -r '.[0].name')
  gitee pr create --repo myorg/billing-svc --source "$HEAD" --target main \\
    --title "Sync to main"

## See also
- ` + "`" + `gitee repo view` + "`" + `            — see default branch + repo metadata
- ` + "`" + `gitee pr create` + "`" + `            — open a PR between two listed branches
- ` + "`" + `gitee pr list` + "`" + `              — see open PRs (useful with --head)
- ` + "`" + `gitee --agent-help` + "`" + `         — overview of all commands with agent help
`
