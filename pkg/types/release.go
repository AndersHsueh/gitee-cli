package types

import "time"

// Release represents a Gitee release.
type Release struct {
	ID              int64      `json:"id"`
	TagName         string     `json:"tag_name"`
	Name            string     `json:"name"`
	Body            string     `json:"body"`
	TargetCommitish string     `json:"target_commitish"`
	Prerelease      bool       `json:"prerelease"`
	Draft           bool       `json:"draft"`
	Author          User       `json:"author"`
	HTMLURL         string     `json:"html_url"`
	CreatedAt       time.Time  `json:"created_at"`
	PublishedAt     *time.Time `json:"published_at"`
	Assets          []Asset    `json:"assets"`
}

// Asset represents a downloadable asset attached to a release.
type Asset struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Label         string `json:"label"`
	ContentType   string `json:"content_type"`
	Size          int    `json:"size"`
	DownloadURL   string `json:"browser_download_url"`
	State         string `json:"state"`
	UpdatedAt     string `json:"updated_at"`
	CreatedAt     string `json:"created_at"`
	DownloadCount int    `json:"download_count"`
}

// ReleaseCreateRequest is used when creating a new release.
type ReleaseCreateRequest struct {
	TagName         string `json:"tag_name"`
	Name            string `json:"name,omitempty"`
	Body            string `json:"body,omitempty"`
	TargetCommitish string `json:"target_commitish,omitempty"`
	Prerelease      bool   `json:"prerelease,omitempty"`
}

// ReleaseEditRequest is used when editing an existing release.
type ReleaseEditRequest struct {
	Name       string `json:"name,omitempty"`
	Body       string `json:"body,omitempty"`
	Prerelease *bool  `json:"prerelease,omitempty"`
}

// Tag represents a Git tag in a Gitee repository, as returned by
// GET /repos/{owner}/{repo}/tags. Only a subset of fields is exposed
// by the Gitee v5 API (name + message + commit metadata).
type Tag struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Commit  struct {
		SHA       string `json:"sha"`
		URL       string `json:"url"`
		Author    *User  `json:"author,omitempty"`
		Committer *User  `json:"committer,omitempty"`
	} `json:"commit"`
	Tagger *User `json:"tagger,omitempty"`
}

// TagCreateRequest is used when creating a new lightweight or annotated
// tag via POST /repos/{owner}/{repo}/tags. Gitee v5 requires the source
// commit SHA (refs) plus the tag name; an optional annotation message
// upgrades a lightweight tag to an annotated one.
type TagCreateRequest struct {
	Refs       string `json:"refs"`
	TagName    string `json:"tag_name"`
	TagMessage string `json:"tag_message,omitempty"`
}
