package types

// User represents a Gitee user account.
type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"` // "User" or "Organization"
	Blog      string `json:"blog"`
	Bio       string `json:"bio"`
	// PublicRepos and PublicGists fields omitted if not in Gitee API response
	Followers int `json:"followers"`
	Following int `json:"following"`
}
