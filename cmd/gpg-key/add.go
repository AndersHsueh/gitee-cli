package gpgkey

import (
	"context"
	"fmt"
	"os"
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
	Example: `  gitee gpg-key add --key ~/my-pubkey.asc
  cat my-pubkey.asc | gitee gpg-key add --key -

# Note: pass the path with a leading "-" to read from stdin
  gitee gpg-key add --key - < my-pubkey.asc`,
	// PreRunE fires BEFORE ValidateRequiredFlags so --agent-help works
	// even when --key is missing.
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			fmt.Print(gpgKeyAddAgentHelp)
			os.Exit(0)
		}
		return nil
	},
	RunE: runAdd,
}

func init() {
	f := addCmd.Flags()
	f.StringVarP(&addFlags.key, "key", "k", "", "ASCII-armored GPG public key OR path to a .asc file (required)")
	_ = addCmd.MarkFlagRequired("key")
}

func runAdd(cobraCmd *cobra.Command, args []string) error {
	key, err := utils.ResolveKeyInput(addFlags.key)
	if err != nil {
		return err
	}
	if err := validateGPPublicKey(key); err != nil {
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
	email := created.PrimaryKeyEmail
	if email == "" {
		email = "(no email)"
	}
	fmt.Printf("%s GPG key %s added (id=%d, email=%s)\n",
		green("✓"), bold(created.PrimaryKeyID), created.ID, bold(email))
	return nil
}

// validateGPPublicKey performs a sanity check on the armored block:
// it MUST contain the PUBLIC marker and MUST NOT contain a PRIVATE
// marker. This is a guard against accidentally uploading a private key.
func validateGPPublicKey(key string) error {
	if !strings.Contains(key, "-----BEGIN PGP PUBLIC KEY BLOCK-----") {
		return fmt.Errorf("input is not a PGP public key (missing '-----BEGIN PGP PUBLIC KEY BLOCK-----' marker)")
	}
	if strings.Contains(key, "PRIVATE") {
		return fmt.Errorf("input appears to be a PGP PRIVATE key — refusing to upload. Re-export with 'gpg --armor --export <key-id>' to get the public key")
	}
	return nil
}

const gpgKeyAddAgentHelp = `# gitee gpg-key add — Agent Help

## Purpose
Register a new GPG public key against the current user's Gitee account.
After this, git commits and tags signed by the matching private key
will show as "verified" on Gitee.

## Pre-conditions
- Authenticated with write scope on user_info
- The key is a PUBLIC key (we reject any input containing the
  "PRIVATE" marker to prevent accidental secret leaks)

## Required flags
  --key / -k   ASCII-armored public key OR path to a .asc file

## Optional flags
  (none)

## Idempotency
NOT idempotent. Each invocation creates a NEW key with a new id.
Re-running with the same key returns 422 (key already exists).

## Atomicity
Atomic. Either the key is registered and an id is returned, or no
change happens.

## Side effects
- The key is immediately usable for signed commit verification
- Webhooks subscribed to user events may fire

## Post-conditions (on success)
- A new GPG key with a unique id exists on the account
- Stdout prints the key id, primary key id, and primary email

## Output (stdout, table mode)
  ✓ GPG key A1B2C3D4E5F6 added (id=7, email=alice@example.com)

## Common errors
- 401 Unauthorized: token missing or expired → gitee auth login
- 422 Unprocessable: key already exists OR format invalid
- "input appears to be a PGP PRIVATE key": you passed the private key
  by mistake. Re-export with ` + "`" + `gpg --armor --export <key-id>` + "`" + `

## Race condition warning ⚠️
After adding a key, signed commits pushed with it may take a few seconds
to show as "verified" on the Gitee web UI. This is server-side caching,
not a CLI bug.

## Next steps (orchestration hints)
1. Export your public key: ` + "`" + `gpg --armor --export <key-id> > my-pubkey.asc` + "`" + `
2. ` + "`" + `gitee gpg-key add --key my-pubkey.asc` + "`" + `
3. Verify: ` + "`" + `gitee gpg-key list` + "`" + ` (look for the new id)
4. Configure git to use this key: ` + "`" + `git config --global user.signingkey <key-id>` + "`" + `
5. Sign a commit: ` + "`" + `git commit -S -m 'signed'` + "`" + `

## OAuth scope required
user_info (write)

## Rate limit
1000 req/hour.

## Example (closing the loop)
  # 1. Export the public key
  gpg --armor --export A1B2C3D4E5F6 > my-pubkey.asc

  # 2. Upload
  gitee gpg-key add --key my-pubkey.asc

  # 3. Verify it landed
  gitee gpg-key list | grep "A1B2C3D4E5F6"

## See also
- ` + "`" + `gitee gpg-key list` + "`" + `   — enumerate keys
- ` + "`" + `gitee gpg-key delete` + "`" + ` — remove a key
- ` + "`" + `gitee auth status` + "`" + `     — confirm token scope
`
