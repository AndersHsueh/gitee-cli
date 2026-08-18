package sshkey

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/cmdkit"
	"gitee-cli/internal/output"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var listFlags struct {
	limit int
	page  int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your SSH public keys",
	Example: `  gitee ssh-key list
  gitee ssh-key list --limit 50
  gitee ssh-key list --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(sshKeyListAgentHelp)
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
	rt := cmd.RuntimeFromCmd(cobraCmd)
	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}
	opts := cmd.NewCmdkitOptionsFrom(rt, client)
	return cmdkit.RunList[types.SSHKey](opts, cmdkit.ListSpec[types.SSHKey]{
		// ssh-key list lists the authenticated user's keys — no target repo.
		ResolveOwnerRepo: func() (string, string, error) {
			return "", "", nil
		},
		Path: func(_, _ string) string {
			// Clamp per_page to Gitee v5's 1-100 range before sending it.
			perPage := listFlags.limit
			if perPage < 1 {
				perPage = 30
			}
			if perPage > 100 {
				perPage = 100
			}
			qv := url.Values{}
			qv.Set("page", strconv.Itoa(listFlags.page))
			qv.Set("per_page", strconv.Itoa(perPage))
			return api.SSHKeysList() + "?" + qv.Encode()
		},
		// ssh-key list caps per_page with --limit, pages via --page, and does
		// a single GET — no --paginate support.
		Fetch: func(ctx context.Context, client *api.Client, path string) ([]types.SSHKey, error) {
			var keys []types.SSHKey
			if err := client.Get(ctx, path, &keys); err != nil {
				return nil, fmt.Errorf("listing SSH keys: %w", err)
			}
			return keys, nil
		},
		Columns: []string{"ID", "TITLE", "CREATED"},
		Row: func(k types.SSHKey) []string {
			return []string{strconv.FormatInt(k.ID, 10), k.Title, k.CreatedAt}
		},
		EmptyText: func(_, _ string) string {
			return "No SSH keys registered. Use 'gitee ssh-key add' to add one."
		},
		Footer: func(count int) string {
			return fmt.Sprintf("\nTotal: %d key(s)\n", count)
		},
		JSON: func(items []types.SSHKey) error {
			return output.PrintJSON(items)
		},
	})
}

const sshKeyListAgentHelp = `# gitee ssh-key list — Agent Help

## Purpose
List all SSH public keys registered against the current user's Gitee
account. Use this to discover key IDs for deletion, audit which machines
have access, or produce an inventory for compliance.

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
- Stdout contains a table of keys (ID, TITLE, CREATED)
- Account state is unchanged

## Output (stdout, table mode)
  ID       TITLE             CREATED
  12345    laptop            2026-01-15T08:23:11+08:00
  12346    workstation       2026-02-04T14:02:55+08:00

## Common errors
- 401 Unauthorized: token missing or expired → gitee auth login

## Race condition warning ⚠️
Read-only; the account's key set can change at any time (another
machine running ` + "`" + `gitee ssh-key add` + "`" + `, a deletion via the web UI).
This list is a point-in-time snapshot: an id shown here may already be
gone by the time you act on it (your delete then 404s), and a
just-added key is missing until you re-list. If you pick an id from
this output to act on later, re-run this command immediately before
the action rather than trusting a cached list. Safe to retry freely
(see Idempotency).

## OAuth scope required
user_info

## Rate limit
Gitee API v5: 1000 req/hour authenticated.

## Example
  gitee ssh-key list --output json | jq '.[] | {id, title}'

## See also
- ` + "`" + `gitee ssh-key add` + "`" + `    — register a new public key
- ` + "`" + `gitee ssh-key delete` + "`" + ` — remove a key by id
`
