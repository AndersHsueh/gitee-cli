package types

// Collaborator represents a Gitee repository collaborator entry.
//
// Gitee v5 returns collaborator records shaped like:
//
//	{
//	  "id": 12345,
//	  "login": "alice",
//	  "name": "Alice",
//	  "avatar_url": "https://...",
//	  "html_url": "https://gitee.com/alice",
//	  "type": "User",
//	  "permission": "push"
//	}
//
// `Permission` is one of "pull" / "push" / "admin" (lowercase; mirrors
// Gitee's documented permission levels). `RoleName` is the legacy field
// some endpoints still emit; we keep it optional.
type Collaborator struct {
	ID         int64  `json:"id"`
	Login      string `json:"login"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatar_url"`
	HTMLURL    string `json:"html_url"`
	Type       string `json:"type"`
	Permission string `json:"permission,omitempty"`
	RoleName   string `json:"role_name,omitempty"`
}

// CollaboratorAddRequest is the body for
// PUT /repos/{owner}/{repo}/collaborators/{username}.
//
// Permission is one of: pull (read-only), push (read+write), admin (full).
// Defaults to "push" on Gitee's side when omitted.
type CollaboratorAddRequest struct {
	Permission string `json:"permission,omitempty"`
}
