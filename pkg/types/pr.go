package types

// PullRequest represents a Gitee pull request.
type PullRequest struct {
	ID           int64   `json:"id"`
	Number       int     `json:"number"`
	State       string  `json:"state"`
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	HTMLURL     string  `json:"html_url"`
	DiffURL     string  `json:"diff_url"`
	PatchURL    string  `json:"patch_url"`
	User        User    `json:"user"`
	Assignees   []User  `json:"assignees"`
	Labels      []Label `json:"labels"`
	Milestone   *Milestone `json:"milestone"`
	Head        *PRBranch `json:"head"`
	Base        *PRBranch `json:"base"`
	Merged      bool   `json:"merged"`
	MergedAt    *string `json:"merged_at,omitempty"`
	Mergeable   *bool   `json:"mergeable,omitempty"`
	Comments    int    `json:"comments"`
	Commits     int    `json:"commits"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
	ChangedFiles int   `json:"changed_files"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// PRBranch represents the head or base branch of a PR.
type PRBranch struct {
	Label string     `json:"label"`
	Ref   string     `json:"ref"`
	SHA   string     `json:"sha"`
	Repo  Repository `json:"repo"`
}

// PRCreateRequest is used when creating a pull request.
type PRCreateRequest struct {
	Title               string   `json:"title"`
	Body                string   `json:"body,omitempty"`
	Head                string   `json:"head"`
	Base                string   `json:"base"`
	Draft               bool     `json:"draft,omitempty"`
	Issue               int      `json:"issue,omitempty"`
	RemoveHeadOnMerge   bool     `json:"remove_head_on_merge,omitempty"`
	SquashMergeCommitTitle string `json:"squash_merge_commit_title,omitempty"`
	SquashMergeCommitMessage string `json:"squash_merge_commit_message,omitempty"`
	MergeCommitTitle    string   `json:"merge_commit_title,omitempty"`
	MergeCommitMessage   string  `json:"merge_commit_message,omitempty"`
}

// PRMergeRequest is used when merging a pull request.
type PRMergeRequest struct {
	MergeCommitTitle string `json:"merge_commit_title,omitempty"`
	MergeCommitMessage string `json:"merge_commit_message,omitempty"`
	Squash          bool   `json:"squash,omitempty"`
	SquashMessage   string `json:"squash_message,omitempty"`
	DoSquashFlag    bool   `json:"do_squash_flag,omitempty"`
	PRUUid         string `json:"pr_u_uuid,omitempty"`
}

// PRReviewRequest is used when reviewing a pull request.
type PRReviewRequest struct {
	Body     string `json:"body,omitempty"`
	Event    string `json:"event"` // "APPROVE", "REQUEST_CHANGES", "COMMENT"
	Comments []struct {
		Path     string `json:"path"`
		Position int    `json:"position"`
		Body     string `json:"body"`
	} `json:"comments,omitempty"`
}

// PRComment represents a comment on a pull request.
type PRComment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	User      User   `json:"user"`
	CreatedAt string `json:"created_at"`
}
