package types

// User represents a Gitee user account.
type User struct {
	ID               int64  `json:"id"`
	Login            string `json:"login"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	AvatarURL        string `json:"avatar_url"`
	HTMLURL          string `json:"html_url"`
	Type             string `json:"type"` // "User" or "Organization"
	Blog             string `json:"blog"`
	Bio              string `json:"bio"`
	PublicRepos      int    `json:"public_repos"`
	PublicGists      int    `json:"public_gists"`
	Followers        int    `json:"followers"`
	Following        int    `json:"following"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	StarredURL       string `json:"starred_url"`
	SubscriptionsURL string `json:"subscriptions_url"`
	OrganizationsURL string `json:"organizations_url"`
}

// UserFollowEdge represents a single follow relationship returned by
// GET /user/following/{username} when checking whether the authenticated
// user follows `username`. Gitee returns one of two shapes:
//
//	{ "following": true }                          — they follow them
//	{ "following": false }                         — they don't
//
// We keep the wrapper explicit so callers don't have to handle a bare
// bool. Not used for /users/{username}/{followers,following} which
// returns full User objects.
type UserFollowEdge struct {
	Following bool `json:"following"`
}
