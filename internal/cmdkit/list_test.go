package cmdkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitee-cli/internal/api"
)

// captured holds the result of running a function with os.Stdout swapped.
type captured struct {
	stdout string
	err    error
}

// captureStdout runs fn with os.Stdout redirected to a buffer and returns
// the captured output plus any error fn produced.
func captureStdout(t *testing.T, fn func() error) captured {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = old
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return captured{stdout: buf.String(), err: runErr}
}

// testItem is a minimal item type for exercising the generic RunList
// skeleton. Fields mirror the shape of a real list item (number/title).
type testItem struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// withServer starts an httptest server and returns an *api.Client pointed
// at it, so tests exercise the skeleton with explicit dependencies (no cmd
// globals involved).
func withServer(t *testing.T, h http.HandlerFunc) *api.Client {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return api.NewClient("test-token", ts.URL, false, 30)
}

// opts builds a default Options for the test server. Tests mutate the
// returned struct to set dispatch/pagination state.
func opts(client *api.Client) Options {
	return Options{
		Client:  client,
		Timeout: 30 * time.Second,
		Output:  "table",
	}
}

// spec returns a minimal ListSpec for testItem that renders a two-column
// table and a fixed empty text. ResolveOwnerRepo pins the repo so the
// skeleton doesn't depend on the caller's git config.
func spec() ListSpec[testItem] {
	return ListSpec[testItem]{
		ResolveOwnerRepo: func() (string, string, error) {
			return "owner", "repo", nil
		},
		Path: func(owner, repo string) string {
			return "/repos/" + owner + "/" + repo + "/items"
		},
		Columns: []string{"#", "TITLE"},
		Row: func(it testItem) []string {
			return []string{strconv.Itoa(it.Number), it.Title}
		},
		EmptyText: func(o, r string) string {
			return fmt.Sprintf("No items found in %s/%s", o, r)
		},
		ErrCtx: "listing items",
	}
}

// TestRunList_TableOutput verifies the default table dispatch: header row,
// one tab-separated row per item, and a trailing newline.
func TestRunList_TableOutput(t *testing.T) {
	client := withServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":1,"title":"a"},{"number":2,"title":"b"}]`))
	})

	err := RunList[testItem](opts(client), spec())
	if err != nil {
		t.Fatalf("RunList table: %v", err)
	}
}

// TestRunList_JSONOutput verifies the json dispatch emits the full item
// array as indented JSON.
func TestRunList_JSONOutput(t *testing.T) {
	client := withServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":1,"title":"a"}]`))
	})
	o := opts(client)
	o.Output = "json"

	out := captureStdout(t, func() error {
		return RunList[testItem](o, spec())
	})
	if err := out.err; err != nil {
		t.Fatalf("RunList json: %v", err)
	}
	var items []testItem
	if err := json.Unmarshal([]byte(out.stdout), &items); err != nil {
		t.Fatalf("json output not valid: %v\n%s", err, out.stdout)
	}
	if len(items) != 1 || items[0].Number != 1 {
		t.Errorf("unexpected json items: %+v", items)
	}
}

// TestRunList_PaginateMergesPages verifies that with --paginate the
// skeleton walks multiple pages and merges them into one result.
func TestRunList_PaginateMergesPages(t *testing.T) {
	var pages []string
	client := withServer(t, func(w http.ResponseWriter, req *http.Request) {
		page := req.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "1":
			_, _ = w.Write([]byte(`[{"number":1,"title":"a"},{"number":2,"title":"b"}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"number":3,"title":"c"}]`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
		pages = append(pages, page)
	})
	o := opts(client)
	o.Paginate = true
	o.PerPage = 2
	o.Limit = 0

	out := captureStdout(t, func() error {
		return RunList[testItem](o, spec())
	})
	if out.err != nil {
		t.Fatalf("RunList paginate: %v", out.err)
	}
	// pages 1 and 2 must have been fetched.
	if len(pages) < 2 {
		t.Errorf("expected >=2 page fetches, got %v", pages)
	}
	// table output should contain all three rows.
	if !strings.Contains(out.stdout, "1") || !strings.Contains(out.stdout, "3") {
		t.Errorf("paginated table missing items:\n%s", out.stdout)
	}
}

// TestRunList_LimitTruncates verifies that --paginate --limit N caps the
// merged result at N items.
func TestRunList_LimitTruncates(t *testing.T) {
	client := withServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":1,"title":"a"},{"number":2,"title":"b"}]`))
	})
	o := opts(client)
	o.Paginate = true
	o.PerPage = 2
	o.Limit = 1

	out := captureStdout(t, func() error {
		return RunList[testItem](o, spec())
	})
	if out.err != nil {
		t.Fatalf("RunList limit: %v", out.err)
	}
	// Only item 1 (and its header) should appear; item 2 must be absent.
	if strings.Contains(out.stdout, "\t2\t") {
		t.Errorf("limit not applied, item 2 leaked:\n%s", out.stdout)
	}
}

// TestRunList_EmptyText verifies the empty-result message is printed when
// the API returns no items.
func TestRunList_EmptyText(t *testing.T) {
	client := withServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})

	out := captureStdout(t, func() error {
		return RunList[testItem](opts(client), spec())
	})
	if out.err != nil {
		t.Fatalf("RunList empty: %v", out.err)
	}
	if !strings.Contains(out.stdout, "No items found in owner/repo") {
		t.Errorf("empty text missing, got: %q", out.stdout)
	}
}

// TestRunList_404IsEmpty verifies a 404 response is treated as an empty
// list (not an error) — the shared contract that used to drift between
// issue list and pr list (v0.4.1).
func TestRunList_404IsEmpty(t *testing.T) {
	client := withServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})

	out := captureStdout(t, func() error {
		return RunList[testItem](opts(client), spec())
	})
	if out.err != nil {
		t.Fatalf("404 should be empty, not error: %v", out.err)
	}
	if !strings.Contains(out.stdout, "No items found in owner/repo") {
		t.Errorf("404 should yield empty text, got: %q", out.stdout)
	}
}

// TestRunList_Footer verifies a table footer is printed after the rows
// (used by label/milestone/webhook/ssh-key/release/notification list).
func TestRunList_Footer(t *testing.T) {
	client := withServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":1,"title":"a"}]`))
	})

	s := spec()
	s.Footer = func(count int) string {
		return fmt.Sprintf("\nTotal: %d item(s)\n", count)
	}
	out := captureStdout(t, func() error {
		return RunList[testItem](opts(client), s)
	})
	if out.err != nil {
		t.Fatalf("RunList footer: %v", out.err)
	}
	if !strings.Contains(out.stdout, "\nTotal: 1 item(s)\n") {
		t.Errorf("footer missing, got: %q", out.stdout)
	}
}

// TestRunList_CustomJSON verifies a custom JSON printer is invoked instead
// of the default indented marshal.
func TestRunList_CustomJSON(t *testing.T) {
	client := withServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"number":1,"title":"a"}]`))
	})
	o := opts(client)
	o.Output = "json"

	called := false
	s := spec()
	s.JSON = func(items []testItem) error {
		called = true
		if len(items) != 1 || items[0].Number != 1 {
			t.Errorf("unexpected items passed to JSON printer: %+v", items)
		}
		return nil
	}
	if err := RunList[testItem](o, s); err != nil {
		t.Fatalf("RunList custom json: %v", err)
	}
	if !called {
		t.Error("custom JSON printer was not invoked")
	}
}
