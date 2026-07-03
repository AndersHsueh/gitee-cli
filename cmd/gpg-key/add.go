package gpgkey

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
	key string
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a GPG public key to your account",
	Example: `  gitee gpg-key add --key ~/public_key.asc
  gitee gpg-key add --key "-----BEGIN PGP PUBLIC KEY BLOCK-----"`,
	// PreRunE fires BEFORE ValidateRequiredFlags so --agent-help works
	// even when --key is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gpgKeyAddAgentHelp)
		}
		return nil
	},
	RunE: runAdd,
}

func init() {
	f := addCmd.Flags()
	f.StringVarP(&addFlags.key, "key", "k", "", "GPG public key text OR path to a public key file (required)")
	_ = addCmd.MarkFlagRequired("key")
}

func runAdd(cobraCmd *cobra.Command, args []string) error {
	key, err := utils.ResolveKeyInput(addFlags.key)
	if err != nil {
		return err
	}
	if err := validateGPGPublicKey(key); err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	payload := types.GPGKeyCreateRequest{
		ArmoredPublicKey: key,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var created types.GPGKey
	if err := client.Post(ctx, api.GPGKeyCreate(), payload, &created); err != nil {
		return fmt.Errorf("adding GPG key: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s GPG key %s added (id=%d)\n", green("✓"), bold(created.PrimaryKeyID), created.ID)
	return nil
}

// validateGPGPublicKey performs a sanity check: the key must be an
// ASCII-armored PGP public key block. This is a SMOKE TEST only — Gitee
// will reject malformed keys with a more specific error.
func validateGPGPublicKey(key string) error {
	requiredPrefixes := []string{
		"-----BEGIN PGP PUBLIC KEY BLOCK-----",
	}
	for _, p := range requiredPrefixes {
		if strings.HasPrefix(key, p) {
			return nil
		}
	}
	return fmt.Errorf("input does not look like an ASCII-armored GPG public key (expected to start with: %s)",
		requiredPrefixes[0])
}

const gpgKeyAddAgentHelp = `# gitee gpg-key add — Agent Help

## Purpose
Register a new GPG public key against the current user's Gitee account.
After this, commits and tags signed by the matching GPG private key
will display as "Verified" on Gitee.

## Pre-conditions
- Authenticated with write scope on user_info
- The key file or string is a PUBLIC key (private keys would be rejected
  by Gitee; we run a sanity check to catch obvious mistakes)

## Required flags
  --key / -k     Public key content OR path to a .asc armored file

## Optional flags
  (none)

## Idempotency
NOT idempotent. Each invocation may register a duplicate key or fail
with 422 if the same armor block has already been uploaded. Gitee v5
does not document an "update" path for GPG keys.

## Atomicity
Atomic. Either the key is registered and an id is returned, or no
change happens.

## Side effects
- The key may immediately appear as a verifiable signer on existing
  and future commits/tags that match its primary key id
- Webhooks subscribed to user events may fire

## Post-conditions (on success)
- A new GPG key with a unique id exists on the account
- Stdout prints "✓ GPG key <primary_key_id> added (id=<n>)"

## Output (stdout, table mode)
  ✓ GPG key ABC12345...DEF added (id=12345)

## Common errors
- 401 Unauthorized: token missing or expired → gitee auth login
- 422 Unprocessable: key already exists OR key format invalid
- 403 Forbidden:    token lacks write scope

## Race condition warning ⚠️
After adding a key, do NOT assume Gitee's commit verification picks it
up instantly. Webhook-driven verification caches can take a minute or
two to propagate.

## Next steps (orchestration hints)
1. ` + "`" + `gitee gpg-key add --key ~/public_key.asc` + "`" + `
2. Verify: ` + "`" + `gitee gpg-key list` + "`" + ` (look for the new id)
3. Configure git to sign: ` + "`" + `git config --global user.signingkey <primary_key_id>` + "`" + `
4. If the key was added by mistake: ` + "`" + `gitee gpg-key delete <id>` + "`" + `

## OAuth scope required
user_info (write)

## Rate limit
1000 req/hour.

## Example (closing the loop)
  # 1. Add the key
  gitee gpg-key add --key ~/public_key.asc

  # 2. Verify it landed
  gitee gpg-key list | grep "ABC12345"

  # 3. Sign a test commit
  git commit --allow-empty -S -m "test signed commit"
  git push origin HEAD
  # Visit the commit on gitee.com — it should display as "Verified".

## See also
- ` + "`" + `gitee gpg-key list` + "`" + `   — enumerate keys
- ` + "`" + `gitee gpg-key delete` + "`" + ` — remove a key
- ` + "`" + `gitee auth status` + "`" + `    — confirm token has the right scope
`
