package sshkey

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
	Short: "Delete an SSH public key by id",
	Example: `  gitee ssh-key delete 12345
  gitee ssh-key delete 12345 --force`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(sshKeyDeleteAgentHelp)
			os.Exit(0)
		}
		return nil
	},
	RunE: runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.Int64VarP(&deleteFlags.id, "id", "i", 0, "SSH key id (alternative to positional)")
	f.BoolVarP(&deleteFlags.force, "force", "f", false, "Skip the y/N confirmation prompt")
}

func runDelete(cobraCmd *cobra.Command, args []string) error {
	id, err := utils.ParseInt64Arg(deleteFlags.id, args, "SSH key")
	if err != nil {
		return err
	}
	if id == 0 {
		return fmt.Errorf("SSH key id is required (positional or --id)")
	}

	if !deleteFlags.force {
		yes, err := utils.ConfirmYN(
			fmt.Sprintf("%s Delete SSH key id=%d?", color.YellowString("?"), id),
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

	if err := client.Delete(ctx, api.SSHKeyDelete(id)); err != nil {
		return fmt.Errorf("deleting SSH key %d: %w", id, err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s SSH key %s deleted\n", green("✓"), bold(fmt.Sprintf("id=%d", id)))
	return nil
}

const sshKeyDeleteAgentHelp = `# gitee ssh-key delete — Agent Help

## Purpose
Remove an SSH public key from the current user's Gitee account. Use
this to revoke access for a lost laptop, decommissioned CI runner, or
rotated key.

## Pre-conditions
- Authenticated with write scope
- The id corresponds to an existing key (returns 404 otherwise)

## Required arguments
  <id>     SSH key id (find via ` + "`" + `gitee ssh-key list` + "`" + `)

## Optional flags
  --id / -i      Alternative to positional
  --force / -f   Skip the y/N confirmation prompt

## Idempotency
NOT idempotent. Re-running on a deleted key returns 404. There is no
"create or update" path on delete; use ` + "`" + `gitee ssh-key add` + "`" + ` to register
a new one.

## Atomicity
Atomic. Either the key is gone, or no change.

## Side effects
- The key can no longer be used for SSH git operations
- Active SSH sessions using the key are NOT terminated (they continue
  until the next reconnect, ~minutes)
- Webhooks subscribed to user events may fire

## Post-conditions (on success)
- The key no longer appears in ` + "`" + `gitee ssh-key list` + "`" + `
- Open SSH sessions using the key will fail on reconnect (typically
  within minutes; not immediate)

## Output (stdout)
  ✓ SSH key id=12345 deleted

## Common errors
- 404 Not Found: id doesn't exist
- 401 Unauthorized: token missing or expired
- 403 Forbidden: token lacks write scope

## OAuth scope required
user_info (write)

## Rate limit
1000 req/hour.

## Example (closing the loop)
  # 1. List keys to find the id of the one to delete
  gitee ssh-key list

  # 2. Delete by id
  gitee ssh-key delete 12345 --force

  # 3. Verify it's gone
  gitee ssh-key list | grep 12345  # should print nothing

## See also
- ` + "`" + `gitee ssh-key list` + "`" + `   — find the id
- ` + "`" + `gitee ssh-key add` + "`" + `    — register a replacement key
`
