package types

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

// GistEditRequest is used when editing a gist.
type GistEditRequest struct {
	Description string `json:"description,omitempty"`
	Files       map[string]struct {
		Content  string `json:"content,omitempty"`
		Filename string `json:"filename,omitempty"`
	} `json:"files,omitempty"`
}
