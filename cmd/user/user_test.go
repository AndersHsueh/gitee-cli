package user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- agent help content (per-verb) -----

func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee user view",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(userViewAgentHelp, want) {
			t.Errorf("view agent help missing required section: %q", want)
		}
	}
}

func TestFollowAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee user follow",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(userFollowAgentHelp, want) {
			t.Errorf("follow agent help missing required section: %q", want)
		}
	}
}

func TestUnfollowAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee user unfollow",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(userUnfollowAgentHelp, want) {
			t.Errorf("unfollow agent help missing required section: %q", want)
		}
	}
}

func TestFollowersAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee user followers",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(userFollowersAgentHelp, want) {
			t.Errorf("followers agent help missing required section: %q", want)
		}
	}
}

func TestFollowingAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee user following",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(userFollowingAgentHelp, want) {
			t.Errorf("following agent help missing required section: %q", want)
		}
	}
}

// ----- wire-format + endpoint contract tests (httptest) -----

// TestUserGetEndpoint verifies GET /users/{username} — path, method,
// auth header, and full JSON unmarshal of the User type.
func TestUserGetEndpoint(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotAuthHdr string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuthHdr = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 12345,
			"login": "alice",
			"name": "Alice Liu",
			"email": "alice@example.com",
			"bio": "Gopher & writer",
			"html_url": "https://gitee.com/alice",
			"type": "User",
			"public_repos": 32,
			"public_gists": 4,
			"followers": 87,
			"following": 12,
			"created_at": "2018-04-22T10:00:00+08:00"
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	var u types.User
	err := client.Get(t.Context(), api.UserGet("alice"), &u)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/users/alice", gotPath)
	assert.Equal(t, "token test-token", gotAuthHdr)
	assert.Equal(t, "alice", u.Login)
	assert.Equal(t, "Alice Liu", u.Name)
	assert.Equal(t, int64(12345), u.ID)
	assert.Equal(t, 32, u.PublicRepos)
	assert.Equal(t, 87, u.Followers)
}

// TestUserCurrentEndpoint verifies GET /user (no segments).
func TestUserCurrentEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": 99,
			"login": "andershsueh",
			"name": "Anders Hsueh",
			"email": "anders@example.com",
			"html_url": "https://gitee.com/andershsueh",
			"public_repos": 5,
			"followers": 1,
			"following": 7
		}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	var u types.User
	err := client.Get(t.Context(), api.UserCurrent(), &u)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/user", gotPath)
	assert.Equal(t, "andershsueh", u.Login)
	assert.Equal(t, int64(99), u.ID)
}

// TestUserFollowersEndpoint verifies GET /users/{username}/followers
// and that the response unmarshals into a User slice.
func TestUserFollowersEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotQuery  string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"id": 222,
				"login": "bob",
				"name": "Bob Chen",
				"html_url": "https://gitee.com/bob"
			},
			{
				"id": 333,
				"login": "carol",
				"name": "Carol Wang",
				"html_url": "https://gitee.com/carol"
			}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	var users []types.User
	err := client.Get(t.Context(), api.UserFollowers("alice")+"?per_page=30", &users)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/users/alice/followers", gotPath)
	assert.Equal(t, "per_page=30", gotQuery)
	require.Len(t, users, 2)
	assert.Equal(t, "bob", users[0].Login)
	assert.Equal(t, int64(222), users[0].ID)
	assert.Equal(t, "carol", users[1].Login)
}

// TestUserFollowingEndpoint verifies GET /users/{username}/following.
func TestUserFollowingEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"id": 444,
				"login": "dan",
				"name": "Dan Smith",
				"html_url": "https://gitee.com/dan"
			}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	var users []types.User
	err := client.Get(t.Context(), api.UserFollowing("alice"), &users)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/users/alice/following", gotPath)
	require.Len(t, users, 1)
	assert.Equal(t, "dan", users[0].Login)
}

// TestUserFollowEndpoint verifies PUT /user/following/{username}.
func TestUserFollowEndpoint(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotAuthHdr string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuthHdr = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	err := client.Put(t.Context(), api.UserFollow("alice"), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/user/following/alice", gotPath)
	assert.Equal(t, "token test-token", gotAuthHdr)
}

// TestUserUnfollowEndpoint verifies DELETE /user/following/{username}.
func TestUserUnfollowEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)
	err := client.Delete(t.Context(), api.UserUnfollow("alice"))
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/user/following/alice", gotPath)
}

// TestUserFollowEdgeRoundTrip verifies the wire shape of the
// UserFollowEdge response type (used by check-follow endpoints,
// not directly by this command group, but added so future
// `user check <username>` doesn't need to re-introduce it).
func TestUserFollowEdgeRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"following true", `{"following":true}`, true},
		{"following false", `{"following":false}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e types.UserFollowEdge
			require.NoError(t, json.Unmarshal([]byte(c.raw), &e))
			assert.Equal(t, c.want, e.Following)
		})
	}
}

// ----- command-level flag/args contracts -----

// resetFlags restores all user-command flag state between tests. Cobra
// preserves parsed flag values after Execute(), so package-level
// variables would otherwise leak across tests.
func resetFlags() {
	viewCmd.SetArgs(nil)
	followCmd.SetArgs(nil)
	unfollowCmd.SetArgs(nil)
	followersCmd.SetArgs(nil)
	followingCmd.SetArgs(nil)
	viewFlags.username = ""
	followFlags.username = ""
	unfollowFlags.username = ""
	followersFlags.username = ""
	followersFlags.limit = 30
	followingFlags.username = ""
	followingFlags.limit = 30
}

// TestFollowCmdRequiresUsername verifies `gitee user follow` errors
// without a positional or --user.
func TestFollowCmdRequiresUsername(t *testing.T) {
	resetFlags()
	defer resetFlags()
	UserCmd.SetArgs([]string{"follow"})
	err := UserCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

// TestUnfollowCmdRequiresUsername verifies the same on unfollow.
func TestUnfollowCmdRequiresUsername(t *testing.T) {
	resetFlags()
	defer resetFlags()
	UserCmd.SetArgs([]string{"unfollow"})
	err := UserCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

// TestFollowersCmdRequiresUsername verifies the same on followers.
func TestFollowersCmdRequiresUsername(t *testing.T) {
	resetFlags()
	defer resetFlags()
	UserCmd.SetArgs([]string{"followers"})
	err := UserCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

// TestFollowingCmdRequiresUsername verifies the same on following.
func TestFollowingCmdRequiresUsername(t *testing.T) {
	resetFlags()
	defer resetFlags()
	UserCmd.SetArgs([]string{"following"})
	err := UserCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

// TestUserFlagsCoveredInAgentHelp ensures every flag exposed by the
// user subcommands is also documented in its agent help. This guards
// the user→agent-help contract end-to-end.
func TestUserFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// view
		{"--user", userViewAgentHelp},
		{"-u", userViewAgentHelp},

		// follow
		{"--user", userFollowAgentHelp},
		{"-u", userFollowAgentHelp},

		// unfollow
		{"--user", userUnfollowAgentHelp},
		{"-u", userUnfollowAgentHelp},

		// followers
		{"--user", userFollowersAgentHelp},
		{"-u", userFollowersAgentHelp},
		{"--limit", userFollowersAgentHelp},
		{"-L", userFollowersAgentHelp},

		// following
		{"--user", userFollowingAgentHelp},
		{"-u", userFollowingAgentHelp},
		{"--limit", userFollowingAgentHelp},
		{"-L", userFollowingAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}
