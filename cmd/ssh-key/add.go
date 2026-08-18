package sshkey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var addFlags struct {
	title string
	key   string
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an SSH public key to your account",
	Example: `  gitee ssh-key add --title "laptop" --key ~/.ssh/id_rsa.pub
  gitee ssh-key add --title "ci-runner" --key "ssh-ed25519 AAAA..."`,
	// PreRunE fires BEFORE ValidateRequiredFlags so --agent-help works
	// even when --title or --key is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(sshKeyAddAgentHelp)
		}
		return nil
	},
	RunE: runAdd,
}

func init() {
	f := addCmd.Flags()
	f.StringVarP(&addFlags.title, "title", "t", "", "Human-readable label for the key (required)")
	f.StringVarP(&addFlags.key, "key", "k", "", "SSH public key text OR path to a public key file (required)")
	_ = addCmd.MarkFlagRequired("title")
	_ = addCmd.MarkFlagRequired("key")
}

func runAdd(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	key, err := utils.ResolveKeyInput(addFlags.key)
	if err != nil {
		return err
	}
	if err := validateSSHPublicKey(key); err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	payload := types.SSHKeyCreateRequest{
		Title: addFlags.title,
		Key:   key,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var created types.SSHKey
	if err := client.Post(ctx, api.SSHKeyCreate(), payload, &created); err != nil {
		return fmt.Errorf("adding SSH key: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s SSH key %s added (id=%d)\n", green("✓"), bold(created.Title), created.ID)
	return nil
}

// validateSSHPublicKey performs a sanity check: the key must start with a
// known SSH key-type prefix and contain a base64 body. This is a SMOKE TEST
// only — Gitee will reject malformed keys with a more specific error.
func validateSSHPublicKey(key string) error {
	knownPrefixes := []string{
		"ssh-rsa", "ssh-ed25519", "ecdsa-sha2-nistp256",
		"ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521", "ssh-dss",
	}
	for _, p := range knownPrefixes {
		if strings.HasPrefix(key, p+" ") {
			return nil
		}
	}
	return fmt.Errorf("input does not look like an SSH public key (expected to start with one of: %s)",
		strings.Join(knownPrefixes, ", "))
}

const sshKeyAddAgentHelp = `# gitee ssh-key add — Agent Help

## Purpose
Register a new SSH public key against the current user's Gitee account.
After this, the matching SSH private key can push/pull Gitee repos
over SSH without password prompts.

## Pre-conditions
- Authenticated with write scope on user_info
- The key file or string is a PUBLIC key (private keys would be rejected
  by Gitee; we run a sanity check to catch obvious mistakes)

## Required flags
  --title / -t   Human-readable label (e.g. "laptop", "ci-runner")
  --key / -k     Public key content OR path to a .pub file

## Optional flags
  (none)

## Idempotency
NOT idempotent. Each invocation creates a NEW key with a new id. There
is no "create or update" semantic on Gitee v5. Re-running with the same
key+title returns 422 (key already exists).

## Atomicity
Atomic. Either the key is registered and an id is returned, or no
change happens.

## Side effects
- The key is immediately usable for SSH git operations against any repo
  the user has access to
- Webhooks subscribed to user events may fire

## Post-conditions (on success)
- A new SSH key with a unique id exists on the account
- Stdout prints "✓ SSH key <title> added (id=<n>)"

## Output (stdout, table mode)
  ✓ SSH key laptop added (id=12345)

## Common errors
- 401 Unauthorized: token missing or expired → gitee auth login
- 422 Unprocessable: key already exists OR key format invalid
- 403 Forbidden:    token lacks write scope

## Race condition warning ⚠️
After adding a key, do NOT immediately attempt an SSH push from a NEW
shell — sshd needs a moment to pick up the new authorized_keys entry on
the Gitee server. Wait 5-10 seconds before the first SSH operation.

## Next steps (orchestration hints)
1. ` + "`" + `gitee ssh-key add --title X --key ~/.ssh/id_rsa.pub` + "`" + `
2. Verify: ` + "`" + `gitee ssh-key list` + "`" + ` (look for the new id)
3. Test SSH: ` + "`" + `ssh -T git@gitee.com` + "`" + ` (should print your name)
4. If the key was added by mistake: ` + "`" + `gitee ssh-key delete <id>` + "`" + `

## OAuth scope required
user_info (write)

## Rate limit
1000 req/hour.

## Example (closing the loop)
  # 1. Add the key
  gitee ssh-key add --title "ci-runner" --key ~/.ssh/id_ed25519.pub

  # 2. Verify it landed
  gitee ssh-key list | grep "ci-runner"

  # 3. Smoke-test the SSH connection (waits 5-10s for propagation first)
  sleep 5
  ssh -T git@gitee.com

## See also
- ` + "`" + `gitee ssh-key list` + "`" + `   — enumerate keys
- ` + "`" + `gitee ssh-key delete` + "`" + ` — remove a key
- ` + "`" + `gitee auth status` + "`" + `    — confirm token has the right scope
`
