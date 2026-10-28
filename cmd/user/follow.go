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

var followFlags struct {
	username string
}

var followCmd = &cobra.Command{
	Use:   "follow <username>",
	Short: "Follow a Gitee user",
	Example: `  gitee user follow alice
  gitee user follow --user bob`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(userFollowAgentHelp)
		}
		return nil
	},
	RunE: runFollow,
}

func init() {
	f := followCmd.Flags()
	f.StringVarP(&followFlags.username, "user", "u", "", "Username to follow (alternative to positional)")
}

func runFollow(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	username := followFlags.username
	if len(args) > 0 {
		username = args[0]
	}
	if username == "" {
		return fmt.Errorf("username is required (positional or --user)")
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	if err := client.Put(ctx, api.UserFollow(username), nil, nil); err != nil {
		return fmt.Errorf("following %s: %w", username, err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Now following %s\n", green("✓"), bold(username))
	return nil
}

const userFollowAgentHelp = `# gitee user follow — Agent Help

## Purpose
Subscribe to a Gitee user's public activity. After following, their
public repos and profile changes surface in your notifications feed
(subscription-based, not "follow" in the microblog sense).

## Pre-conditions
- Authenticated with ` + "`" + `user_info` + "`" + ` write scope
- The username resolves to an existing user (404 otherwise)
- You are NOT already following them (re-PUT is a 204 server-side no-op)

## Required arguments
  ` + "`" + `<username>` + "`" + `   Gitee login to follow

## Optional flags
  --user / -u   Equivalent to passing the username as a positional arg

## Idempotency
YES. Re-running on an existing follow is a server-side no-op (204
either way; no error). Re-running on yourself is also a no-op.

## Atomicity
Atomic. Either the follow edge exists, or no change.

## Side effects
- The user appears in your ` + "`" + `user following` + "`" + ` list immediately
- Their activity may surface in your notifications (subject to your
  global notification settings)
- The target user is NOT notified that you followed them

## Post-conditions (on success)
- The follow edge exists in Gitee's graph
- Stdout prints ` + "`" + `✓ Now following <username>` + "`" + `

## Output (stdout)
  ✓ Now following alice

## Common errors
- 404 Not Found: username doesn't exist
- 401 Unauthorized: token missing or expired
- 403 Forbidden: token lacks ` + "`" + `user_info` + "`" + ` write scope

## Race condition warning ⚠️
The shared state is the follow edge between you and ` + "`" + `<username>` + "`" + ` in
Gitee's social graph (plus the target's followers count). Two agents
racing ` + "`" + `user follow` + "`" + ` and ` + "`" + `user unfollow` + "`" + ` on the same target are
last-write-wins: whichever request lands second decides the final
state, with no error on either side. A read-back via
` + "`" + `gitee user following <your-login>` + "`" + ` immediately after this call can
lag a second or two — do not treat the new edge's absence there as
failure; trust the 204 this command already received. Retrying after
a network timeout is SAFE (idempotent no-op — see Idempotency). In
multi-agent setups, serialize follow/unfollow on the same target.

## OAuth scope required
user_info (write)

## Rate limit
1000 req/hour.

## Example (closing the loop)
  # 1. Inspect the target user
  gitee user view alice

  # 2. Follow them
  gitee user follow alice

  # 3. Verify the follow edge
  gitee user following --user me        # 'me' is not literal; use your own login
  # or:
  gitee user following <your-login>     # your 'following' list should now contain alice

## See also
- ` + "`" + `gitee user unfollow` + "`" + `  — reverse the follow
- ` + "`" + `gitee user view` + "`" + `     — inspect a profile
- ` + "`" + `gitee user following` + "`" + ` — list who you follow
- ` + "`" + `gitee user followers` + "`" + ` — list who follows you
`
