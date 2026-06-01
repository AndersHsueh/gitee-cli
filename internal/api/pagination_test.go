package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pageServer is a minimal test double: it returns `pageItems[page]` as JSON
// for each request, or an empty array if the page is beyond the data.
// It also asserts the request query params match what FetchAllPages should send.
type pageServer struct {
	pageItems map[int][]map[string]any // pageNum -> items
	perPage   int
	calls     int32 // atomic counter
}

func (s *pageServer) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&s.calls, 1)
		page := r.URL.Query().Get("page")
		perPage := r.URL.Query().Get("per_page")
		assert.Equal(t, fmt.Sprintf("%d", s.perPage), perPage, "per_page should match what caller requested")

		var pageNum int
		fmt.Sscanf(page, "%d", &pageNum)

		items, ok := s.pageItems[pageNum]
		if !ok {
			items = []map[string]any{} // empty = end of data
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	}
}

func mkItem(n int) map[string]any {
	return map[string]any{"id": n, "name": fmt.Sprintf("item-%d", n)}
}

func TestFetchAllPages_SinglePage(t *testing.T) {
	srv := &pageServer{
		pageItems: map[int][]map[string]any{
			1: {mkItem(1), mkItem(2), mkItem(3)},
		},
		perPage: 30,
	}
	client, ts := newTestClient(t, srv.handler(t))
	defer ts.Close()

	var got []map[string]any
	err := FetchAllPages(context.Background(), client, "/items", 30, 0, &got)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Equal(t, int32(1), atomic.LoadInt32(&srv.calls), "should make exactly 1 HTTP call")
}

func TestFetchAllPages_MultiplePages(t *testing.T) {
	// 3 full pages of 5 items each = 15 items total.
	// Gitee API's convention is that the LAST page may return fewer items
	// than per_page, so a full page means "might be more — try the next one".
	// Expect 4 calls: pages 1, 2, 3, then 4 (empty, probing for end-of-data).
	srv := &pageServer{
		pageItems: map[int][]map[string]any{
			1: {mkItem(1), mkItem(2), mkItem(3), mkItem(4), mkItem(5)},
			2: {mkItem(6), mkItem(7), mkItem(8), mkItem(9), mkItem(10)},
			3: {mkItem(11), mkItem(12), mkItem(13), mkItem(14), mkItem(15)},
			// page 4 absent = empty array = end of data (probe)
		},
		perPage: 5,
	}
	client, ts := newTestClient(t, srv.handler(t))
	defer ts.Close()

	var got []map[string]any
	err := FetchAllPages(context.Background(), client, "/items", 5, 0, &got)
	require.NoError(t, err)
	assert.Len(t, got, 15)
	assert.Equal(t, int32(4), atomic.LoadInt32(&srv.calls), "3 full pages + 1 empty probe")
}

func TestFetchAllPages_ShortLastPage(t *testing.T) {
	// 1 full page + 1 short page = 8 items total; should NOT make a 3rd call
	srv := &pageServer{
		pageItems: map[int][]map[string]any{
			1: {mkItem(1), mkItem(2), mkItem(3), mkItem(4), mkItem(5)},
			2: {mkItem(6), mkItem(7), mkItem(8)}, // 3 < perPage(5) = last page
		},
		perPage: 5,
	}
	client, ts := newTestClient(t, srv.handler(t))
	defer ts.Close()

	var got []map[string]any
	err := FetchAllPages(context.Background(), client, "/items", 5, 0, &got)
	require.NoError(t, err)
	assert.Len(t, got, 8)
	assert.Equal(t, int32(2), atomic.LoadInt32(&srv.calls), "should stop after short page, not probe for empty page 3")
}

func TestFetchAllPages_EmptyResult(t *testing.T) {
	srv := &pageServer{
		pageItems: map[int][]map[string]any{}, // no pages at all
		perPage:   30,
	}
	client, ts := newTestClient(t, srv.handler(t))
	defer ts.Close()

	var got []map[string]any
	err := FetchAllPages(context.Background(), client, "/items", 30, 0, &got)
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, int32(1), atomic.LoadInt32(&srv.calls), "should probe page 1 then stop")
}

func TestFetchAllPages_LimitCap(t *testing.T) {
	// 3 full pages of 10 = 30 items, but limit=12 should stop after page 2
	srv := &pageServer{
		pageItems: map[int][]map[string]any{
			1: {mkItem(1), mkItem(2), mkItem(3), mkItem(4), mkItem(5),
				mkItem(6), mkItem(7), mkItem(8), mkItem(9), mkItem(10)},
			2: {mkItem(11), mkItem(12), mkItem(13), mkItem(14), mkItem(15),
				mkItem(16), mkItem(17), mkItem(18), mkItem(19), mkItem(20)},
			3: {mkItem(21), mkItem(22), mkItem(23), mkItem(24), mkItem(25),
				mkItem(26), mkItem(27), mkItem(28), mkItem(29), mkItem(30)},
		},
		perPage: 10,
	}
	client, ts := newTestClient(t, srv.handler(t))
	defer ts.Close()

	var got []map[string]any
	err := FetchAllPages(context.Background(), client, "/items", 10, 12, &got)
	require.NoError(t, err)
	assert.Len(t, got, 12, "should stop at limit")
	assert.Equal(t, int32(2), atomic.LoadInt32(&srv.calls), "should not fetch page 3 after hitting limit")
}

func TestFetchAllPages_PreservesExistingQuery(t *testing.T) {
	// Caller passes ?state=open&labels=bug; FetchAllPages should keep those
	// and add page/per_page.
	var capturedQuery string
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	})
	client, ts := newTestClient(t, srv)
	defer ts.Close()

	var got []map[string]any
	err := FetchAllPages(context.Background(), client, "/issues?state=open&labels=bug", 30, 0, &got)
	require.NoError(t, err)
	assert.Contains(t, capturedQuery, "state=open")
	assert.Contains(t, capturedQuery, "labels=bug")
	assert.Contains(t, capturedQuery, "page=1")
	assert.Contains(t, capturedQuery, "per_page=30")
}

func TestFetchAllPages_ValidationErrors(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	defer ts.Close()
	ctx := context.Background()

	t.Run("perPage=0", func(t *testing.T) {
		var got []map[string]any
		err := FetchAllPages(ctx, client, "/items", 0, 0, &got)
		assert.Error(t, err)
	})
	t.Run("perPage=101", func(t *testing.T) {
		var got []map[string]any
		err := FetchAllPages(ctx, client, "/items", 101, 0, &got)
		assert.Error(t, err)
	})
	t.Run("negative limit", func(t *testing.T) {
		var got []map[string]any
		err := FetchAllPages(ctx, client, "/items", 30, -1, &got)
		assert.Error(t, err)
	})
	t.Run("nil result", func(t *testing.T) {
		var r *[]map[string]any // explicit nil pointer to help type inference
		err := FetchAllPages(ctx, client, "/items", 30, 0, r)
		assert.Error(t, err)
	})
	t.Run("nil client", func(t *testing.T) {
		var got []map[string]any
		err := FetchAllPages(ctx, nil, "/items", 30, 0, &got)
		assert.Error(t, err)
	})
}
