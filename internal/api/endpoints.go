package api

import "fmt"

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

func IssuesGet(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)
}

func IssuesCreate(owner string) string {
	return fmt.Sprintf("/repos/%s/issues", owner)
}

func IssuesEdit(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)
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
	return fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, number)
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
