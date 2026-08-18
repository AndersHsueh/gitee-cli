package repo

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"gitee-cli/cmd"
	"gitee-cli/internal/utils"

	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse [<owner>/<repo>]",
	Short: "Open repository in browser",
	Example: `  gitee repo browse owner/repo
  gitee repo browse --repo owner/repo`,
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(repoBrowseAgentHelp)
		}
		return nil
	},
	RunE: runBrowse,
}

func runBrowse(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	// --agent-help handling is in PreRunE (runs before required-flag validation).

	var owner, repoName string
	var err error

	if len(args) > 0 {
		owner, repoName = utils.ParseOwnerRepo(args[0])
	} else {
		owner, repoName, err = resolveRepo("", "", nil)
		if err != nil {
			return err
		}
	}

	url := fmt.Sprintf("https://%s/%s/%s", rt.Hostname, owner, repoName)

	// Try to open browser
	browser := os.Getenv("BROWSER")
	if browser == "" {
		switch runtime.GOOS {
		case "darwin":
			browser = "open"
		case "linux":
			browser = "xdg-open"
		default:
			browser = "open"
		}
	}

	execCmd := exec.Command(browser, url)
	execCmd.Stdin = os.Stdin
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}

const repoBrowseAgentHelp = `# gitee repo browse — Agent Help

## Purpose
Open a Gitee repository in the user's default web browser. Useful as
a UI escape hatch when an agent (or a human) wants to look at something
the CLI doesn't render well — release pages, contributor graphs, comment
threads with images, web-only settings, etc.

## Pre-conditions
- A target must resolve to ` + "`" + `owner/repo` + "`" + ` via one of:
    - positional argument:        ` + "`" + `gitee repo browse owner/repo` + "`" + `
    - --repo / -R flag (ignored here; see below)
    - global --repo fallback, or local git remote fallback
  If none of these resolve, the command exits with an error.
- A browser launcher is on PATH. Detection order:
    1. ` + "`" + `$BROWSER` + "`" + ` environment variable (used verbatim)
    2. macOS ` + "`" + `open` + "`" + `, Linux ` + "`" + `xdg-open` + "`" + `, other OSes fall back to ` + "`" + `open` + "`" + `
- The local environment is interactive (this command spawns a GUI process).
  Agents running headless should NOT call this; use ` + "`" + `gitee repo view` + "`" + `.

## Optional flags
  (none — this verb has no first-class flags. Resolution chain:
   positional > --repo > global --repo > git remote.)

## Idempotency
YES. Each invocation just opens a browser tab/window to the same URL.
Calling it N times opens N tabs/windows.

## Atomicity
Not applicable. The command is ` + "`" + `exec(browser, url)` + "`" + ` — fire-and-forget;
the launching process exits as soon as the child starts.

## Side effects
- Spawns the user's default browser (or ` + "`" + `$BROWSER` + "`" + ` if set) with a new
  window/tab pointed at ` + "`" + `https://<hostname>/<owner>/<repo>` + "`" + `
- On some systems the first launch may briefly block for ~100ms while
  the browser registers a new tab
- No network calls are made by the CLI itself; the URL only resolves
  when the browser actually loads it

## Post-conditions (on success)
- Exit status 0
- A browser window/tab opens (visible only on a desktop session)
- No stdout/stderr output to parse

## Common errors
- "exec: <browser>: executable file not found": set ` + "`" + `$BROWSER` + "`" + ` or install one
  (e.g. on Linux: ` + "`" + `apt install xdg-utils` + "`" + `)
- "could not resolve owner/repo": none of the resolution sources fired
  → pass ` + "`" + `owner/repo` + "`" + ` positionally, or ` + "`" + `cd` + "`" + ` into a git clone first
- Headless / SSH session: the GUI browser has nothing to attach to.
  Use ` + "`" + `gitee repo view` + "`" + ` for terminal-friendly output instead.

## Race condition warning ⚠️
This command makes no API call and shares no server-side state — the
only mutable state is the user's desktop browser session. Two races to
know about: (1) the repo can be renamed, made private, or deleted
between when you resolve owner/repo and when the browser actually
loads the URL, so the tab may land on a 404 page even though this
command exited 0; (2) concurrent invocations each spawn a new tab, so
N agents calling browse in a loop will flood the user's browser with
N tabs — serialize at the orchestration layer. Because the command is
fire-and-forget, a "successful" exit says nothing about what the user
eventually sees; if the outcome matters, verify the repo still exists
first with ` + "`" + `gitee repo view owner/repo` + "`" + `.

## Next steps (orchestration hints)
After ` + "`" + `gitee repo browse` + "`" + ` hands off the GUI to the user, an agent
typically wants to:
1. Switch to non-interactive operations:
   ` + "`" + `gitee repo view owner/repo -o json` + "`" + ` for structured details
2. To drill into issues via UI: follow the issues tab URL
3. To drill into PRs via UI: follow the pulls tab URL
4. For further local work: ` + "`" + `gitee repo clone owner/repo` + "`" + `

## OAuth scope required
None. The browser does the auth via the user's existing web session.

## Rate limit
None on the CLI side. Browser tab fetches the HTML page (1 GET, cached),
not the API — does NOT consume the 1000/hr quota.

## Example
  # Open via positional
  gitee repo browse myorg/billing-svc

  # Open via global flag set elsewhere
  gitee --repo myorg/billing-svc repo browse

  # Use a custom browser (e.g. inside containers)
  BROWSER=firefox gitee repo browse myorg/billing-svc

  # Agent pattern: don't call this at all in headless mode
  if [ -n "$DISPLAY" ] || [ "$(uname)" = "Darwin" ]; then
    gitee repo browse myorg/billing-svc &
  else
    gitee repo view --repo myorg/billing-svc -o json
  fi

## See also
- ` + "`" + `gitee repo view` + "`" + `            — same metadata, terminal-friendly
- ` + "`" + `gitee repo clone` + "`" + `           — fetch the code locally
- ` + "`" + `gitee issue browse` + "`" + `        — browse an issue in browser (if available)
- ` + "`" + `gitee pr browse` + "`" + `           — browse a PR in browser (if available)
- ` + "`" + `gitee --agent-help` + "`" + `        — overview of all commands with agent help
`
