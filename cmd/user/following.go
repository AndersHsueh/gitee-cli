package user

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/internal/output"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var followingFlags struct {
	username string
	limit    int
}

var followingCmd = &cobra.Command{
	Use:   "following [<username>]",
	Short: "List the users that a given user follows",
	Example: `  gitee user following alice
  gitee user following --user bob
  gitee user following --limit 100 --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(userFollowingAgentHelp)
		}
		return nil
	},
	RunE: runFollowing,
}

func init() {
	f := followingCmd.Flags()
	f.StringVarP(&followingFlags.username, "user", "u", "", "Username whose followings to list (alternative to positional)")
	f.IntVarP(&followingFlags.limit, "limit", "L", 30, "Maximum number of followings to fetch")
}

func runFollowing(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	username := followingFlags.username
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

	path := fmt.Sprintf("%s?per_page=%d", api.UserFollowing(username), followingFlags.limit)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var users []types.User
	if err := client.Get(ctx, path, &users); err != nil {
		return fmt.Errorf("listing users %s follows: %w", username, err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(users, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(users) == 0 {
		fmt.Printf("%s follows no one\n", username)
		return nil
	}

	w := output.NewTableWriter()
	fmt.Fprintln(w, "LOGIN\tNAME\tURL")
	for _, u := range users {
		fmt.Fprintf(w, "%s\t%s\t%s\n", u.Login, u.Name, u.HTMLURL)
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d user(s) followed by %s\n", len(users), username)
	return nil
}

const userFollowingAgentHelp = `# gitee user following — Agent Help

## Purpose
Enumerate every account that a given user follows. Use this to map
their interests, discover accounts they endorse, or back-fill a
follow plan ("follow everyone alice follows").

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `) — raises the rate limit
  even though /users/{username}/following is a public list
- The username resolves to an existing user (404 otherwise)

## Required arguments
  ` + "`" + `<username>` + "`" + `   Gitee login whose followings to enumerate

## Optional flags
  --user / -u   Equivalent to passing the username as a positional arg
  --limit / -L  Maximum number of followings (default 30, max 100 per page)
  --output      ` + "`" + `table` + "`" + ` (default) or ` + "`" + `json` + "`" + `

## Idempotency
YES. Read-only.

## Atomicity
Atomic. One GET. --limit is that request's page size (default 30, max 100).

## Side effects
None. Pure GET.

## Post-conditions (on success)
- Stdout contains a table of users (LOGIN, NAME, URL) or a JSON
  array of User objects
- If ` + "`" + `<username>` + "`" + ` follows no one, ` + "`" + `<username> follows no one` + "`" + `
  is printed
- Account state is unchanged

## Output (stdout, table mode)
  LOGIN     NAME         URL
  dan       Dan Smith    https://gitee.com/dan
  eve       Eve Liu      https://gitee.com/eve

  Total: 2 user(s) followed by alice

## Output (stdout, json mode)
  [
    {
      "id": 333,
      "login": "dan",
      "name": "Dan Smith",
      "html_url": "https://gitee.com/dan",
      ...
    }
  ]

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 404 Not Found:    username doesn't exist

## Race condition warning ⚠️
The following list is live server-side state: edges are added and
removed continuously, by the user themselves or by other sessions.
This output is a snapshot — between listing and acting (e.g. piping
logins into ` + "`" + `gitee user unfollow` + "`" + `), an edge may already be gone,
and unfollow then returns 404 (see its Idempotency note); treat that
as "already unfollowed". This command returns a single page, so a
long following list is truncated at --limit. Read-only and always
safe to retry.

## Next steps (orchestration hints)
After listing who someone follows, an agent typically wants to:
1. Pipe the JSON list to ` + "`" + `gitee user follow <login>` + "`" + ` for each
   in a loop to bulk-follow
2. Pick a ` + "`" + `login` + "`" + ` and run ` + "`" + `gitee user view <login>` + "`" + ` for details
3. Cross-reference with ` + "`" + `gitee repo star list` + "`" + ` for overlap analysis

## OAuth scope required
None (public endpoint, but auth raises the rate limit)

## Rate limit
Gitee API v5: 1000 req/hour authenticated; 60 req/hour anonymous.

## Example
  # Pull every "following" login as a JSON array
  gitee user following alice --output json | jq -r '.[].login'

  # Cap at 100 results (Gitee per_page max)
  gitee user following alice --limit 100

## See also
- ` + "`" + `gitee user followers` + "`" + ` — the inverse: who follows ` + "`" + `<username>` + "`" + `
- ` + "`" + `gitee user view` + "`" + `       — profile details for one user
- ` + "`" + `gitee user follow` + "`" + `     — start following one of them
- ` + "`" + `gitee user unfollow` + "`" + `   — stop following someone
`
