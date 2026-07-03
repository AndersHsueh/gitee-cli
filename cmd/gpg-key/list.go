package gpgkey

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/handler"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var listFlags struct {
	limit int
	page  int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your GPG public keys",
	Example: `  gitee gpg-key list
  gitee gpg-key list --limit 50
  gitee gpg-key list --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(gpgKeyListAgentHelp)
		}
		return nil
	},
	RunE: runList,
}

func init() {
	f := listCmd.Flags()
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of keys to return")
	f.IntVar(&listFlags.page, "page", 1, "Page number (1-based)")
}

func runList(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.GPGKeysList()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var keys []types.GPGKey
	if err := client.Get(ctx, path, &keys); err != nil {
		return fmt.Errorf("listing GPG keys: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		return handler.PrintJSON(keys)
	}

	if len(keys) == 0 {
		fmt.Println("No GPG keys registered. Use 'gitee gpg-key add' to add one.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPRIMARY_KEY_ID\tEMAIL\tCREATED")
	for _, k := range keys {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", k.ID, k.PrimaryKeyID, k.PrimaryKeyEmail, k.CreatedAt)
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d key(s)\n", len(keys))
	return nil
}

const gpgKeyListAgentHelp = `# gitee gpg-key list — Agent Help

## Purpose
List all GPG public keys registered against the current user's Gitee
account. Use this to discover key IDs for deletion, audit which signing
keys are trusted by your account, or produce an inventory for compliance.

## Pre-conditions
- Authenticated (gitee auth status)
- Token has ` + "`" + `user_info` + "`" + ` or ` + "`" + `projects` + "`" + ` scope

## Optional flags
  --limit / -L   Maximum number of keys (default 30)
  --page         Page number (default 1; use --paginate to traverse all)

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET.

## Post-conditions (on success)
- Stdout contains a table of keys (ID, PRIMARY_KEY_ID, EMAIL, CREATED)
- Account state is unchanged

## Output (stdout, table mode)
  ID       PRIMARY_KEY_ID            EMAIL                CREATED
  12345    ABC12345...DEF            user@example.com     2026-01-15T08:23:11+08:00
  12346    XYZ67890...ABC            alt@example.com      2026-02-04T14:02:55+08:00

## Common errors
- 401 Unauthorized: token missing or expired → gitee auth login

## OAuth scope required
user_info

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  gitee gpg-key list --output json | jq '.[] | {id, primary_key_id}'

## See also
- ` + "`" + `gitee gpg-key add` + "`" + `    — register a new public key
- ` + "`" + `gitee gpg-key delete` + "`" + ` — remove a key by id
`
