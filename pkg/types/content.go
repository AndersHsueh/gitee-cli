package types

// ContentEntry is the response shape of GET /repos/{owner}/{repo}/contents/{path}.
//
// Gitee v5 has the same GitHub-style duality: a single file is returned as
// one object with a "type" of "file", a directory is returned as an array
// of these objects, and a 404 indicates the path does not exist. The JSON
// fields below are aligned with the swagger spec and the GitHub Contents
// API that Gitee mirrors.
type ContentEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`           // blob SHA (empty for submodules)
	Size        int    `json:"size"`          // file size in bytes
	Type        string `json:"type"`          // "file" | "dir" | "symlink" | "submodule"
	Encoding    string `json:"encoding"`      // for files: "base64" (Gitee always base64-encodes)
	Content     string `json:"content"`       // base64-encoded file body; empty for dirs
	Target      string `json:"target"`        // symlink target (when type=symlink)
	URL         string `json:"url"`           // API URL
	HTMLURL     string `json:"html_url"`      // web URL
	DownloadURL string `json:"download_url"`  // raw file URL
	Links       struct {
		Self    string `json:"self"`
		Git     string `json:"git"`
		HTML    string `json:"html"`
		Raw     string `json:"raw,omitempty"`
	} `json:"_links"`
}

// ContentCreateRequest is the body for POST .../contents/{path}.
// `Content` must be base64-encoded; `Message` is the commit message; the
// optional `Branch` field targets a non-default branch. SHA is NOT used
// on create (a new file has no prior blob SHA).
type ContentCreateRequest struct {
	Content    string `json:"content"`               // base64-encoded
	Message    string `json:"message"`               // commit message (required)
	Branch     string `json:"branch,omitempty"`      // target branch
	Author     string `json:"author,omitempty"`      // author name (uses token owner if blank)
	AuthorEmail string `json:"author_email,omitempty"`
	Comitter   string `json:"committer,omitempty"`
	ComitterEmail string `json:"committer_email,omitempty"`
}

// ContentUpdateRequest is the body for PUT .../contents/{path}. Differs
// from the create request only by the required `SHA` field (the blob SHA
// of the file being replaced — used for optimistic concurrency).
type ContentUpdateRequest struct {
	Content    string `json:"content"`
	Message    string `json:"message"`
	SHA        string `json:"sha"`                   // blob SHA of existing file
	Branch     string `json:"branch,omitempty"`
	Author     string `json:"author,omitempty"`
	AuthorEmail string `json:"author_email,omitempty"`
	Comitter   string `json:"committer,omitempty"`
	ComitterEmail string `json:"committer_email,omitempty"`
}

// ContentDeleteRequest is the body for DELETE .../contents/{path}.
// Only SHA + Message are required; the author/email fields are honored
// when present.
type ContentDeleteRequest struct {
	SHA            string `json:"sha"`
	Message        string `json:"message"`
	Branch         string `json:"branch,omitempty"`
	Author         string `json:"author,omitempty"`
	AuthorEmail    string `json:"author_email,omitempty"`
	Comitter       string `json:"committer,omitempty"`
	ComitterEmail  string `json:"committer_email,omitempty"`
}

// ContentCommit is the nested "commit" object Gitee returns alongside the
// updated ContentEntry after a successful POST/PUT/DELETE on
// /contents/{path}. It is the same shape as a Commit's `commit` object
// but trimmed to the fields the CLI surfaces.
type ContentCommit struct {
	SHA     string `json:"sha"`
	URL     string `json:"url"`
	HTMLURL string `json:"html_url"`
	Message string `json:"message"`
	Author  struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Username string `json:"username,omitempty"`
		Date     string `json:"date"`
	} `json:"author"`
	Committer struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Username string `json:"username,omitempty"`
		Date     string `json:"date"`
	} `json:"committer"`
}

// ContentResponse is the response shape from POST/PUT/DELETE on
// .../contents/{path}. Gitee returns the resulting ContentEntry together
// with a ContentCommit describing the commit that was created. We use a
// single struct so the caller gets both fields without parsing twice.
type ContentResponse struct {
	Content ContentEntry `json:"content"`
	Commit  ContentCommit `json:"commit"`
}
