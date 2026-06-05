package types

import "time"

// Webhook represents a Gitee repository webhook (also called "Hook" in the
// upstream swagger schema). Gitee returns webhooks as flat objects — there is
// no nested project/user; project_id is a scalar on the response.
//
// Gitee does not expose an `active` toggle or an `updated_at` field. To mute
// a webhook, PATCH every event boolean to false (or DELETE the hook outright).
// To rotate the secret, PATCH `password` (with `encryption_type=0`).
type Webhook struct {
	ID                 int64     `json:"id"`
	URL                string    `json:"url"` // delivery URL (not a Gitee browse URL)
	Password           string    `json:"password,omitempty"`
	Result             string    `json:"result,omitempty"`
	ProjectID          int64     `json:"project_id,omitempty"`
	ResultCode         int       `json:"result_code,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	PushEvents         bool      `json:"push_events"`
	TagPushEvents      bool      `json:"tag_push_events"`
	IssuesEvents       bool      `json:"issues_events"`
	NoteEvents         bool      `json:"note_events"`
	MergeRequestEvents bool      `json:"merge_requests_events"`
	Title              string    `json:"title,omitempty"`
}

// WebhookCreateRequest is the body for POST /repos/{owner}/{repo}/hooks.
// Only `URL` is required. Event booleans are pointers so that "do not send"
// is distinguishable from "send false" — important because the server
// defaults `push_events` to true on create.
type WebhookCreateRequest struct {
	URL                string `json:"url"`
	Title              string `json:"title,omitempty"`
	EncryptionType     *int   `json:"encryption_type,omitempty"` // 0=password, 1=signing key
	Password           string `json:"password,omitempty"`
	PushEvents         *bool  `json:"push_events,omitempty"`
	TagPushEvents      *bool  `json:"tag_push_events,omitempty"`
	IssuesEvents       *bool  `json:"issues_events,omitempty"`
	NoteEvents         *bool  `json:"note_events,omitempty"`
	MergeRequestEvents *bool  `json:"merge_requests_events,omitempty"`
}

// WebhookEditRequest is the body for PATCH /repos/{owner}/{repo}/hooks/{id}.
// Gitee's swagger spec marks `url` as required on PATCH too, so callers must
// supply the existing URL alongside any fields they want to change.
type WebhookEditRequest struct {
	URL                string `json:"url"`
	Title              string `json:"title,omitempty"`
	EncryptionType     *int   `json:"encryption_type,omitempty"`
	Password           string `json:"password,omitempty"`
	PushEvents         *bool  `json:"push_events,omitempty"`
	TagPushEvents      *bool  `json:"tag_push_events,omitempty"`
	IssuesEvents       *bool  `json:"issues_events,omitempty"`
	NoteEvents         *bool  `json:"note_events,omitempty"`
	MergeRequestEvents *bool  `json:"merge_requests_events,omitempty"`
}
