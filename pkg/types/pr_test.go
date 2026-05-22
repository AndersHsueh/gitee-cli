package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPRReviewRequest_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		request  PRReviewRequest
		wantJSON string
	}{
		{
			name:     "approve",
			request:  PRReviewRequest{Body: "LGTM", Event: "approve"},
			wantJSON: `"event":"approve"`,
		},
		{
			name:     "request changes",
			request:  PRReviewRequest{Body: "Needs work", Event: "request_changes"},
			wantJSON: `"event":"request_changes"`,
		},
		{
			name:     "comment only",
			request:  PRReviewRequest{Body: "Nice!", Event: "comment"},
			wantJSON: `"event":"comment"`,
		},
		{
			name:     "empty body",
			request:  PRReviewRequest{Event: "approve"},
			wantJSON: `"event":"approve"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			require.NoError(t, err)
			assert.Contains(t, string(data), tt.wantJSON)
		})
	}
}

func TestPRReviewRequest_Unmarshal(t *testing.T) {
	raw := `{"event":"approve","body":"LGTM"}`
	var req PRReviewRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)
	assert.Equal(t, "approve", req.Event)
	assert.Equal(t, "LGTM", req.Body)
}