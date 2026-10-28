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

// Tag endpoints (P0-2)
//
// Gitee v5 exposes the same path for list (GET) and create (POST):
// /repos/{owner}/{repo}/tags. Unlike labels / releases we DO NOT need
// URL-escaping for tag names because Gitee tag names follow the same
// restrictions as Git ref names (no characters requiring escaping).
// No DELETE endpoint exists in Gitee v5 — tag removal is delegated to
// `git push origin :refs/tags/<name>` (documented in the tag --agent-help).
func ReposTagsList(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/tags", owner, repo)
}

func ReposTagsCreate(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/tags", owner, repo)
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

// IssuesEdit returns the path for updating an existing issue on Gitee v5.
//
// IMPORTANT: Gitee v5's issue update endpoint is shaped DIFFERENTLY from the
// read endpoint. The READ path is /repos/{owner}/{repo}/issues/{number}
// (matches GitHub), but the WRITE path drops {repo} from the URL and
// instead accepts `repo` as a formData parameter alongside the fields
// being changed. All other update fields (title, body, state, assignee,
// collaborators, milestone, labels, ...) are also formData, NOT JSON.
//
// Verified against https://gitee.com/api/v5/swagger_doc.json (path
// /v5/repos/{owner}/issues/{number} with PATCH) and against a live
// gitee.com call: PATCH {write-path} with `repo=`+`state=` formData
// returns 200, while the read-shape path with JSON body returns 404
// "project or enterprise" / 400.
//
// Callers MUST send the request as application/x-www-form-urlencoded
// (use Client.PatchForm) and MUST include `repo` in the formData. The
// `repo` argument is kept in the signature for symmetry with the other
// getters/builders; callers should pass it through to PatchForm.
func IssuesEdit(owner, repo, number string) string {
	_ = repo // intentionally unused in URL path; supplied via formData
	return fmt.Sprintf("/repos/%s/issues/%s", owner, number)
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

// Gist comment endpoints (P2-11)
//
// Gitee v5 gist comment paths under /gists/{gist_id}/comments:
//
//	GET    — list comments on a gist
//	POST   — create a new comment on the gist (body: {body})
//	DELETE — remove a comment by id
//
// The same path is used for list (GET) and create (POST); the numeric
// comment id is appended as a path segment for DELETE. Gist ids are
// URL-safe alphanumeric/hyphen strings so no QueryEscape is needed —
// matches the existing GistGet / GistEdit builders above.
func GistCommentsList(gistID string) string {
	return fmt.Sprintf("/gists/%s/comments", gistID)
}

func GistCommentCreate(gistID string) string {
	return fmt.Sprintf("/gists/%s/comments", gistID)
}

func GistCommentDelete(gistID string, commentID int64) string {
	return fmt.Sprintf("/gists/%s/comments/%d", gistID, commentID)
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

// Issue comments list endpoint (P1c)
// GET /repos/{owner}/{repo}/issues/{number}/comments — distinct from the
// existing IssuesComment POST builder (which is the create endpoint).
func IssueCommentsList(owner, repo, number string) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%s/comments", owner, repo, number)
}

// PR edit endpoint (P1c)
// Gitee v5 PATCHes on the PR resource for edit, ready, draft toggle etc.
// — no separate /edit sub-path. Same path as PRsGet, different method.
func PRsEdit(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
}

// PR check-runs endpoint (P1c)
// Gitee v5 surfaces CI status via the GitHub-style "Checks" API at
// /repos/{owner}/{repo}/commits/{ref}/check-runs. To show the latest
// build status of a PR, fetch the PR (for head ref), then fetch check-runs
// with ?pull_request_id={id}&filter=latest.
func PRsChecks(owner, repo, ref string) string {
	return fmt.Sprintf("/repos/%s/%s/commits/%s/check-runs", owner, repo, url.QueryEscape(ref))
}

// Milestone endpoints (P0-3)
//
// Gitee v5 milestones are scoped per-repo and addressed by `number`
// (not by internal id) on read/edit/delete. The id is reserved for
// read-only contexts (e.g. webhook payloads) and is not part of the URL
// for normal CRUD. `state` defaults to "open" on POST when omitted.
func MilestonesList(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/milestones", owner, repo)
}

func MilestoneGet(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/milestones/%d", owner, repo, number)
}

func MilestoneCreate(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/milestones", owner, repo)
}

func MilestoneUpdate(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/milestones/%d", owner, repo, number)
}

func MilestoneDelete(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/milestones/%d", owner, repo, number)
}

// Release asset endpoints (v0.10.0)
// Gitee v5 asset operations use the numeric release ID (not tag name) and the
// `attach_files` path (not GitHub's `/assets`). The caller should first call
// ReleasesGet to resolve tag → ID, then use these builders with the numeric ID.
func ReleaseAssetUpload(owner, repo string, releaseID int64) string {
	return fmt.Sprintf("/repos/%s/%s/releases/%d/attach_files", owner, repo, releaseID)
}

// ReleaseAssetsList lists the attachments of a release. Unlike the `assets`
// array embedded in the release object (which only carries name +
// browser_download_url), this endpoint returns the numeric asset IDs that the
// delete endpoint needs.
func ReleaseAssetsList(owner, repo string, releaseID int64) string {
	return fmt.Sprintf("/repos/%s/%s/releases/%d/attach_files", owner, repo, releaseID)
}

func ReleaseDeleteAsset(owner, repo string, releaseID int64, assetID int64) string {
	return fmt.Sprintf("/repos/%s/%s/releases/%d/attach_files/%d", owner, repo, releaseID, assetID)
}

// Notification endpoints (P0-4)
//
// Gitee v5 exposes a GitHub-compatible notification center:
//
//	GET    /notifications/count          — unread count for the authenticated user
//	GET    /notifications/threads        — list threads (optionally ?unread=true)
//	PUT    /notifications/threads/{id}   — mark a single thread as read
//	PUT    /notifications/threads        — mark ALL threads as read (body controls scope)
//
// These are user-scoped (no {owner}/{repo} in the path), so the endpoints
// are constants rather than builders. The id in the mark-read-single path
// is a string (Gitee uses alphanumeric thread ids) so it is URL-escaped.
func NotificationsCount() string {
	return "/notifications/count"
}

func NotificationsList() string {
	return "/notifications/threads"
}

func NotificationMarkRead(threadID string) string {
	return fmt.Sprintf("/notifications/threads/%s", url.QueryEscape(threadID))
}

// NotificationsMarkAllRead is the same path as NotificationsList; the
// distinction is purely the HTTP verb (PUT vs GET). Kept as a separate
// builder for symmetry and to make call-sites self-documenting.
func NotificationsMarkAllRead() string {
	return "/notifications/threads"
}

// Collaborator endpoints (P1-5)
//
// Gitee v5 collaborator paths under /repos/{owner}/{repo}/collaborators.
// The username in the add/remove path is URL-escaped because Gitee usernames
// may contain characters that require encoding (rare but possible).
// The list (GET) and add (PUT) share the same path; the add call additionally
// accepts a `permission` body field (pull | push | admin). Remove uses the
// username path-segment with DELETE.
func CollaboratorsList(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/collaborators", owner, repo)
}

func CollaboratorAdd(owner, repo, username string) string {
	return fmt.Sprintf("/repos/%s/%s/collaborators/%s", owner, repo, url.QueryEscape(username))
}

func CollaboratorRemove(owner, repo, username string) string {
	return fmt.Sprintf("/repos/%s/%s/collaborators/%s", owner, repo, url.QueryEscape(username))
}

// Star / unstar endpoints (P1-6)
//
// Gitee v5 stars and watches are user-scoped (no {owner}/{repo} in the
// body — the path itself carries it). The watch endpoint additionally
// accepts a `watch_type` body field with three values:
//
//	watching       — full activity (issues, PRs, releases, comments)
//	releases_only  — release-only notifications
//	ignoring       — unsubscribe entirely (the user can still star)
//
// Star and watch live at different paths (starred vs subscriptions) so
// they are independent: starring does not subscribe, and subscribing
// does not star. Both PUT/DELETE pairs return 204 No Content on
// success; the list endpoint echoes the user's starred repos as full
// Repository objects (compatible with /user/repos shape).
func RepoStar(owner, repo string) string {
	return fmt.Sprintf("/user/starred/%s/%s", owner, repo)
}

func RepoUnstar(owner, repo string) string {
	return fmt.Sprintf("/user/starred/%s/%s", owner, repo)
}

func RepoWatch(owner, repo string) string {
	return fmt.Sprintf("/user/subscriptions/%s/%s", owner, repo)
}

func RepoUnwatch(owner, repo string) string {
	return fmt.Sprintf("/user/subscriptions/%s/%s", owner, repo)
}

// UserStarredList returns the path to list repositories the authenticated
// user has starred. Optional query params (sort, direction, per_page) are
// appended at the call site; the builder is a constant because the path
// is user-scoped with no segments to fill.
func UserStarredList() string {
	return "/user/starred"
}

// File content endpoints (P1-7)
//
// Gitee v5 file/contents API lives at /repos/{owner}/{repo}/contents/{path}.
// The same path is used for:
//
//	GET    — read a file's metadata + base64 content, or list a directory
//	POST   — create a new file (requires base64 content + commit message)
//	PUT    — update an existing file (requires sha + base64 content + message)
//	DELETE — remove a file (requires sha + commit message)
//
// And /repos/{owner}/{repo}/raw/{path} serves the raw bytes for direct
// download (no JSON envelope).
//
// The {path} segment can contain slashes; we do not URL-escape it because
// Gitee's v5 endpoints expect literal slashes in the path. Callers that
// want to send a file at the repo root pass "." as the path.
func RepoContentsGet(owner, repo, path string) string {
	return fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
}

func RepoContentsCreate(owner, repo, path string) string {
	return fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
}

func RepoContentsUpdate(owner, repo, path string) string {
	return fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
}

func RepoContentsDelete(owner, repo, path string) string {
	return fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
}

func RepoRawContent(owner, repo, path string) string {
	return fmt.Sprintf("/repos/%s/%s/raw/%s", owner, repo, path)
}

// Commit / compare endpoints (P1-8)
//
// Gitee v5 exposes the commit history and diff comparison under:
//
//	GET  /repos/{owner}/{repo}/commits              — list commits
//	GET  /repos/{owner}/{repo}/commits/{sha}        — single commit detail
//	GET  /repos/{owner}/{repo}/compare/{base}...{head}
//	                                               — diff two refs
//
// The {sha} in the single-commit path is URL-escaped because Gitee
// accepts both 40-char SHAs and short refs (branch/tag names) and the
// latter can contain slashes. The compare path also accepts short refs
// (branches, tags) so its segments are likewise escaped.
//
// List/detail/compare support optional query params (sha, path, author,
// since, until, per_page, page) that the cmd layer attaches via url.Values
// when it builds the final request URL — no parameter passing through the
// builder, matching how ReposTagsList / RepoContentsGet work elsewhere
// in this file.
func ReposCommitsList(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/commits", owner, repo)
}

func RepoCommitGet(owner, repo, sha string) string {
	return fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, url.QueryEscape(sha))
}

func ReposCompare(owner, repo, base, head string) string {
	return fmt.Sprintf("/repos/%s/%s/compare/%s...%s",
		owner, repo, url.QueryEscape(base), url.QueryEscape(head))
}

// User endpoints (P2-10)
//
// Gitee v5 exposes user accounts under /users/{username} (any user,
// public data) and the authenticated user under /user (no segments).
// Followers/following are per-user lists at /users/{username}/{followers,following}.
//
// The follow / unfollow verb is user-scoped, not per-pair: PUT (or DELETE)
// /user/following/{username} targets the AUTHENTICATED user's relationship
// with `username`. There is no body — PUT simply creates the follow edge
// and DELETE removes it; both return 204 No Content on success.
//
// `username` in the followers/following list paths is URL-escaped even
// though Gitee usernames are usually plain slugs — same defensive habit
// as the SSH key builders.
func UserGet(username string) string {
	return fmt.Sprintf("/users/%s", url.QueryEscape(username))
}

// UserCurrent returns the authenticated user's profile. Path is constant
// because it carries no segments; the token on the request identifies
// the user.
func UserCurrent() string {
	return "/user"
}

func UserFollowers(username string) string {
	return fmt.Sprintf("/users/%s/followers", url.QueryEscape(username))
}

func UserFollowing(username string) string {
	return fmt.Sprintf("/users/%s/following", url.QueryEscape(username))
}

func UserFollow(username string) string {
	return fmt.Sprintf("/user/following/%s", url.QueryEscape(username))
}

func UserUnfollow(username string) string {
	return fmt.Sprintf("/user/following/%s", url.QueryEscape(username))
}

// Branch protection endpoints (P2-12)
//
// Gitee v5 branch protection paths under /repos/{owner}/{repo}/branches:
//
//	PUT    /repos/{owner}/{repo}/branches/{branch}/protection — enable protection
//	DELETE /repos/{owner}/{repo}/branches/{branch}/protection — remove protection
//	GET    /repos/{owner}/{repo}/branches/{branch}            — read `protected` field
//
// The {branch} segment is URL-escaped because branch names may contain
// slashes (e.g. `feature/foo`) or other ref-name characters that require
// encoding. Both protect and unprotect share the same path; the verb
// (PUT vs DELETE) is what differentiates them, mirroring how
// RepoStar / RepoUnstar, RepoWatch / RepoUnwatch, and the
// collaborator add/remove endpoints are structured in this file.
//
// The PUT body is empty — protection in Gitee v5 is binary on/off with
// no per-branch options exposed by the API at this time (the
// `protected` field returned by the GET branch response is the only
// state we can toggle).
func RepoBranchProtection(owner, repo, branch string) string {
	return fmt.Sprintf("/repos/%s/%s/branches/%s/protection", owner, repo, url.QueryEscape(branch))
}

// RepoUnprotectBranch shares the path with RepoBranchProtection; the
// distinction is purely the HTTP verb (DELETE vs PUT). Kept as a
// separate builder for symmetry and to make call-sites self-documenting.
func RepoUnprotectBranch(owner, repo, branch string) string {
	return fmt.Sprintf("/repos/%s/%s/branches/%s/protection", owner, repo, url.QueryEscape(branch))
}
