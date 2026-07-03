package user

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var unfollowFlags struct {
	username string
}

var unfollowCmd = &cobra.Command{
	Use:   "unfollow <username>",
	Short: "Unfollow a Gitee user",
	Example: `  gitee user unfollow alice
  gitee user unfollow --user bob`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(userUnfollowAgentHelp)
		}
		return nil
	},
	RunE: runUnfollow,
}

func init() {
	f := unfollowCmd.Flags()
	f.StringVarP(&unfollowFlags.username, "user", "u", "", "Username to unfollow (alternative to positional)")
}

func runUnfollow(cobraCmd *cobra.Command, args []string) error {
	username := unfollowFlags.username
	if len(args) > 0 {
		username = args[0]
	}
	if username == "" {
		return fmt.Errorf("username is required (positional or --user)")
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, api.UserUnfollow(username)); err != nil {
		return fmt.Errorf("unfollowing %s: %w", username, err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Unfollowed %s\n", green("✓"), bold(username))
	return nil
}

const userUnfollowAgentHelp = `# gitee user unfollow — Agent Help

## Purpose
Remove a previously created follow edge between the authenticated
user and ` + "`" + `<username>` + "`" + `. Their activity stops appearing in your
notifications (subject to global notification settings).

## Pre-conditions
- Authenticated with ` + "`" + `user_info` + "`" + ` write scope
- The username resolves to an existing user (404 if you were never
  following them — Gitee returns 404 rather than 204 for missing edges)

## Required arguments
  ` + "`" + `<username>` + "`" + `   Gitee login to unfollow

## Optional flags
  --user / -u   Equivalent to passing the username as a positional arg

## Idempotency
PARTIAL. Re-running on a non-followed user returns 404 (NOT idempotent
in the strict sense — there is a non-zero error on re-run). Use
` + "`" + `user following` + "`" + ` to confirm before unfollowing in scripts.

## Atomicity
Atomic. Either the follow edge is gone, or no change.

## Side effects
- The user is removed from your ` + "`" + `user following` + "`" + ` list immediately
- Their activity stops surfacing in your notifications
- The target user is NOT notified that you unfollowed them

## Post-conditions (on success)
- The follow edge no longer exists in Gitee's graph
- Stdout prints ` + "`" + `✓ Unfollowed <username>` + "`" + `

## Output (stdout)
  ✓ Unfollowed alice

## Common errors
- 404 Not Found: you weren't following them (or username doesn't exist)
- 401 Unauthorized: token missing or expired
- 403 Forbidden: token lacks ` + "`" + `user_info` + "`" + ` write scope

## OAuth scope required
user_info (write)

## Rate limit
1000 req/hour.

## Example (closing the loop)
  # 1. See who you currently follow
  gitee user following <your-login>

  # 2. Unfollow one
  gitee user unfollow alice

  # 3. Verify they're gone from the list
  gitee user following <your-login> | grep alice   # should print nothing

## See also
- ` + "`" + `gitee user follow` + "`" + `    — create the follow edge
- ` + "`" + `gitee user following` + "`" + ` — list who you follow (verify before / after)
- ` + "`" + `gitee user view` + "`" + `     — inspect a profile
`