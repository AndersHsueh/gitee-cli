package gpgkey

import (
	"context"
	"fmt"
	"os"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var deleteFlags struct {
	id    int64
	force bool
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a GPG public key by id",
	Example: `  gitee gpg-key delete 7
  gitee gpg-key delete 7 --force`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(gpgKeyDeleteAgentHelp)
			os.Exit(0)
		}
		return nil
	},
	RunE: runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.Int64VarP(&deleteFlags.id, "id", "i", 0, "GPG key id (alternative to positional)")
	f.BoolVarP(&deleteFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

func runDelete(cobraCmd *cobra.Command, args []string) error {
	id, err := utils.ParseInt64Arg(deleteFlags.id, args, "GPG key")
	if err != nil {
		return err
	}
	if id == 0 {
		return fmt.Errorf("GPG key id is required (positional or --id)")
	}

	if !deleteFlags.force {
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Delete GPG key id=%d?", color.YellowString("?"), id),
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, api.GPGKeyDelete(id)); err != nil {
		return fmt.Errorf("deleting GPG key %d: %w", id, err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s GPG key %s deleted\n", green("✓"), bold(fmt.Sprintf("id=%d", id)))
	return nil
}

const gpgKeyDeleteAgentHelp = `# gitee gpg-key delete — Agent Help

## Purpose
Remove a GPG public key from the current user's Gitee account. After
this, commits signed by the matching private key will no longer show
as "verified" on Gitee (existing signed commits are NOT modified).

## Pre-conditions
- Authenticated with write scope
- The id corresponds to an existing key (returns 404 otherwise)

## Required arguments
  <id>     GPG key id (find via ` + "`" + `gitee gpg-key list` + "`" + `)

## Optional flags
  --id / -i      Alternative to positional
  --force / -f   Skip the y/N confirmation prompt

## Idempotency
NOT idempotent. Re-running on a deleted key returns 404. There is no
"create or update" path on delete; use ` + "`" + `gitee gpg-key add` + "`" + ` to register
a new one.

## Atomicity
Atomic. Either the key is gone, or no change.

## Side effects
- The key can no longer verify new commits
- Existing signed commits keep their signature but show as "unverified"
- Webhooks subscribed to user events may fire

## Post-conditions (on success)
- The key no longer appears in ` + "`" + `gitee gpg-key list` + "`" + `
- Local git config (` + "`" + `user.signingkey` + "`" + `) is UNCHANGED — your local
  setup still uses the key; only Gitee no longer trusts it

## Output (stdout)
  ✓ GPG key id=7 deleted

## Common errors
- 404 Not Found: id doesn't exist
- 401 Unauthorized: token missing or expired
- 403 Forbidden: token lacks write scope

## OAuth scope required
user_info (write)

## Rate limit
1000 req/hour.

## Example (closing the loop)
  # 1. List to find the id
  gitee gpg-key list

  # 2. Delete
  gitee gpg-key delete 7 --force

  # 3. Verify
  gitee gpg-key list | grep "^7"  # should print nothing

## See also
- ` + "`" + `gitee gpg-key list` + "`" + `   — find the id
- ` + "`" + `gitee gpg-key add` + "`" + `    — register a replacement key
`
