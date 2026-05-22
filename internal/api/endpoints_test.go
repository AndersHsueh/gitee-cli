package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPRsReviewEndpoint(t *testing.T) {
	tests := []struct {
		name   string
		owner  string
		repo   string
		number int
		want   string
	}{
		{"standard", "owner", "repo", 1, "/repos/owner/repo/pulls/1/review"},
		{"org repo", "gitee-cli", "gitee", 42, "/repos/gitee-cli/gitee/pulls/42/review"},
		{"large number", "user", "project", 9999, "/repos/user/project/pulls/9999/review"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PRsReview(tt.owner, tt.repo, tt.number)
			assert.Equal(t, tt.want, got)
		})
	}
}