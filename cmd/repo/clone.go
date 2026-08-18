package repo

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gitee-cli/cmd"

	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <owner>/<repo> [directory]",
	Short: "Clone a repository",
	Example: `  gitee repo clone owner/repo
  gitee repo clone owner/repo my-dir
  gitee repo clone owner/repo --depth 1`,
	Args: cmd.AgentHelpAwareArgs(cobra.MinimumNArgs(1)),
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(repoCloneAgentHelp)
		}
		return nil
	},
	RunE: runClone,
}

func runClone(cobraCmd *cobra.Command, args []string) error {
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	repo := args[0]
	var targetDir string
	if len(args) > 1 {
		targetDir = args[1]
	} else {
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) == 2 {
			targetDir = parts[1]
		} else {
			targetDir = repo
		}
	}

	gitURL := buildGitURL(repo)

	cmdArgs := []string{"clone", gitURL}
	if targetDir != "" {
		cmdArgs = append(cmdArgs, targetDir)
	}

	execCmd := exec.Command("git", cmdArgs...)
	execCmd.Stdin = os.Stdin
	execCmd.Stderr = os.Stderr
	// NOTE: stdout is left default so git's progress output (e.g. for HTTPS
	// auth prompts, or shallow progress) is preserved.
	return execCmd.Run()
}

func buildGitURL(repo string) string {
	return fmt.Sprintf("https://%s/%s", cmd.GlobalHostname, repo)
}

const repoCloneAgentHelp = `# gitee repo clone — Agent Help

## Purpose
Clone a Gitee repository to the local working directory by shelling out
to ` + "`" + `git clone` + "`" + `. Use this as the standard "fetch the repo"
step at the start of any agent workflow that needs local git history.

## Pre-conditions
- A positional argument must be supplied in the form ` + "`" + `owner/repo` + "`" + `
  (Cobra enforces MinimumNArgs(1)). No flag-based form is supported here;
  use ` + "`" + `gitee repo view --repo owner/repo` + "`" + ` if you only need metadata.
- ` + "`" + `git` + "`" + ` is installed and on PATH (this command execs ` + "`" + `git clone` + "`" + `)
- The repo URL ` + "`" + `https://<hostname>/<owner>/<repo>` + "`" + ` is reachable from
  this host, OR the user has the right git credentials cached for
  pushes (HTTPS basic, or SSH key — depending on the remote URL)

## Required arguments
  <owner>/<repo>     Positional, required. Format ` + "`" + `owner/repo` + "`" + `.

## Optional flags
  (none — this verb has no first-class flags. Forward advanced git options
   by editing ~/.gitconfig or by wrapping this command in your own shell.)
  [directory]        Optional second positional; the directory to clone into.
                     Defaults to the trailing path segment of ` + "`" + `<owner>/<repo>` + "`" + `,
                     i.e. ` + "`" + `repo` + "`" + ` for ` + "`" + `owner/repo` + "`" + `.

## Idempotency
NOT idempotent at the filesystem level. If the destination directory
already exists and is non-empty, ` + "`" + `git clone` + "`" + ` will FAIL with
"destination path already exists and is not an empty directory".
Pre-check with ` + "`" + `test -d <dir> && ! rmdir <dir> 2>/dev/null` + "`" + `
or pass ` + "`" + `[directory]` + "`" + ` explicitly to control the target.

## Atomicity
NOT atomic. ` + "`" + `git clone` + "`" + ` may leave a partially-initialized directory
on disk if interrupted mid-transfer. The downstream "next steps" assume
a clean, completed clone.

## Side effects
- A new directory ` + "`" + `<owner>/<repo>` + "`" + ` (or the supplied ` + "`" + `[directory]` + "`" + `) is created
  in CWD with ` + "`" + `.git/` + "`" + ` and a working tree
- The origin remote is configured to the HTTPS URL:
    ` + "`" + `origin\thttps://<hostname>/<owner>/<repo>` + "`" + `
  No SSH remote is added; for SSH, run ` + "`" + `git remote set-url origin git@gitee.com:<owner>/<repo>` + "`" + `
- Working tree contains the source's default branch (HEAD only; other
  branches are tracked as remotes/origin/*)
- A ` + "`" + `HEAD` + "`" + ` ref points to the default branch tip; if you need
  a specific branch, ` + "`" + `git -C <dir> checkout <branch>` + "`" + ` next

## Post-conditions (on success)
- Exit status 0; the repo is cloned into CWD/<dir>
- Auth credentials (if required) come from the user's git/SSH config;
  this command does not pass any token to git

## Common errors
- "destination path already exists": dir is non-empty → pick another name
- "could not read Username": HTTPS auth prompt failed → ensure credentials helper
- "Repository not found": Gitee 404 on the clone URL → verify owner/repo spelling,
                          visibility, network access
- exec fails (missing git): install git or fix PATH

## Race condition warning ⚠️
Two pieces of shared mutable state. (1) LOCAL: the destination
directory. Two concurrent clones into the same [directory] interleave
` + "`" + `git clone` + "`" + `'s non-atomic writes and typically one fails with
"destination path already exists" — or both corrupt the target. Give
each agent its own directory. (2) REMOTE: the clone is a SNAPSHOT of
the repo at fetch time. If the source repo is pushed to, renamed, made
private, or deleted while (or after) the clone runs, your local copy
is stale or the clone fails partway with a partial directory on disk
(see Atomicity). On retry after a failed clone, remove the partial
target directory first — ` + "`" + `git clone` + "`" + ` will not resume into a
non-empty dir. If freshness matters, ` + "`" + `git -C <dir> fetch origin` + "`" + `
after clone before acting on the tree.

## Next steps (orchestration hints)
After a successful clone, an agent typically wants to:
1. ` + "`" + `cd <dir>` + "`" + ` (or ` + "`" + `git -C <dir> ...` + "`" + ` in non-shell contexts)
2. Inspect default branch and upstream:
   ` + "`" + `git -C <dir> symbolic-ref --short HEAD` + "`" + `
   ` + "`" + `git -C <dir> remote -v` + "`" + `
3. To enumerate branches without checking them out:
   ` + "`" + `git -C <dir> branch -r` + "`" + ` (or ` + "`" + `gitee repo branches --repo owner/repo` + "`" + `)
4. To work on a specific branch:
   ` + "`" + `git -C <dir> checkout <branch>` + "`" + ` then commit and push normally
5. To open a PR after a local change:
   ` + "`" + `gitee pr create --repo owner/repo --source <branch> --target main ...` + "`" + `

## OAuth scope required
None — clone is unauthenticated for public repos; private clones require
the user to have SSH key access or HTTPS PAT in their git credentials.

## Rate limit
Subject to git transport limits (bandwidth / concurrent connections),
NOT to Gitee API rate-limit quotas. The accompanying API calls in
later steps (branch list, PR create, etc.) ARE quota-bound (1000/hr).

## Example
  # Bare clone
  gitee repo clone myorg/billing-svc

  # Clone into a specific directory
  gitee repo clone myorg/billing-svc ~/work/billing-svc

  # Shallow clone (forwarded via wrapper)
  git clone --depth 1 \$(gitee host)/myorg/billing-svc /tmp/billing-shallow

  # Non-shell / agent pattern: capture errors explicitly
  if ! gitee repo clone myorg/billing-svc /tmp/billing 2>/tmp/clone.err; then
    cat /tmp/clone.err
    exit 1
  fi

## See also
- ` + "`" + `gitee repo view` + "`" + `           — inspect repo metadata first
- ` + "`" + `gitee repo branches` + "`" + `      — list all branches via API
- ` + "`" + `gitee repo fork` + "`" + `          — fork before cloning for writes
- ` + "`" + `gitee repo browse` + "`" + `        — open in web browser (no local clone)
- ` + "`" + `gitee repo create` + "`" + `       — create a new repo to clone from
- ` + "`" + `gitee --agent-help` + "`" + `       — overview of all commands with agent help
`
