package types

import "time"

// NotificationThread is a single notification thread returned by
// GET /notifications/threads. Gitee's payload is GitHub-compatible:
//
//   {
//     "id": "1234",
//     "unread": true,
//     "reason": "subscribed",
//     "updated_at": "...",
//     "last_read_at": "...",
//     "subject": {
//       "title": "...",
//       "url": "...",
//       "latest_comment_url": "..."
//       "type": "Issue"   // or "PullRequest", "Commit", "Release"
//     },
//     "repository": { ... full repo object ... }
//   }
type NotificationThread struct {
	ID         string                    `json:"id"`
	Unread     bool                      `json:"unread"`
	Reason     string                    `json:"reason"`
	UpdatedAt  time.Time                 `json:"updated_at"`
	LastReadAt *time.Time                `json:"last_read_at,omitempty"`
	Subject    NotificationSubject       `json:"subject"`
	Repository NotificationRepoReference `json:"repository"`
}

// NotificationSubject is the embedded "what is the notification about"
// object inside a thread.
type NotificationSubject struct {
	Title            string `json:"title"`
	URL              string `json:"url"`
	LatestCommentURL string `json:"latest_comment_url,omitempty"`
	Type             string `json:"type"` // "Issue" | "PullRequest" | "Commit" | "Release"
}

// NotificationRepoReference is the lightweight repo metadata embedded
// inside each thread. We only deserialize the fields a CLI consumer is
// likely to display or filter on; the full repo object is not needed
// for `notification list`.
type NotificationRepoReference struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

// NotificationUnreadCount is the body of GET /notifications/count.
// Gitee returns a bare integer in some deployments and an object in
// others; we always wrap so the response shape is uniform.
type NotificationUnreadCount struct {
	Count int `json:"count"`
}

// NotificationMarkReadRequest is the (optional) body for
// PUT /notifications/threads/{id}. Sending an empty body marks the
// thread read with the server-side timestamp; the optional `unread`
// flag can be used to flip state back to unread.
type NotificationMarkReadRequest struct {
	Unread bool `json:"unread,omitempty"`
}

// NotificationMarkAllReadRequest is the body for
// PUT /notifications/threads (no id). `unread=true` marks all as read;
// `last_read_at` scopes the bulk update to threads updated since that
// timestamp. Both fields are optional — an empty body marks all
// currently-unread threads as read at server-side timestamp.
type NotificationMarkAllReadRequest struct {
	Unread     bool       `json:"unread,omitempty"`
	LastReadAt *time.Time `json:"last_read_at,omitempty"`
}