package api

import (
	"fmt"
	"net/url"
)

// Endpoint builders for Gitee API v5.

// Repos endpoints
func ReposList(owner string) string {
	if owner != "" {
		return fmt.Sprintf("/users/%s/repos", owner)
	}
	return "/user/repos"
}

func ReposGet(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s", owner, repo)
}

func ReposCreate() string {
	return "/user/repos"
}

func ReposCreateForOrg(org string) string {
	return fmt.Sprintf("/orgs/%s/repos", org)
}

func ReposFork(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/forks", owner, repo)
}

func ReposEdit(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s", owner, repo)
}

func ReposDelete(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s", owner, repo)
}

func ReposBranches(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/branches", owner, repo)
}

// Issues endpoints
func IssuesList(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
}

func IssuesGet(owner, repo, number string) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number)
}

func IssuesCreate(owner string) string {
	return fmt.Sprintf("/repos/%s/issues", owner)
}

func IssuesEdit(owner, repo, number string) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number)
}

func IssuesComment(owner, repo, number string) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%s/comments", owner, repo, number)
}

// Pull requests endpoints
func PRsList(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
}

func PRsGet(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
}

func PRsCreate(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
}

func PRsMerge(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, number)
}

func PRsReview(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d/review", owner, repo, number)
}

func PRsComments(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number)
}

// Gist endpoints
func GistsList() string {
	return "/gists"
}

func GistGet(gistID string) string {
	return fmt.Sprintf("/gists/%s", gistID)
}

func GistsCreate() string {
	return "/gists"
}

func GistEdit(gistID string) string {
	return fmt.Sprintf("/gists/%s", gistID)
}

func GistDelete(gistID string) string {
	return fmt.Sprintf("/gists/%s", gistID)
}

func GistStar(gistID string) string {
	return fmt.Sprintf("/gists/%s/star", gistID)
}

func GistUnstar(gistID string) string {
	return fmt.Sprintf("/gists/%s/star", gistID)
}

// Organization endpoints
func OrgsList() string {
	return "/user/orgs"
}

func OrgGet(org string) string {
	return fmt.Sprintf("/orgs/%s", org)
}

func OrgMembers(org string) string {
	return fmt.Sprintf("/orgs/%s/members", org)
}

func OrgRepos(org string) string {
	return fmt.Sprintf("/orgs/%s/repos", org)
}

func OrgIssues(org string) string {
	return fmt.Sprintf("/orgs/%s/issues", org)
}

// Label endpoints
func LabelList(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/labels", owner, repo)
}

func LabelCreate(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/labels", owner, repo)
}

// LabelGet / LabelDelete accept a RAW label name; URL-escaping happens here
// so callers don't have to remember to escape.
func LabelGet(owner, repo string, name string) string {
	return fmt.Sprintf("/repos/%s/%s/labels/%s", owner, repo, url.QueryEscape(name))
}

func LabelDelete(owner, repo string, name string) string {
	return fmt.Sprintf("/repos/%s/%s/labels/%s", owner, repo, url.QueryEscape(name))
}

// Release endpoints
func ReleasesList(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/releases", owner, repo)
}

// ReleasesGet / ReleaseEdit / ReleaseDelete accept a RAW tag name; URL-escaping
// happens here so callers don't have to remember to escape.
func ReleasesGet(owner, repo, tagName string) string {
	return fmt.Sprintf("/repos/%s/%s/releases/tags/%s", owner, repo, url.QueryEscape(tagName))
}

func ReleaseCreate(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/releases", owner, repo)
}

func ReleaseEdit(owner, repo, tagName string) string {
	return fmt.Sprintf("/repos/%s/%s/releases/tags/%s", owner, repo, url.QueryEscape(tagName))
}

func ReleaseDelete(owner, repo, tagName string) string {
	return fmt.Sprintf("/repos/%s/%s/releases/tags/%s", owner, repo, url.QueryEscape(tagName))
}

// Repo lifecycle endpoints (P0)
// Gitee v5 archives and renames a repo via PATCH on the repo resource.
// Sync fork is implemented as a special POST to refresh the fork from upstream.
func ReposArchive(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s", owner, repo)
}

func ReposRename(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s", owner, repo)
}

func ReposSync(owner, repo string) string {
	// Gitee v5: POST /repos/{owner}/{repo}/pull  (refresh fork from upstream)
	return fmt.Sprintf("/repos/%s/%s/pull", owner, repo)
}

// SSH key endpoints (P1a)
// Gitee v5 does NOT expose a single-key GET; list + delete by id only
// (same as GitHub).
func SSHKeysList() string          { return "/user/keys" }
func SSHKeyCreate() string         { return "/user/keys" }
func SSHKeyDelete(id int64) string { return fmt.Sprintf("/user/keys/%d", id) }

// GPG key endpoints (P1a)
func GPGKeysList() string          { return "/user/gpg_keys" }
func GPGKeyCreate() string         { return "/user/gpg_keys" }
func GPGKeyDelete(id int64) string { return fmt.Sprintf("/user/gpg_keys/%d", id) }

// Webhook endpoints (P1b)
// Gitee v5 webhook paths under /repos/{owner}/{repo}/hooks. The id is an
// integer so no URL escaping is required. "Test" is Gitee's equivalent of
// GitHub's "ping" — POST with no body, returns 204 on success.
func WebhooksList(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/hooks", owner, repo)
}

func WebhookGet(owner, repo string, id int64) string {
	return fmt.Sprintf("/repos/%s/%s/hooks/%d", owner, repo, id)
}

func WebhookCreate(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/hooks", owner, repo)
}

func WebhookEdit(owner, repo string, id int64) string {
	return fmt.Sprintf("/repos/%s/%s/hooks/%d", owner, repo, id)
}

func WebhookDelete(owner, repo string, id int64) string {
	return fmt.Sprintf("/repos/%s/%s/hooks/%d", owner, repo, id)
}

func WebhookTest(owner, repo string, id int64) string {
	return fmt.Sprintf("/repos/%s/%s/hooks/%d/tests", owner, repo, id)
}
