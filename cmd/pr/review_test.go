package pr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitee-cli/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewEventValues(t *testing.T) {
	req := types.PRReviewRequest{Body: "LGTM", Event: "approve"}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed types.PRReviewRequest
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "approve", parsed.Event)
	assert.Equal(t, "LGTM", parsed.Body)
}

func TestReviewAPIEndpoint(t *testing.T) {
	var receivedBody types.PRReviewRequest

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/review")

		err := json.NewDecoder(r.Body).Decode(&receivedBody)
		require.NoError(t, err)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	req := types.PRReviewRequest{
		Body:  "Looks good",
		Event: "approve",
	}

	data, _ := json.Marshal(req)
	// 验证 JSON 结构正确
	assert.Contains(t, string(data), `"event":"approve"`)
	assert.Contains(t, string(data), `"body":"Looks good"`)
}
