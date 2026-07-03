package user

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var followersFlags struct {
	username string
	limit    int
}

var followersCmd = &cobra.Command{
	Use:   "followers [<username>]",
	Short: "List a user's followers",
	Example: `  gitee user followers alice
  gitee user followers --user bob
  gitee user followers --limit 100 --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(userFollowersAgentHelp)
		}
		return nil
	},
	RunE: runFollowers,
}

func init() {
	f := followersCmd.Flags()
	f.StringVarP(&followersFlags.username, "user", "u", "", "Username whose followers to list (alternative to positional)")
	f.IntVarP(&followersFlags.limit, "limit", "L", 30, "Maximum number of followers to fetch")
}

func runFollowers(cobraCmd *cobra.Command, args []string) error {
	username := followersFlags.username
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

	path := fmt.Sprintf("%s?per_page=%d", api.UserFollowers(username), followersFlags.limit)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var users []types.User
	if err := client.Get(ctx, path, &users); err != nil {
		return fmt.Errorf("listing followers of %s: %w", username, err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(users, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(users) == 0 {
		fmt.Printf("No followers for %s\n", username)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "LOGIN\tNAME\tURL")
	for _, u := range users {
		fmt.Fprintf(w, "%s\t%s\t%s\n", u.Login, u.Name, u.HTMLURL)
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d follower(s) of %s\n", len(users), username)
	return nil
}

const userFollowersAgentHelp = `# gitee user followers — Agent Help

## Purpose
Enumerate every account that follows a given user. Use this for
audience analysis, spam investigations, or to feed a CRM-style
tool with a list of accounts interested in someone.

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `) — raises the rate limit
  even though /users/{username}/followers is a public list
- The username resolves to an existing user (404 otherwise)

## Required arguments
  ` + "`" + `<username>` + "`" + `   Gitee login whose followers to enumerate

## Optional flags
  --user / -u   Equivalent to passing the username as a positional arg
  --limit / -L  Maximum number of followers (default 30, max 100 per page)
  --output      ` + "`" + `table` + "`" + ` (default) or ` + "`" + `json` + "`" + `

## Idempotency
YES. Read-only.

## Atomicity
Atomic per request. The server returns one page (up to ` + "`" + `per_page` + "`" + `
items); use ` + "`" + `--paginate` + "`" + ` (a root flag) to traverse all pages.

## Side effects
None. Pure GET.

## Post-conditions (on success)
- Stdout contains a table of followers (LOGIN, NAME, URL) or a JSON
  array of User objects
- If the user has no followers, ` + "`" + `No followers for <username>` + "`" + ` is printed
- Account state is unchanged

## Output (stdout, table mode)
  LOGIN     NAME         URL
  bob       Bob Chen     https://gitee.com/bob
  carol     Carol Wang   https://gitee.com/carol

  Total: 2 follower(s) of alice

## Output (stdout, json mode)
  [
    {
      "id": 222,
      "login": "bob",
      "name": "Bob Chen",
      "html_url": "https://gitee.com/bob",
      ...
    }
  ]

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 404 Not Found:    username doesn't exist

## Next steps (orchestration hints)
After listing followers, an agent typically wants to:
1. Pick a ` + "`" + `login` + "`" + ` and run ` + "`" + `gitee user view <login>` + "`" + ` for details
2. Pipe the JSON list to ` + "`" + `gitee user follow <login>` + "`" + ` for each
   in a loop to follow them back
3. Cross-reference with ` + "`" + `gitee repo star list` + "`" + ` for overlap analysis

## OAuth scope required
None (public endpoint, but auth raises the rate limit)

## Rate limit
Gitee API v5: 1000 req/hour authenticated; 60 req/hour anonymous.

## Example
  # Pull every follower login as a JSON array
  gitee user followers alice --output json | jq -r '.[].login'

  # Cap at 100 results (Gitee per_page max)
  gitee user followers alice --limit 100

## See also
- ` + "`" + `gitee user following` + "`" + ` — the inverse: who ` + "`" + `<username>` + "`" + ` follows
- ` + "`" + `gitee user view` + "`" + `      — profile details for one user
- ` + "`" + `gitee user follow` + "`" + `    — follow one of them back
`