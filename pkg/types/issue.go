package types

import "time"

// User is now in user.go

type Label struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	URL   string `json:"url"`
}

type Milestone struct {
	ID     int64  `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

type Issue struct {
	ID        int64      `json:"id"`
	Number    string     `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	HTMLURL   string     `json:"html_url"`
	User      User       `json:"user"`
	Assignee  *User      `json:"assignee"`
	Labels    []Label    `json:"labels"`
	Milestone *Milestone `json:"milestone"`
	Comments  int        `json:"comments"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"finished_at"`
}

type IssueCreateRequest struct {
	Repo            string `json:"-"`
	Title           string `json:"title"`
	Body            string `json:"body,omitempty"`
	Labels          string `json:"labels,omitempty"`
	Assignees       string `json:"assignees,omitempty"`
	MilestoneNumber int    `json:"milestone_number,omitempty"`
}

type IssueEditRequest struct {
	Title     string `json:"title,omitempty"`
	Body      string `json:"body,omitempty"`
	Labels    string `json:"labels,omitempty"`
	Assignees string `json:"assignees,omitempty"`
	Milestone int    `json:"milestone_number,omitempty"`
	State     string `json:"state,omitempty"`
}

type IssueCommentRequest struct {
	Body string `json:"body"`
}
