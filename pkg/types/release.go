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
