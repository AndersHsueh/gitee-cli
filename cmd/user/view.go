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

var viewFlags struct {
	username string
}

var viewCmd = &cobra.Command{
	Use:   "view [<username>]",
	Short: "View a Gitee user's profile (yours if no username is given)",
	Example: `  gitee user view alice
  gitee user view                  # authenticated user (yourself)
  gitee user view --user bob
  gitee user view alice --output json`,
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(userViewAgentHelp)
		}
		return nil
	},
	RunE: runView,
}

func init() {
	f := viewCmd.Flags()
	f.StringVarP(&viewFlags.username, "user", "u", "", "Username to view (alternative to positional)")
}

func runView(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	username := viewFlags.username
	if len(args) > 0 {
		username = args[0]
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	// No username → authenticated user via /user; otherwise /users/{username}.
	var path string
	if username == "" {
		path = api.UserCurrent()
	} else {
		path = api.UserGet(username)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var u types.User
	if err := client.Get(ctx, path, &u); err != nil {
		return fmt.Errorf("viewing user: %w", err)
	}

	if rt.Output == "json" {
		b, _ := json.MarshalIndent(u, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	w := output.NewTableWriter()
	fmt.Fprintf(w, "Login:\t%s\n", u.Login)
	fmt.Fprintf(w, "Name:\t%s\n", u.Name)
	if u.Email != "" {
		fmt.Fprintf(w, "Email:\t%s\n", u.Email)
	}
	fmt.Fprintf(w, "Bio:\t%s\n", u.Bio)
	if u.Blog != "" {
		fmt.Fprintf(w, "Blog:\t%s\n", u.Blog)
	}
	fmt.Fprintf(w, "Type:\t%s\n", u.Type)
	fmt.Fprintf(w, "Public Repos:\t%d\n", u.PublicRepos)
	fmt.Fprintf(w, "Public Gists:\t%d\n", u.PublicGists)
	fmt.Fprintf(w, "Followers:\t%d\n", u.Followers)
	fmt.Fprintf(w, "Following:\t%d\n", u.Following)
	fmt.Fprintf(w, "Created:\t%s\n", u.CreatedAt)
	fmt.Fprintf(w, "URL:\t%s\n", u.HTMLURL)
	return w.Flush()
}

const userViewAgentHelp = `# gitee user view — Agent Help

## Purpose
Fetch a single Gitee user's profile (login, name, bio, public_repos,
followers, etc.). Pass a username to inspect any account; omit it to
read the authenticated user's own profile.

## Pre-conditions
- Authenticated (` + "`" + `gitee auth status` + "`" + `) for /user (the self-view path).
  /users/{username} is a public endpoint, but a token raises the rate limit.
- For the username form: the login must resolve to an existing user.

## Required arguments
  Optional: ` + "`" + `<username>` + "`" + ` — the Gitee login. If omitted, the authenticated
  user is returned via ` + "`" + `/user` + "`" + `.

## Optional flags
  --user / -u    Equivalent to passing the username as a positional arg
  --output       ` + "`" + `table` + "`" + ` (default) or ` + "`" + `json` + "`" + `

## Idempotency
YES. Read-only.

## Atomicity
Atomic.

## Side effects
None. Pure GET.

## Post-conditions (on success)
- Stdout contains a key/value block (Login, Name, Bio, Public Repos,
  Followers, Following, URL) or a JSON User object
- Account state is unchanged

## Output (stdout, table mode)
  Login:        alice
  Name:         Alice Liu
  Bio:          Gopher & writer
  Type:         User
  Public Repos: 32
  Public Gists: 4
  Followers:    87
  Following:    12
  Created:      2018-04-22T10:00:00+08:00
  URL:          https://gitee.com/alice

## Output (stdout, json mode)
  {
    "id": 12345,
    "login": "alice",
    "name": "Alice Liu",
    "email": "alice@example.com",
    "public_repos": 32,
    "followers": 87,
    "following": 12,
    ...
  }

## Common errors
- 401 Unauthorized: token missing or expired → ` + "`" + `gitee auth login` + "`" + `
- 404 Not Found:    username doesn't exist (typo, or account deleted)

## Race condition warning ⚠️
The profile is live server-side state: counters (followers, public
repos) and mutable fields (bio, blog, email) can change the moment
after you read them. Treat this output as a snapshot — if a later
action depends on a value (e.g. "only follow if followers < 100"),
re-run this command immediately before acting rather than trusting an
older read. The account itself can also be renamed or deleted between
your view and a follow-up call, surfacing as a 404 there. Read-only
and always safe to retry.

## Next steps (orchestration hints)
After reading a profile, an agent typically wants to:
1. Run ` + "`" + `gitee user followers <login>` + "`" + ` to enumerate their follower set
2. Run ` + "`" + `gitee user following <login>` + "`" + ` to see who they follow
3. Run ` + "`" + `gitee user follow <login>` + "`" + ` to subscribe to their activity
4. Run ` + "`" + `gitee search repos --user <login>` + "`" + ` to enumerate their repos

## OAuth scope required
None for /users/{username} (public). For /user (self-view) the token
needs ` + "`" + `user_info` + "`" + `.

## Rate limit
Gitee API v5: 1000 req/hour authenticated; 60 req/hour anonymous.

## Example
  # Read another user's profile as JSON, extract the public repos count
  gitee user view alice --output json | jq -r '.public_repos'

  # Read your own profile
  gitee user view

  # Equivalent using --user flag (no positional)
  gitee user view --user alice

## See also
- ` + "`" + `gitee user follow` + "`" + `    — start following a user
- ` + "`" + `gitee user unfollow` + "`" + `  — stop following
- ` + "`" + `gitee user followers` + "`" + ` — enumerate their followers
- ` + "`" + `gitee user following` + "`" + ` — enumerate who they follow
- ` + "`" + `gitee search users` + "`" + `  — discover users by query
`
