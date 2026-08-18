package notification

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- agent help content (per-verb) -----

func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee notification list",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(notificationListAgentHelp, want) {
			t.Errorf("list agent help missing required section: %q", want)
		}
	}
}

func TestCountAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee notification count",
		"## Purpose",
		"## Pre-conditions",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(notificationCountAgentHelp, want) {
			t.Errorf("count agent help missing required section: %q", want)
		}
	}
}

func TestReadAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee notification read",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(notificationReadAgentHelp, want) {
			t.Errorf("read agent help missing required section: %q", want)
		}
	}
}

// ----- wire-format + endpoint contract tests (httptest) -----

// TestNotificationsListEndpoint verifies GET /notifications/threads
// with no filters — response shape, path, and method.
func TestNotificationsListEndpoint(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotQuery   string
		gotAuthHdr string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuthHdr = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"id":"1234",
				"unread":true,
				"reason":"subscribed",
				"updated_at":"2026-07-04T09:12:00+08:00",
				"subject":{
					"title":"API: invoice webhooks time out",
					"url":"https://gitee.com/myorg/billing-svc/issues/42",
					"type":"Issue"
				},
				"repository":{
					"id":99,
					"full_name":"myorg/billing-svc",
					"html_url":"https://gitee.com/myorg/billing-svc"
				}
			}
		]`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	var threads []types.NotificationThread
	err := client.Get(t.Context(), api.NotificationsList(), &threads)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/notifications/threads", gotPath)
	assert.Equal(t, "", gotQuery)
	assert.Equal(t, "token test-token", gotAuthHdr)
	require.Len(t, threads, 1)
	assert.Equal(t, "1234", threads[0].ID)
	assert.True(t, threads[0].Unread)
	assert.Equal(t, "subscribed", threads[0].Reason)
	assert.Equal(t, "Issue", threads[0].Subject.Type)
	assert.Equal(t, "myorg/billing-svc", threads[0].Repository.FullName)
	assert.Equal(t, int64(99), threads[0].Repository.ID)
}

// TestNotificationsCountEndpoint verifies GET /notifications/count.
func TestNotificationsCountEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":7}`))
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	var c types.NotificationUnreadCount
	err := client.Get(t.Context(), api.NotificationsCount(), &c)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/notifications/count", gotPath)
	assert.Equal(t, 7, c.Count)
}

// TestNotificationMarkReadEndpoint verifies PUT
// /notifications/threads/{id} — request body shape and the path used.
func TestNotificationMarkReadEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   types.NotificationMarkReadRequest
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	err := client.Put(t.Context(), api.NotificationMarkRead("1234"),
		types.NotificationMarkReadRequest{Unread: false}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/notifications/threads/1234", gotPath)
	// Unread is omitted from the wire payload when false (omitempty).
	assert.False(t, gotBody.Unread)
}

// TestNotificationsMarkAllReadEndpoint verifies PUT /notifications/threads
// (no id in the path) with an explicit bulk-unread body.
func TestNotificationsMarkAllReadEndpoint(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   types.NotificationMarkAllReadRequest
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := api.NewClient("test-token", ts.URL, false, 30)

	err := client.Put(t.Context(), api.NotificationsMarkAllRead(),
		types.NotificationMarkAllReadRequest{Unread: true}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/notifications/threads", gotPath)
	assert.True(t, gotBody.Unread)
}

// TestNotificationsTypeRoundTrip verifies the wire shape of the
// request payloads and the unmarshal of a full thread object.
func TestNotificationsTypeRoundTrip(t *testing.T) {
	// mark-read-single: empty body when Unread is false → omitempty
	r1 := types.NotificationMarkReadRequest{Unread: false}
	data, err := json.Marshal(r1)
	require.NoError(t, err)
	// body should still serialize as `{}` (no unread field present)
	assert.Equal(t, "{}", string(data))

	// mark-read-single: unread=true → explicit true
	r2 := types.NotificationMarkReadRequest{Unread: true}
	data, err = json.Marshal(r2)
	require.NoError(t, err)
	assert.Equal(t, `{"unread":true}`, string(data))

	// mark-all-read: optional fields dropped when zero
	r3 := types.NotificationMarkAllReadRequest{}
	data, err = json.Marshal(r3)
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))

	// Full thread unmarshal (mirrors Gitee payload).
	raw := `{
		"id":"1234",
		"unread":true,
		"reason":"mention",
		"updated_at":"2026-07-04T09:12:00+08:00",
		"last_read_at":"2026-07-03T18:00:00+08:00",
		"subject":{
			"title":"feat(notification): add cmd group",
			"url":"https://gitee.com/andershsueh/gitee-cli/issues/1",
			"latest_comment_url":"https://gitee.com/andershsueh/gitee-cli/issues/1#comment-1",
			"type":"Issue"
		},
		"repository":{
			"id":42,
			"full_name":"andershsueh/gitee-cli",
			"html_url":"https://gitee.com/andershsueh/gitee-cli"
		}
	}`
	var th types.NotificationThread
	require.NoError(t, json.Unmarshal([]byte(raw), &th))
	assert.Equal(t, "1234", th.ID)
	assert.True(t, th.Unread)
	assert.Equal(t, "mention", th.Reason)
	assert.Equal(t, "Issue", th.Subject.Type)
	assert.Equal(t, "feat(notification): add cmd group", th.Subject.Title)
	assert.Equal(t, int64(42), th.Repository.ID)
	assert.Equal(t, "andershsueh/gitee-cli", th.Repository.FullName)
	require.NotNil(t, th.LastReadAt)
	assert.Equal(t, 2026, th.LastReadAt.Year())
}

// resetReadFlags restores readCmd flag state between tests. Cobra's
// parsed flag values persist on the command after Execute() returns,
// so the package-level readFlags would otherwise leak across tests.
func resetReadFlags() {
	readCmd.SetArgs(nil)
	readFlags.all = false
	readFlags.unread = false
	readFlags.force = false
}

// TestNotificationReadCmdMutualExclusion verifies the Cobra wrapper
// rejects the combination of a positional id AND --all.
func TestNotificationReadCmdMutualExclusion(t *testing.T) {
	resetReadFlags()
	defer resetReadFlags()
	NotificationCmd.SetArgs([]string{"read", "some-id", "--all", "--force"})
	err := NotificationCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

// TestNotificationReadCmdRequiresIdOrAll verifies the Cobra wrapper
// rejects the empty case (no id, no --all).
func TestNotificationReadCmdRequiresIdOrAll(t *testing.T) {
	resetReadFlags()
	defer resetReadFlags()
	NotificationCmd.SetArgs([]string{"read", "--force"})
	err := NotificationCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

// TestNotificationFlagsCoveredInAgentHelp ensures every flag exposed
// by the notification subcommands is also documented in its agent help.
func TestNotificationFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// list
		{"--unread", notificationListAgentHelp},
		{"--read", notificationListAgentHelp},
		{"--limit", notificationListAgentHelp},
		{"-L", notificationListAgentHelp},

		// read
		{"--all", notificationReadAgentHelp},
		{"--unread", notificationReadAgentHelp},
		{"--force", notificationReadAgentHelp},
		{"-f", notificationReadAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}

// withAPIClientGlobals returns a Runtime bound to the httptest server
// so domain clients built from it hit the mock instead of gitee.com.
func withAPIClientGlobals(t *testing.T, serverURL string) *cmd.Runtime {
	t.Helper()
	return &cmd.Runtime{
		APIURL:  serverURL,
		Token:   "test-token",
		Timeout: 30,
	}
}

// captureStdout temporarily redirects os.Stdout to a pipe and returns a
// restore func plus a drain func that yields the captured text. The
// caller must call drain() after the code under test has finished.
func captureStdout(t *testing.T) (restore func(), drain func() string) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	restore = func() { os.Stdout = old }
	drain = func() string {
		_ = w.Close()
		b, _ := io.ReadAll(r)
		return string(b)
	}
	return restore, drain
}

// TestNotificationReadUnreadDirectionOutput verifies the --unread flag
// drives the SUCCESS output direction: a single-thread mark-read with
// markUnread=true must print "marked as UNREAD" (not "as read"). This
// is the post-confirmation mirror of the prompt-direction fix and
// locks the direction variable used in both the prompt and the output.
func TestNotificationReadUnreadDirectionOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	rt := withAPIClientGlobals(t, ts.URL)

	restoreOut, drain := captureStdout(t)
	dc, err := cmd.NewDomainClientFrom(rt)
	require.NoError(t, err)
	err = dc.NotificationRead("1234", false, true /*markUnread*/, true /*force*/)
	out := drain()
	restoreOut()

	require.NoError(t, err)
	assert.Contains(t, out, "Notification 1234 marked as UNREAD")
	assert.NotContains(t, out, "marked as read")
}

// TestNotificationReadDefaultDirectionOutput verifies the default
// (markUnread=false) direction prints "marked as read".
func TestNotificationReadDefaultDirectionOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	rt := withAPIClientGlobals(t, ts.URL)

	restoreOut, drain := captureStdout(t)
	dc, err := cmd.NewDomainClientFrom(rt)
	require.NoError(t, err)
	err = dc.NotificationRead("1234", false, false /*markUnread*/, true /*force*/)
	out := drain()
	restoreOut()

	require.NoError(t, err)
	assert.Contains(t, out, "Notification 1234 marked as read")
}

// TestNotificationReadAllUnreadDirectionOutput verifies the bulk (--all)
// path also reflects the --unread direction in its output.
func TestNotificationReadAllUnreadDirectionOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	rt := withAPIClientGlobals(t, ts.URL)

	restoreOut, drain := captureStdout(t)
	dc, err := cmd.NewDomainClientFrom(rt)
	require.NoError(t, err)
	err = dc.NotificationRead("", true /*all*/, true /*markUnread*/, true /*force*/)
	out := drain()
	restoreOut()

	require.NoError(t, err)
	assert.Contains(t, out, "All notifications marked as UNREAD")
	assert.NotContains(t, out, "marked as read")
}
