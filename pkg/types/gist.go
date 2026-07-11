package types

import "time"

// Gist represents a Gitee gist.
type Gist struct {
	ID          string              `json:"id"`
	URL         string              `json:"url"`
	HTMLURL     string              `json:"html_url"`
	Files       map[string]GistFile `json:"files"`
	Public      bool                `json:"public"`
	Description string              `json:"description"`
	Comments    int                 `json:"comments"`
	User        *User               `json:"user"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
	GistURL     string              `json:"gist_url"`

	// Optional git URLs. Present only when the gist is git-backed (similar
	// to GitHub). When empty, `gitee gist clone` falls back to downloading
	// each file via RawURL into a local directory.
	GitPullURL string `json:"git_pull_url"`
	GitPushURL string `json:"git_push_url"`
	SSHURL     string `json:"ssh_url"`
}

// GistFile represents a file within a gist.
type GistFile struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Language string `json:"language"`
	RawURL   string `json:"raw_url"`
	Size     int    `json:"size"`
	Content  string `json:"content,omitempty"`
}

// GistCreateRequest is used when creating a gist.
type GistCreateRequest struct {
	Description string                     `json:"description,omitempty"`
	Public      bool                       `json:"public"`
	Files       map[string]GistFileContent `json:"files"`
}

// GistFileContent is used when creating/editing gist files.
type GistFileContent struct {
	Content string `json:"content"`
}

// GistEditFile is the value type for GistEditRequest.Files. The
// Filename field (when non-empty) renames the gist file keyed by the
// map key; Content replaces the file's content.
type GistEditFile struct {
	Content  string `json:"content,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// GistEditRequest is used when editing a gist.
type GistEditRequest struct {
	Description string `json:"description,omitempty"`
	Files       map[string]GistEditFile `json:"files,omitempty"`
}

// GistComment represents a comment on a gist. Returned by
// GET /gists/{gist_id}/comments and by the response of POST. The id is a
// numeric comment id used as the path segment for DELETE.
type GistComment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	User      User      `json:"user"`
	// Parsed time mirrors of the timestamp strings; populated when the
	// timestamp parses successfully, otherwise zero. Kept optional so a
	// malformed response from the API does not break unmarshal.
	CreatedAtTime time.Time `json:"-"`
	UpdatedAtTime time.Time `json:"-"`
}

// GistCommentRequest is the create-payload for
// POST /gists/{gist_id}/comments. Distinct from GistComment (the
// response object).
type GistCommentRequest struct {
	Body string `json:"body"`
}
