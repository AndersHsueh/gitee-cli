package notification

import (
	"fmt"

	"gitee-cli/cmd"
	"gitee-cli/internal/handler"

	"github.com/spf13/cobra"
)

var readFlags struct {
	all    bool
	unread bool
	force  bool
}

var readCmd = &cobra.Command{
	Use:   "read [<id> | --all]",
	Short: "Mark notification threads as read",
	Long: `Mark notification threads as read. By default a single thread
id is required; pass ` + "`" + `--all` + "`" + ` to mark every thread as read at once.

The verb is reversible: pass ` + "`" + `--unread` + "`" + ` to flip a thread (or all
threads) back to unread, e.g. to re-surface something you want to
look at later.`,
	Example: `  gitee notification read 1234
  gitee notification read 1234 --force
  gitee notification read --all
  gitee notification read --all --unread      # bulk mark-as-unread
  gitee notification read 1234 --unread        # flip one back`,
	Args: func(c *cobra.Command, args []string) error {
		// Either an <id> positional OR --all; never both, never neither.
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 positional argument (notification id)")
		}
		return nil
	},
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(notificationReadAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		threadID := ""
		if len(args) >= 1 {
			threadID = args[0]
		}
		if threadID == "" && !readFlags.all {
			return fmt.Errorf("notification id is required (positional argument) or use --all")
		}
		if threadID != "" && readFlags.all {
			return fmt.Errorf("--all and a positional id are mutually exclusive")
		}
		return handler.NotificationRead(threadID, readFlags.all, readFlags.unread, readFlags.force)
	},
}

func init() {
	f := readCmd.Flags()
	f.BoolVar(&readFlags.all, "all", false, "Mark ALL threads as read (no id required)")
	f.BoolVar(&readFlags.unread, "unread", false, "Reverse direction: mark as UNREAD instead of read")
	f.BoolVarP(&readFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

const notificationReadAgentHelp = `# gitee notification read — Agent Help

## Purpose
Mark a single notification thread (or all of them) as read. The verb
is reversible via ` + "`" + `--unread` + "`" + `, so it can also re-surface a thread
that you want to look at later.

## Pre-conditions
- Authenticated with ` + "`" + `notifications` + "`" + ` scope (write)
- For single-thread mode: the id exists in the inbox (returns 404
  otherwise). Find the id via ` + "`" + `gitee notification list` + "`" + `.

## Required arguments
  Either an ` + "`" + `<id>` + "`" + ` positional OR ` + "`" + `--all` + "`" + ` — the two are mutually exclusive.

## Optional flags
  --all         Mark ALL threads as read (PUT /notifications/threads)
  --unread      Flip state: mark as UNREAD instead of read
  --force / -f  Skip the y/N confirmation prompt (use in scripts / agents)

## Idempotency
PARTIAL.
- mark-as-read:  re-running is a server-side no-op; same thread stays
  read.
- mark-as-unread: re-running is idempotent (sets unread=true again).
- mixing directions (read → unread → read) leaves the thread in the
  last-requested state.

## Atomicity
- single-thread: Atomic. One PUT, one row updated.
- bulk (--all):   NOT atomic. Gitee processes bulk mark-read in a single
  PUT, but the call covers all matching threads at once.

## Side effects
- The thread (or every thread) moves between read and unread
- The notification counter (` + "`" + `notification count` + "`" + `) updates to reflect
  the new unread count immediately on the next call
- Webhooks subscribed to user activity events may fire

## Post-conditions (on success)
- Stdout prints ` + "`" + `✓ Notification <id> marked as read` + "`" + ` (or ` + "`" + `UNREAD` + "`" + `)
- For ` + "`" + `--all` + "`" + `: ` + "`" + `✓ All notifications marked as read` + "`" + `

## Output (stdout)
  ✓ Notification 1234 marked as read
  ✓ All notifications marked as read
  ✓ Notification 1234 marked as UNREAD    # with --unread

## Common errors
- 404 Not Found:  id doesn't exist in this user's inbox
- 401 Unauthorized: token missing or expired
- 403 Forbidden:  token lacks ` + "`" + `notifications` + "`" + ` scope

## Race condition warning ⚠️
The read/unread flag on each thread is shared, mutable server state.
Concurrent marks from two agents are last-writer-wins, but because
each direction is idempotent (see Idempotency), same-direction races
converge — the real hazard is a conflicting ` + "`" + `--unread` + "`" + ` from
elsewhere silently re-surfacing a thread you just cleared. The bigger
trap is ` + "`" + `--all` + "`" + `: the PUT covers every thread that exists WHEN IT
LANDS, including threads that arrived after your last
` + "`" + `gitee notification list` + "`" + ` — you can mark-as-read notifications you
have never seen. Prefer explicit ids from a fresh list for surgical
triage, reserve ` + "`" + `--all` + "`" + ` for "I truly want a clean inbox", and verify
with ` + "`" + `gitee notification count` + "`" + ` afterwards.

## OAuth scope required
notifications (write)

## Rate limit
1000 req/hour. Bulk (` + "`" + `--all` + "`" + `) counts as 1 request.

## Example
  # 1. Discover what's unread
  gitee notification list --unread

  # 2. Mark the oldest as read (use --force to skip the prompt)
  gitee notification read 1234 --force

  # 3. Triage loop: clear everything
  gitee notification read --all --force

  # 4. Oops, re-surface one you want to come back to
  gitee notification read 1234 --unread

## See also
- ` + "`" + `gitee notification list` + "`" + `  — discover ids and current state
- ` + "`" + `gitee notification count` + "`" + ` — confirm count is back to 0
`