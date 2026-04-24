package types

// Repository represents a Gitee repository.
type Repository struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	FullName        string     `json:"full_name"`
	Owner           User       `json:"owner"`
	Private         bool       `json:"private"`
	HTMLURL         string     `json:"html_url"`
	Description     string     `json:"description"`
	Fork            bool       `json:"fork"`
	Language        string     `json:"language"`
	StarsCount      int        `json:"stars_count"`
	ForksCount      int        `json:"forks_count"`
	WatchersCount   int        `json:"watchers_count"`
	DefaultBranch   string     `json:"default_branch"`
	OpenIssuesCount int        `json:"open_issues_count"`
	PushedAt        string     `json:"pushed_at"`
	CreatedAt       string     `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
	Homepage        string     `json:"homepage"`
	Permissions     *struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions,omitempty"`
}

// RepositoryCreateRequest is used when creating a new repository.
type RepositoryCreateRequest struct {
	Name                  string `json:"name"`
	Description           string `json:"description,omitempty"`
	Private               bool   `json:"private"`
	AutoInit              bool   `json:"auto_init,omitempty"`
	GitignoreTemplate     string `json:"gitignore_template,omitempty"`
	LicenseTemplate       string `json:"license_template,omitempty"`
	ReadmeTemplate        string `json:"readme_template,omitempty"`
	Organization          string `json:"organization,omitempty"`
	Homepage              string `json:"homepage,omitempty"`
	HasIssues             bool   `json:"has_issues,omitempty"`
	HasWiki               bool   `json:"has_wiki,omitempty"`
	HasPullRequests       bool   `json:"has_pull_requests,omitempty"`
	HasProjects           bool   `json:"has_projects,omitempty"`
	SquashMergeCommitTitle string `json:"squash_merge_commit_title,omitempty"`
	SquashMergeCommitMessage string `json:"squash_merge_commit_message,omitempty"`
	MergeCommitTitle      string `json:"merge_commit_title,omitempty"`
	MergeCommitMessage    string `json:"merge_commit_message,omitempty"`
}

// RepositoryEditRequest is used when editing a repository.
type RepositoryEditRequest struct {
	Name                  string `json:"name,omitempty"`
	Description           string `json:"description,omitempty"`
	Private               *bool  `json:"private,omitempty"`
	DefaultBranch         string `json:"default_branch,omitempty"`
	HasIssues             *bool  `json:"has_issues,omitempty"`
	HasWiki               *bool  `json:"has_wiki,omitempty"`
	HasPullRequests       *bool  `json:"has_pull_requests,omitempty"`
	HasProjects           *bool  `json:"has_projects,omitempty"`
	Archived              *bool  `json:"archived,omitempty"`
}

// Branch represents a repository branch.
type Branch struct {
	Name     string `json:"name"`
	Commit   *struct {
		SHA string `json:"sha"`
	} `json:"commit"`
	Protected bool `json:"protected"`
}
