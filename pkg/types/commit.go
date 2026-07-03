package types

// Commit / compare payload types for the Gitee v5 commit API. The fields
// below mirror the swagger schema for:
//
//   GET /repos/{owner}/{repo}/commits
//   GET /repos/{owner}/{repo}/commits/{sha}
//   GET /repos/{owner}/{repo}/compare/{base}...{head}
//
// Gitee's commit payload is GitHub-shaped: the top-level object exposes
// repo metadata (sha/url/html_url) and a nested "commit" object that
// carries the author/committer/message/tree fields. We keep both layers
// on the same struct because list and detail endpoints share the same
// shape — the only difference is that the detail endpoint additionally
// populates `stats` and `files` (omitted from the list response to keep
// it small).

// CommitAuthor describes one side (author or committer) of a commit.
// Gitee uses distinct "name + email" pairs in the embedded `commit.author`
// / `commit.committer` blocks (git metadata) and a separate `author` /
// `committer` block at the top level (the Gitee user object, when the
// author has a linked Gitee account).
type CommitAuthor struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Date     string `json:"date,omitempty"`
	Username string `json:"username,omitempty"`
}

// CommitStats describes the file-level change summary returned by Gitee
// for a single-commit detail request. List requests omit this object.
type CommitStats struct {
	Additions int `json:"additions,omitempty"`
	Deletions int `json:"deletions,omitempty"`
	Total     int `json:"total,omitempty"`
}

// CommitFile describes a single file in the diff of a commit. The same
// shape is reused by CommitFileList and by the compare endpoint's
// files array.
type CommitFile struct {
	Filename  string `json:"filename"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
	Status    string `json:"status,omitempty"` // added | modified | removed | renamed
	Patch     string `json:"patch,omitempty"`  // unified diff fragment
	Previous  string `json:"previous_filename,omitempty"`
}

// Commit is the response shape of both the commit list and detail
// endpoints. List responses populate everything except `Stats` and
// `Files`; the detail endpoint fills them in.
type Commit struct {
	SHA       string       `json:"sha"`
	URL       string       `json:"url"`
	HTMLURL   string       `json:"html_url"`
	Commit    *CommitInner `json:"commit"`
	Author    *User        `json:"author,omitempty"`     // Gitee user (may be null for unlinked emails)
	Committer *User        `json:"committer,omitempty"`  // Gitee user (may be null)
	Parents   []CommitRef  `json:"parents,omitempty"`   // parent commit refs (typically 0..n)
	Stats     *CommitStats `json:"stats,omitempty"`     // detail endpoint only
	Files     []CommitFile `json:"files,omitempty"`     // detail endpoint only
	Message   string       `json:"message,omitempty"`   // convenience: top-level message if backend echoes it
}

// CommitInner is the nested git-metadata object Gitee embeds in each
// commit payload. Carries the canonical message, author/committer git
// identity (name+email+date), and the tree SHA.
type CommitInner struct {
	Author    CommitAuthor `json:"author"`
	Committer CommitAuthor `json:"committer"`
	Message   string       `json:"message"`
	Tree      *CommitRef   `json:"tree,omitempty"`
	CommentCount int       `json:"comment_count,omitempty"`
}

// CommitRef is a minimal pointer used in parents[] and tree. The same
// shape applies to both.
type CommitRef struct {
	SHA string `json:"sha"`
	URL string `json:"url,omitempty"`
}

// CompareCommit is the lightweight commit shape Gitee returns inside the
// compare response under commits[]. It is a subset of Commit: same
// author/committer/message/url but no stats/files.
type CompareCommit struct {
	SHA       string       `json:"sha"`
	URL       string       `json:"url"`
	HTMLURL   string       `json:"html_url"`
	Commit    *CommitInner `json:"commit"`
	Author    *User        `json:"author,omitempty"`
	Committer *User        `json:"committer,omitempty"`
	Parents   []CommitRef  `json:"parents,omitempty"`
	Message   string       `json:"message,omitempty"`
}

// CompareResult is the response shape of
// GET /repos/{owner}/{repo}/compare/{base}...{head}. It bundles:
//
//   - status:       "ahead" / "behind" / "identical" / "diverged"
//   - ahead_by / behind_by:  commit counts
//   - total_commits:        len(commits)
//   - base_commit / merge_base_commit:  the resolved ref metadata
//   - commits:              the list of commits unique to `head`
//   - files:                the aggregated file diff
type CompareResult struct {
	Status            string           `json:"status"`              // ahead | behind | identical | diverged
	AheadBy           int              `json:"ahead_by"`
	BehindBy          int              `json:"behind_by"`
	TotalCommits      int              `json:"total_commits"`
	BaseCommit        *Commit          `json:"base_commit,omitempty"`
	MergeBaseCommit   *Commit          `json:"merge_base_commit,omitempty"`
	Commits           []CompareCommit  `json:"commits,omitempty"`
	Files             []CommitFile     `json:"files,omitempty"`
	HTMLURL           string           `json:"html_url,omitempty"`
	Permalink         string           `json:"permalink,omitempty"`
	DiffURL           string           `json:"diff_url,omitempty"`
	PatchURL          string           `json:"patch_url,omitempty"`
}