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

func TestReleaseEndpoints(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"list", ReleasesList("owner", "repo"), "/repos/owner/repo/releases"},
		{"get", ReleasesGet("owner", "repo", "v1.0.0"), "/repos/owner/repo/releases/tags/v1.0.0"},
		{"create", ReleaseCreate("owner", "repo"), "/repos/owner/repo/releases"},
		{"edit", ReleaseEdit("owner", "repo", "v1.0.0"), "/repos/owner/repo/releases/tags/v1.0.0"},
		{"delete", ReleaseDelete("owner", "repo", "v1.0.0"), "/repos/owner/repo/releases/tags/v1.0.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}

func TestRepoLifecycleEndpoints(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"archive", ReposArchive("owner", "repo"), "/repos/owner/repo"},
		{"rename", ReposRename("owner", "repo"), "/repos/owner/repo"},
		{"sync", ReposSync("owner", "repo"), "/repos/owner/repo/pull"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}

func TestSSHKeyEndpoints(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"list", SSHKeysList(), "/user/keys"},
		{"create", SSHKeyCreate(), "/user/keys"},
		{"delete", SSHKeyDelete(42), "/user/keys/42"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}


func TestWebhookEndpoints(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"list", WebhooksList("owner", "repo"), "/repos/owner/repo/hooks"},
		{"get", WebhookGet("owner", "repo", 42), "/repos/owner/repo/hooks/42"},
		{"create", WebhookCreate("owner", "repo"), "/repos/owner/repo/hooks"},
		{"edit", WebhookEdit("owner", "repo", 42), "/repos/owner/repo/hooks/42"},
		{"delete", WebhookDelete("owner", "repo", 42), "/repos/owner/repo/hooks/42"},
		{"test", WebhookTest("owner", "repo", 42), "/repos/owner/repo/hooks/42/tests"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}

func TestIssueCommentsListEndpoint(t *testing.T) {
	cases := []struct {
		name, got, want string
	}{
		{"simple", IssueCommentsList("owner", "repo", "42"), "/repos/owner/repo/issues/42/comments"},
		{"org repo", IssueCommentsList("gitee-cli", "gitee", "IJP83Z"), "/repos/gitee-cli/gitee/issues/IJP83Z/comments"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}

func TestPRsEditEndpoint(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"standard", PRsEdit("owner", "repo", 1), "/repos/owner/repo/pulls/1"},
		{"org repo", PRsEdit("gitee-cli", "gitee", 42), "/repos/gitee-cli/gitee/pulls/42"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}

func TestPRsChecksEndpoint(t *testing.T) {
	cases := []struct {
		name, got, want string
	}{
		{"simple", PRsChecks("owner", "repo", "main"), "/repos/owner/repo/commits/main/check-runs"},
		{"with slash", PRsChecks("owner", "repo", "feature/x"), "/repos/owner/repo/commits/feature%2Fx/check-runs"},
		{"sha", PRsChecks("owner", "repo", "abc123def"), "/repos/owner/repo/commits/abc123def/check-runs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}

func TestReleaseAssetEndpoints(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"upload", ReleaseAssetUpload("owner", "repo", 42), "/repos/owner/repo/releases/42/assets"},
		{"delete", ReleaseDeleteAsset("owner", "repo", 42, 7), "/repos/owner/repo/releases/42/assets/7"},
		{"org upload", ReleaseAssetUpload("gitee-cli", "gitee", 9999), "/repos/gitee-cli/gitee/releases/9999/assets"},
		{"org delete", ReleaseDeleteAsset("gitee-cli", "gitee", 100, 200), "/repos/gitee-cli/gitee/releases/100/assets/200"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got)
		})
	}
}
