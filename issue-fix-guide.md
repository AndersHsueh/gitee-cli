# gitee-cli 修复指南（给 MiniMax）

本文档列出 `gitee-cli` 仓库当前的所有已知 Bug 与死代码，按优先级排好顺序。**请严格按顺序执行每一节，不要跳步、不要合并、不要"顺手优化"未列出的地方。**

---

## 0. 开始之前的准备（必读）

### 0.1 工作目录
所有命令默认在 `/Users/xueyuheng/powerful-app/gitee-cli/` 下执行。

### 0.2 全程通用的"安全验证"命令
**每改完一个 fix，就必须依次跑以下三条命令，三条全过才能进入下一个 fix：**

```bash
gofmt -l cmd internal pkg          # 应该没有任何文件被列出（除非这就是 gofmt fix 步骤）
go build ./...                     # 编译必须通过
go vet ./...                       # vet 必须通过
go test ./... -count=1             # 测试必须通过
```

如果上面任何一条失败，**立即停下、回滚刚才的改动**，仔细对照本文档检查你是不是看错了行号或者改错了文件。**不要尝试"再多改一点点"来掩盖错误。**

### 0.3 提交策略
**每一个编号 fix 完成后，单独 commit 一次。** Commit message 使用：

```
fix(模块): 简短描述

修复 issue-fix-guide.md 第 N 节
```

模块名按文件路径取，例如 `cmd/issue/label.go` 的修改用 `fix(issue): ...`。

### 0.4 你不准做的事
- ❌ 不要修改本文档没列出来的任何文件。
- ❌ 不要为了"代码风格"重新排版没要求改的代码。
- ❌ 不要添加新的依赖到 `go.mod`。
- ❌ 不要修改 `go.mod` 的 `go` 版本。
- ❌ 不要写新文件，除非本文档明确要求（例如 Fix 14 删除整个包后不需要写新文件）。
- ❌ 不要添加新的命令、新的 flag、新的子模块。
- ❌ 不要"顺便"修中文注释里的错别字。
- ❌ 不要修改 `doc/`、`README*.md`、`documents/`、`test/*.sh` 这些文档相关目录，除非本文档明确要求。

---

## Fix 1 — 全仓库 gofmt（必须最先做）

### 问题
全仓库 45 个 Go 文件缩进错乱、不符合 gofmt。这些文件大多 tab 缩进掉到了第 0 列。

### 操作
```bash
gofmt -w cmd internal pkg
```

就这一条命令。**不要手动改任何空格、tab、换行。**

### 验证
```bash
gofmt -l cmd internal pkg        # 输出应为空
go build ./...                    # 应成功
go vet ./...                      # 应通过
go test ./... -count=1            # 应通过
```

### Commit
```
style: gofmt entire codebase

修复 issue-fix-guide.md 第 1 节
```

---

## Fix 2 — `gitee issue label` 全套命令的 owner 丢失（严重）

### 问题
`cmd/issue/label.go` 里所有命令调用 `resolveLabelRepo` 时把 owner 用 `_` 丢弃了，导致拼接出 `/repos//<repo>/labels` 这种空段 URL，全部 404。

### 文件
`cmd/issue/label.go`

### 修改点（共 6 处）
全部修改思路一致：把 `_, repoName, err := resolveLabelRepo(...)` 改成 `owner, repoName, err := resolveLabelRepo(...)`，然后把后面所有 `api.LabelXxx("", repoName, ...)` 的第一个参数从空字符串改成 `owner`。

#### 2.1 `runLabelList`（约 137 行附近）

**找到：**
```go
func runLabelList(c *cobra.Command, args []string) error {
	_, repoName, err := resolveLabelRepo(labelListFlags.repo)
	if err != nil {
		return err
	}
```
**改为：**
```go
func runLabelList(c *cobra.Command, args []string) error {
	owner, repoName, err := resolveLabelRepo(labelListFlags.repo)
	if err != nil {
		return err
	}
```

然后在同一个函数里继续找：
```go
	path := api.LabelList("", repoName)
```
**改为：**
```go
	path := api.LabelList(owner, repoName)
```

#### 2.2 `runLabelView`（约 188 行附近）

```go
	_, repoName, err := resolveLabelRepo(labelViewFlags.repo)
```
**改为：**
```go
	owner, repoName, err := resolveLabelRepo(labelViewFlags.repo)
```

然后：
```go
	path := api.LabelGet("", repoName, encodedName)
```
**改为：**
```go
	path := api.LabelGet(owner, repoName, encodedName)
```

#### 2.3 `runLabelCreate`（约 244 行附近）

```go
	_, repoName, err := resolveLabelRepo(labelCreateFlags.repo)
```
**改为：**
```go
	owner, repoName, err := resolveLabelRepo(labelCreateFlags.repo)
```

然后：
```go
	path := api.LabelCreate("", repoName)
```
**改为：**
```go
	path := api.LabelCreate(owner, repoName)
```

#### 2.4 `runLabelUpdate`（约 298 行附近）

```go
	_, repoName, err := resolveLabelRepo(labelUpdateFlags.repo)
```
**改为：**
```go
	owner, repoName, err := resolveLabelRepo(labelUpdateFlags.repo)
```

然后：
```go
	path := api.LabelGet("", repoName, encodedName)
```
**改为：**
```go
	path := api.LabelGet(owner, repoName, encodedName)
```

#### 2.5 `runLabelDelete`（约 363 行附近）

```go
	_, repoName, err := resolveLabelRepo(labelDeleteFlags.repo)
```
**改为：**
```go
	owner, repoName, err := resolveLabelRepo(labelDeleteFlags.repo)
```

然后**先**修改打印语句（这一处也用了空 owner）：
```go
	fmt.Printf("%s Delete label %q from %s/%s? [y/N] ", yellow("?"), labelDeleteFlags.name, "", repoName)
```
**改为：**
```go
	fmt.Printf("%s Delete label %q from %s/%s? [y/N] ", yellow("?"), labelDeleteFlags.name, owner, repoName)
```

然后再改：
```go
	path := api.LabelDelete("", repoName, encodedName)
```
**改为：**
```go
	path := api.LabelDelete(owner, repoName, encodedName)
```

### 验证
```bash
gofmt -l cmd internal pkg && go build ./... && go vet ./... && go test ./... -count=1
```

然后做一次**搜索式自检**（如果你正确地修完了，下面两条都应该输出空）：

```bash
grep -n '_, repoName, err := resolveLabelRepo' cmd/issue/label.go    # 应为空
grep -n 'api.Label.*"",' cmd/issue/label.go                          # 应为空
```

如果是 macOS bash 不识别 grep -n 也可以省略 -n。

### Commit
```
fix(issue): pass owner to label API calls

修复 issue-fix-guide.md 第 2 节
```

---

## Fix 3 — Issue 编号从 int 改为 string（严重）

### 问题
Gitee 的 issue 编号是字母数字字符串（例如 `I7T34V`），但代码里多处把它当 `int` 处理：
- `internal/api/endpoints.go` 里 `IssuesGet/IssuesEdit/IssuesComment` 的 `number` 参数类型是 `int`
- `cmd/issue/helpers.go` 里用 `fmt.Sscanf(args[0], "%d", &issueNum)` 解析，遇到 `I7T34V` 会沉默地解析成 0

结果：`gitee issue edit/close/reopen/comment` 在真实 Gitee issue 上全部失败。

`pkg/types/issue.go` 第 24 行已经把 `Issue.Number` 正确声明为 `string`，证明早期作者意识到了类型，只是 API endpoints 和 helpers 没跟上。

### 文件
- `internal/api/endpoints.go`
- `cmd/issue/helpers.go`

### 修改

#### 3.1 修改 `internal/api/endpoints.go`

找到这三个函数：

```go
func IssuesGet(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)
}
```
**改为：**
```go
func IssuesGet(owner, repo, number string) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number)
}
```

```go
func IssuesEdit(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)
}
```
**改为：**
```go
func IssuesEdit(owner, repo, number string) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%s", owner, repo, number)
}
```

```go
func IssuesComment(owner, repo string, number int) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)
}
```
**改为：**
```go
func IssuesComment(owner, repo, number string) string {
	return fmt.Sprintf("/repos/%s/%s/issues/%s/comments", owner, repo, number)
}
```

注意 3 个函数的签名：
- 第一个 `%d` → `%s`
- `number int` → `number string`
- **不要**改 owner / repo 这两个参数。

#### 3.2 修改 `cmd/issue/helpers.go`

这个文件里有 4 个 `runXxx` 函数（runEdit、runClose、runReopen、runComment），每个的修改模式相同：

**找到（每个函数都有）：**
```go
	var issueNum int
	fmt.Sscanf(args[0], "%d", &issueNum)
```
**改为：**
```go
	issueNum := args[0]
```

注意：
- 不需要 `strconv.Atoi`，Gitee 编号是字符串
- 不需要验证格式，Gitee API 自己会拒绝无效编号

然后每个函数里还会有一行用 `issueNum` 的地方：

- `runEdit`：`path := api.IssuesEdit(owner, repoName, issueNum)` — 不用改，因为现在 `issueNum` 已经是 string，函数签名也改成 string 了，类型对得上。
- `runClose`：`path := api.IssuesEdit(owner, repoName, issueNum)` — 同样不用改。
- `runReopen`：`path := api.IssuesEdit(owner, repoName, issueNum)` — 同样不用改。
- `runComment`：`path := api.IssuesComment(owner, repoName, issueNum)` — 同样不用改。

最后，**这 4 个函数里的 `fmt.Printf("Issue #%d ...", issueNum)` 全部要改成 `%s`：**

`runEdit` 找：
```go
	fmt.Printf("Issue #%d updated\n", issueNum)
```
改为：
```go
	fmt.Printf("Issue #%s updated\n", issueNum)
```

`runClose` 找：
```go
	fmt.Printf("Issue #%d closed\n", issueNum)
```
改为：
```go
	fmt.Printf("Issue #%s closed\n", issueNum)
```

`runReopen` 找：
```go
	fmt.Printf("Issue #%d reopened\n", issueNum)
```
改为：
```go
	fmt.Printf("Issue #%s reopened\n", issueNum)
```

`runComment` 找：
```go
	fmt.Printf("Comment added to issue #%d\n", issueNum)
```
改为：
```go
	fmt.Printf("Comment added to issue #%s\n", issueNum)
```

#### 3.3 验证完整性

修完后跑：
```bash
grep -n 'fmt.Sscanf.*issueNum' cmd/issue/helpers.go     # 应为空
grep -n 'Issue #%d' cmd/issue/helpers.go                # 应为空
grep -n 'var issueNum int' cmd/issue/helpers.go         # 应为空
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
fix(issue): treat issue number as string

Gitee issue numbers are alphanumeric (e.g. I7T34V), not integers.

修复 issue-fix-guide.md 第 3 节
```

---

## Fix 4 — `gitee pr diff` 不返回 diff（严重）

### 问题
`cmd/pr/diff.go` 调的是 `api.PRsGet`（返回 PR 的 JSON 详情），却把响应当 diff 输出。

### 文件
`cmd/pr/diff.go`

### 修改思路
PR 的 JSON 里有 `diff_url` 字段（在 `pkg/types/pr.go:11`）。正确的做法是：先 GET PR 详情，从里面取 `diff_url`，然后用一个**独立的 http.Client**（不要走 resty 那个带 BaseURL 的客户端，避免把 BaseURL 拼到完整 URL 上）去拉这个 URL。

### 完整替换 `cmd/pr/diff.go`

把整个文件替换为：

```go
package pr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
)

var diffFlags struct {
	owner string
	repo  string
}

var diffCmd = &cobra.Command{
	Use:     "diff <number>",
	Short:   "Show pull request diff",
	Example: `  gitee pr diff 42 --repo owner/repo`,
	RunE:    runDiff,
}

func init() {
	f := diffCmd.Flags()
	f.StringVar(&diffFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&diffFlags.repo, "repo", "R", "", "Repository name or owner/repo")
}

func runDiff(cobraCmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("PR number is required")
	}

	owner, repoName, err := resolvePRRepo(diffFlags.owner, diffFlags.repo)
	if err != nil {
		return err
	}

	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: must be an integer", args[0])
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var pr types.PullRequest
	if err := client.Get(ctx, api.PRsGet(owner, repoName, prNum), &pr); err != nil {
		return fmt.Errorf("getting PR: %w", err)
	}

	if pr.DiffURL == "" {
		return fmt.Errorf("PR #%d has no diff_url", prNum)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pr.DiffURL, nil)
	if err != nil {
		return fmt.Errorf("building diff request: %w", err)
	}

	httpClient := &http.Client{Timeout: time.Duration(cmd.GlobalTimeout) * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching diff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("diff request returned HTTP %d", resp.StatusCode)
	}

	_, err = io.Copy(cobraCmd.OutOrStdout(), resp.Body)
	return err
}
```

**注意**：上面的代码用到了 `strconv` 包，记得在 import 里加上 `"strconv"`。完整的 import 块应该长这样：

```go
import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
)
```

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
fix(pr): actually return a diff in pr diff command

Was previously fetching the PR JSON via PRsGet and printing it raw.

修复 issue-fix-guide.md 第 4 节
```

---

## Fix 5 — `git clone` 不能把 Token 写进 URL（安全）

### 问题
`cmd/repo/clone.go:52-61` 把 token 直接拼进 `https://<token>@gitee.com/...` 这种 URL，会被 git 持久化到 `.git/config`，并且暴露在 `ps`、shell history、错误日志里。

另外，主机名硬编码成 `gitee.com`，企业版用户不能用。

### 文件
`cmd/repo/clone.go`

### 修改思路
- 完全不在 URL 里塞 token，让 git 走自己的 credential helper / SSH。
- 主机名改用 `cmd.GlobalHostname`。

### 修改

**找到 `buildGitURL` 函数：**
```go
func buildGitURL(repo string) string {
	token := cmd.GlobalToken
	if token == "" {
		token = config.GetToken(cmd.GlobalHostname)
	}
	if token != "" {
		return fmt.Sprintf("https://%s@gitee.com/%s", token, repo)
	}
	return "https://gitee.com/" + repo
}
```

**整个函数替换为：**
```go
func buildGitURL(repo string) string {
	return fmt.Sprintf("https://%s/%s", cmd.GlobalHostname, repo)
}
```

然后把文件顶部的 import 里这一行删掉（不再用到 config 包）：
```go
	"gitee-cli/internal/config"
```

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
grep -n 'token' cmd/repo/clone.go          # 应为空
grep -n 'gitee.com' cmd/repo/clone.go      # 应为空
```

### Commit
```
fix(repo): stop embedding token in git clone URL

也修复了硬编码 gitee.com hostname 的问题。

修复 issue-fix-guide.md 第 5 节
```

---

## Fix 6 — `gitee repo browse` 主机名硬编码

### 问题
`cmd/repo/browse.go:34` 写死了 `https://gitee.com/...`，企业版用户没法用。

### 文件
`cmd/repo/browse.go`

### 修改
找到：
```go
	url := fmt.Sprintf("https://gitee.com/%s/%s", owner, repoName)
```
**改为：**
```go
	url := fmt.Sprintf("https://%s/%s/%s", cmd.GlobalHostname, owner, repoName)
```

然后确认 import 块里**已经有** `"gitee-cli/cmd"`（应该已经有了，因为文件其他地方在用）。如果没有就加上。

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
grep -n 'gitee.com' cmd/repo/browse.go     # 应为空
```

### Commit
```
fix(repo): honor --hostname in repo browse

修复 issue-fix-guide.md 第 6 节
```

---

## Fix 7 — `gitee repo list --owner` 错误信息误导

### 问题
`cmd/repo/list.go:63-69` 任何错误都被报告成 "user not found (HTTP 404)"，包括 401、500、网络错误。

### 文件
`cmd/repo/list.go`

### 修改
找到：
```go
	var repos []types.Repository
	if err := client.Get(ctx, path, &repos); err != nil {
		// Provide friendlier error message for --owner with non-existent user
		if listFlags.owner != "" {
			return fmt.Errorf("user %q not found or has no public repositories (HTTP 404). Check the username or ensure the user exists on Gitee", listFlags.owner)
		}
		return fmt.Errorf("listing repositories: %w", err)
	}
```

**改为：**
```go
	var repos []types.Repository
	if err := client.Get(ctx, path, &repos); err != nil {
		return fmt.Errorf("listing repositories: %w", err)
	}
```

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
fix(repo): stop misreporting all errors as 'user not found'

修复 issue-fix-guide.md 第 7 节
```

---

## Fix 8 — `repo edit --private` 单向不可逆

### 问题
`cmd/repo/edit.go` 的 `--private` 是普通 bool，永远没法把仓库改回 public，因为只有 `if editFlags.privateValue == true` 时才赋值。

### 文件
`cmd/repo/edit.go`

### 修改

Step 1：在 `init` 里把 `--private` 改成能跟踪"是否被设置"的形式。

**找到：**
```go
	f.BoolVar(&editFlags.privateValue, "private", false, "Make repository private")
```
**保持不变。** 不需要改 flag 注册。

Step 2：修改 `runEdit` 里使用 `editFlags.privateValue` 的逻辑。

**找到：**
```go
	if editFlags.privateValue {
		req.Private = &editFlags.privateValue
	}
```

**改为：**
```go
	if cobraCmd.Flags().Changed("private") {
		req.Private = &editFlags.privateValue
	}
```

解释：cobra 的 `Flags().Changed("name")` 在用户显式在命令行写了 `--private=true` 或 `--private=false` 时返回 true，没写时返回 false。这样：
- `gitee repo edit foo/bar --private` → req.Private = &true
- `gitee repo edit foo/bar --private=false` → req.Private = &false
- `gitee repo edit foo/bar`（没写 --private）→ req.Private 保持 nil，API 不会动这个字段

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
fix(repo): allow flipping repo back to public via --private=false

修复 issue-fix-guide.md 第 8 节
```

---

## Fix 9 — `gitee search` 默默吞错误

### 问题
`cmd/search/search.go:82-88` 失败时第二次重试的错误被 `_ =` 吞掉，用户以为搜出 0 条结果，实际是出错了。

### 文件
`cmd/search/search.go`

### 修改

**找到（在 `runSearchRepos` 函数内）：**
```go
	var result []interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		// Gitee may 400 on unsupported params; retry without filters
		if result == nil {
			path2 := fmt.Sprintf("/search/repositories?q=%s&per_page=%d", q, reposFlags.limit)
			_ = client.Get(ctx, path2, &result)
		}
	}

	return printSearchResults(result, "repos")
```

**改为：**
```go
	var result []interface{}
	if err := client.Get(ctx, path, &result); err != nil {
		return fmt.Errorf("search repos: %w", err)
	}

	return printSearchResults(result, "repos")
```

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
fix(search): propagate errors instead of silently returning empty results

修复 issue-fix-guide.md 第 9 节
```

---

## Fix 10 — 查询参数 URL 编码

### 问题
多处把用户输入直接拼接到 URL 的 query string 里，遇到空格、`&`、`+`、`#`、中文就会出错。

### 文件
- `cmd/search/search.go`
- `cmd/issue/list.go`
- `cmd/pr/list.go`
- `cmd/org/issue.go`

### 修改思路
统一改用 `net/url` 包的 `url.Values` 来构造 query。

#### 10.1 `cmd/search/search.go`

**找到 `runSearchRepos`：**
```go
	// Build path
	path := fmt.Sprintf("/search/repositories?q=%s&per_page=%d", q, reposFlags.limit)
	if reposFlags.sort != "" {
		path += "&sort=" + reposFlags.sort
	}
	if reposFlags.order != "" {
		path += "&order=" + reposFlags.order
	}
	if reposFlags.language != "" {
		path += "&language=" + reposFlags.language
	}
```

**改为：**
```go
	qv := url.Values{}
	qv.Set("q", q)
	qv.Set("per_page", fmt.Sprintf("%d", reposFlags.limit))
	if reposFlags.sort != "" {
		qv.Set("sort", reposFlags.sort)
	}
	if reposFlags.order != "" {
		qv.Set("order", reposFlags.order)
	}
	if reposFlags.language != "" {
		qv.Set("language", reposFlags.language)
	}
	path := "/search/repositories?" + qv.Encode()
```

**找到 `runSearchIssues`：**
```go
	path := fmt.Sprintf("/search/issues?q=%s&per_page=%d", q, issuesFlags.limit)
	if issuesFlags.state != "" && issuesFlags.state != "all" {
		path += "&state=" + issuesFlags.state
	}
	if issuesFlags.labels != "" {
		path += "&labels=" + issuesFlags.labels
	}
```

**改为：**
```go
	qv := url.Values{}
	qv.Set("q", q)
	qv.Set("per_page", fmt.Sprintf("%d", issuesFlags.limit))
	if issuesFlags.state != "" && issuesFlags.state != "all" {
		qv.Set("state", issuesFlags.state)
	}
	if issuesFlags.labels != "" {
		qv.Set("labels", issuesFlags.labels)
	}
	path := "/search/issues?" + qv.Encode()
```

**找到 `runSearchUsers`：**
```go
	path := fmt.Sprintf("/search/users?q=%s&per_page=%d", q, usersFlags.limit)
```

**改为：**
```go
	qv := url.Values{}
	qv.Set("q", q)
	qv.Set("per_page", fmt.Sprintf("%d", usersFlags.limit))
	path := "/search/users?" + qv.Encode()
```

最后，**确保文件顶部 import 里有 `"net/url"`。** 如果没有，加上。

#### 10.2 `cmd/issue/list.go`

**找到：**
```go
path := fmt.Sprintf("/repos/%s/%s/issues?state=%s&per_page=%d",
owner, repoName, listFlags.state, listFlags.limit)
if listFlags.labels != "" {
path += "&labels=" + listFlags.labels
}
if listFlags.assignee != "" {
path += "&assignee=" + listFlags.assignee
}
if listFlags.milestone != "" {
path += "&milestone=" + listFlags.milestone
}
```

**改为：**
```go
qv := url.Values{}
qv.Set("state", listFlags.state)
qv.Set("per_page", fmt.Sprintf("%d", listFlags.limit))
if listFlags.labels != "" {
	qv.Set("labels", listFlags.labels)
}
if listFlags.assignee != "" {
	qv.Set("assignee", listFlags.assignee)
}
if listFlags.milestone != "" {
	qv.Set("milestone", listFlags.milestone)
}
path := fmt.Sprintf("/repos/%s/%s/issues?%s", owner, repoName, qv.Encode())
```

**记得在 import 里加 `"net/url"`。**

#### 10.3 `cmd/pr/list.go`

**找到：**
```go
	path := fmt.Sprintf("%s?state=%s&per_page=%d", api.PRsList(owner, repoName), listFlags.state, listFlags.limit)
	if listFlags.head != "" {
		path += "&head=" + listFlags.head
	}
	if listFlags.base != "" {
		path += "&base=" + listFlags.base
	}
```

**改为：**
```go
	qv := url.Values{}
	qv.Set("state", listFlags.state)
	qv.Set("per_page", fmt.Sprintf("%d", listFlags.limit))
	if listFlags.head != "" {
		qv.Set("head", listFlags.head)
	}
	if listFlags.base != "" {
		qv.Set("base", listFlags.base)
	}
	path := api.PRsList(owner, repoName) + "?" + qv.Encode()
```

**记得在 import 里加 `"net/url"`。**

#### 10.4 `cmd/org/issue.go`

**找到：**
```go
	path := fmt.Sprintf("%s?state=%s&per_page=%d", api.OrgIssues(org), orgIssueFlags.state, orgIssueFlags.limit)
	if orgIssueFlags.labels != "" {
		path += "&labels=" + orgIssueFlags.labels
	}
```

**改为：**
```go
	qv := url.Values{}
	qv.Set("state", orgIssueFlags.state)
	qv.Set("per_page", fmt.Sprintf("%d", orgIssueFlags.limit))
	if orgIssueFlags.labels != "" {
		qv.Set("labels", orgIssueFlags.labels)
	}
	path := api.OrgIssues(org) + "?" + qv.Encode()
```

**记得在 import 里加 `"net/url"`。**

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
# 下面三条应该都是空（除非别的地方还有未处理的）
grep -rn 'path += "&' cmd/                       # 应为空
grep -rn 'fmt.Sprintf.*"q=%s' cmd/               # 应为空
```

### Commit
```
fix: URL-encode query parameters

防止 search 词、label、branch 名字等包含特殊字符时构造出错误的 URL。

修复 issue-fix-guide.md 第 10 节
```

---

## Fix 11 — PR 命令的 `fmt.Sscanf` 错误检查

### 问题
`cmd/pr/{merge,close,reopen,review,diff,view,comments}.go` 共 7 处使用 `fmt.Sscanf(args[0], "%d", &prNum)` 但忽略返回值。如果用户传非数字，`prNum` 静默为 0 然后请求 `/pulls/0`。

**注意：** `cmd/pr/diff.go` 已经在 Fix 4 改完了，这里不要再动它。

### 文件
- `cmd/pr/merge.go`
- `cmd/pr/close.go`
- `cmd/pr/reopen.go`
- `cmd/pr/review.go`
- `cmd/pr/view.go`
- `cmd/pr/comments.go`

### 修改
每个文件做相同的修改：

**找到（每个文件都有这一段）：**
```go
	var prNum int
	fmt.Sscanf(args[0], "%d", &prNum)
```

**改为：**
```go
	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR number %q: must be an integer", args[0])
	}
```

每个文件**还需要做两件事**：

1. 在 import 块里加上 `"strconv"`（如果还没有）。
2. 检查这段代码下面有没有 `client, err := cmd.NewAPIClient()` 这样的语句 — 因为我们已经声明了一个 `err`，下面再用 `:=` 重新声明 `err` 是允许的（左边有至少一个新变量 `client`），所以不会报错。**不需要把 `:=` 改成 `=`。**

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
grep -rn 'fmt.Sscanf.*prNum' cmd/pr/                    # 应为空
grep -rn 'var prNum int' cmd/pr/                        # 应为空
```

### Commit
```
fix(pr): validate PR number is an integer

修复 issue-fix-guide.md 第 11 节
```

---

## Fix 12 — `cmd/pr/list.go` 的 `resolvePRRepo` 忽略 owner 参数

### 问题
`cmd/pr/list.go:107` 的 `resolvePRRepo(owner, repo string)` 函数完全没用 `owner` 参数。所有 PR 命令传 `--owner` 都没效果。

### 文件
`cmd/pr/list.go`

### 修改

**找到整个 `resolvePRRepo` 函数：**
```go
func resolvePRRepo(owner, repo string) (string, string, error) {
	combined := repo
	if combined == "" {
		combined = cmd.GlobalRepo
	}
	if combined == "" {
		return "", "", fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}
	o, r := utils.ParseOwnerRepo(combined)
	if o == "" || r == "" {
		return "", "", fmt.Errorf("invalid --repo format %q, expected owner/repo", combined)
	}
	return o, r, nil
}
```

**整个函数替换为：**
```go
func resolvePRRepo(owner, repo string) (string, string, error) {
	combined := repo
	if combined == "" {
		combined = cmd.GlobalRepo
	}
	if combined == "" {
		return "", "", fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}
	o, r := utils.ParseOwnerRepo(combined)
	if o != "" && r != "" {
		return o, r, nil
	}
	// combined 不是 owner/repo 形式；如果显式给了 --owner，把 combined 当 repo 名
	if owner != "" && combined != "" && !strings.Contains(combined, "/") {
		return owner, combined, nil
	}
	return "", "", fmt.Errorf("invalid --repo format %q, expected owner/repo", combined)
}
```

**记得在 import 里加 `"strings"`（如果还没有）。**

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
fix(pr): honor --owner flag in resolvePRRepo

修复 issue-fix-guide.md 第 12 节
```

---

## Fix 13 — Token 短串切片可能 panic

### 问题
`cmd/auth/status.go:113`：`token[:4] + "..." + token[len(token)-4:]` 在 token 长度 < 4 时直接越界 panic。

### 文件
`cmd/auth/status.go`

### 修改

**找到：**
```go
	} else {
		// Redact token: show only last 4 chars
		masked := token[:4] + "..." + token[len(token)-4:]
		fmt.Printf("  Token:    %s\n", masked)
		fmt.Printf("  Hint:     Use --show-token to display full token\n")
	}
```

**改为：**
```go
	} else {
		fmt.Printf("  Token:    %s\n", maskToken(token))
		fmt.Printf("  Hint:     Use --show-token to display full token\n")
	}
```

然后在文件**底部**（最后一个函数后面）加一个新函数：

```go
// maskToken returns a redacted form of the token. Tokens shorter than 8
// characters are fully masked to avoid revealing meaningful prefixes.
func maskToken(token string) string {
	if len(token) < 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + "..." + token[len(token)-4:]
}
```

**记得在 import 里加 `"strings"`（如果还没有）。**

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
fix(auth): guard token masking against short tokens

修复 issue-fix-guide.md 第 13 节
```

---

## Fix 14 — 删除整个未使用的 `internal/repo` 包

### 问题
`internal/repo/repo.go` 整个包没有任何 import 和调用方，是死代码。

### 验证（开始前必做）
```bash
grep -rn '"gitee-cli/internal/repo"' --include="*.go" .       # 应为空
```
如果输出非空，**停止操作并告诉我**，意味着包并非死代码。

### 操作
```bash
rm -rf internal/repo
```

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
chore: remove unused internal/repo package

修复 issue-fix-guide.md 第 14 节
```

---

## Fix 15 — 删除整个未使用的 `internal/printer` 包

### 问题
`internal/printer/printer.go` 整个包没有任何 import 和调用方。

### 验证（开始前必做）
```bash
grep -rn '"gitee-cli/internal/printer"' --include="*.go" .    # 应为空
```
如果输出非空，**停止操作并告诉我**。

### 操作
```bash
rm -rf internal/printer
```

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
chore: remove unused internal/printer package

修复 issue-fix-guide.md 第 15 节
```

---

## Fix 16 — 删除 `internal/utils/utils.go` 中的死函数

### 问题
该文件有 14 个导出函数没人用。

### 文件
`internal/utils/utils.go`

### 修改
**保留**这三个（确实在用）：
- `Truncate`
- `TruncateDate`
- `ParseOwnerRepo`

**删除**以下函数及其注释（按出现顺序）：
- `HumanDuration`
- `FormatNumber`
- `Pluralize`
- `Contains`
- `Unique`
- `ParseBool`
- `CleanString`
- `ToTitleCase`
- `ParseTime`
- `DaysSince`
- `FormatStars`
- `NormalizeVisibility`
- `MaskString`
- `SplitLines`
- `JoinNonEmpty`

删完后，文件顶部的 import 块只保留还在用的包。最终 imports 应该是：

```go
import (
	"strings"
)
```

（因为只有 `ParseOwnerRepo` 用到 `strings`，其它都是被删函数用到的。`Truncate`、`TruncateDate` 都只用了基础语法不依赖任何包。）

**注意：** 如果 `gofmt` 之后发现 `strings` 也没用，那是因为 `ParseOwnerRepo` 的实现没用 `strings.Index`。检查代码后再决定是否保留 `strings`。如果 `go build` 报 "imported and not used: strings"，就把 strings 也删掉。

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
# 文件应该只剩下 3 个函数
grep -c '^func ' internal/utils/utils.go                # 应输出 3
```

### Commit
```
chore: remove unused functions from internal/utils

修复 issue-fix-guide.md 第 16 节
```

---

## Fix 17 — 删除 `cmd/auth` 里的死代码

### 问题
- `cmd/auth/status.go` 底部有两个导出但无人使用的函数：`GetUserInfo`、`FormatUserJSON`。
- `cmd/auth/token.go` 里 `logoutYes` 变量和 `--yes` flag 注册了但从未读取。

### 修改

#### 17.1 `cmd/auth/status.go`

**找到并删除这整段：**
```go
// GetUserInfo fetches detailed user information.
func GetUserInfo() (*userResponse, error) {
	apiClient, err := cmd.NewAPIClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var user userResponse
	if err := apiClient.Get(ctx, "/user", &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// FormatUserJSON formats user info as JSON.
func FormatUserJSON(user *userResponse) string {
	b, _ := json.MarshalIndent(user, "", "  ")
	return string(b)
}
```

如果删完后 `encoding/json` import 用不到了，把它也从 import 块里删掉。

#### 17.2 `cmd/auth/token.go`

**找到：**
```go
var logoutYes bool
```
**删掉这一行。**

**找到：**
```go
func init() {
	logoutCmd.Flags().BoolVar(&logoutYes, "yes", false, "Skip confirmation prompt")
}
```
**删掉整个 `func init()`。**

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
grep -n 'GetUserInfo\|FormatUserJSON\|logoutYes' cmd/auth/   # 应为空
```

### Commit
```
chore(auth): remove dead helpers and unused --yes flag

修复 issue-fix-guide.md 第 17 节
```

---

## Fix 18 — 删除 `cmd/gist/edit.go` 中未使用的扩展类型

### 问题
`cmd/gist/edit.go:116-129` 定义了 `GistEditRequestExtended` 类型和它的 `MarshalJSON`，但全仓库没人用。

### 文件
`cmd/gist/edit.go`

### 修改
**删除文件末尾的这一整段：**
```go
// GistEditRequestExtended extends the types.GistEditRequest to handle multi-file edits properly
type GistEditRequestExtended struct {
	Description string `json:"description,omitempty"`
	Files       map[string]struct {
		Content  string `json:"content,omitempty"`
		Filename string `json:"filename,omitempty"`
	} `json:"files,omitempty"`
}

// MarshalJSON implements json.Marshaler for proper multi-file edit format
func (r GistEditRequestExtended) MarshalJSON() ([]byte, error) {
	type alias GistEditRequestExtended
	return json.Marshal((*alias)(&r))
}
```

**注意：** `encoding/json` import 可能还被文件其它地方用着，**不要随便删 import**。先跑 `go build`，如果报 "imported and not used: encoding/json" 再删。

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
grep -n 'GistEditRequestExtended' cmd/                  # 应为空
```

### Commit
```
chore(gist): remove unused GistEditRequestExtended type

修复 issue-fix-guide.md 第 18 节
```

---

## Fix 19 — 删除其他零碎死代码

### 问题汇总
- `cmd/issue/view.go` 有个空的 `func init() {}`。
- `cmd/pr/merge.go` 注册了 `--delete-branch` flag 但代码里不读。
- `cmd/api/raw.go` 注册了 `--header` flag 但代码里不读。
- `cmd/config/config.go` 有一行 `_ = os.Stderr // suppress unused import`，应改成删 import。
- `cmd/issue/list.go` 的 `listFlags` 结构里有个 `repo string` 字段从来没绑定 flag。

### 修改

#### 19.1 `cmd/issue/view.go`
**删除：**
```go
func init() {
}
```

#### 19.2 `cmd/pr/merge.go`

**找到 `var mergeFlags struct {`，里面这一行：**
```go
	deleteBranch bool
```
**删掉。**

**找到 `init()` 里：**
```go
	f.BoolVar(&mergeFlags.deleteBranch, "delete-branch", false, "Delete source branch after merge")
```
**删掉。**

#### 19.3 `cmd/api/raw.go`

**找到 `var rawFlags struct {`，里面这一行：**
```go
	header  []string
```
**删掉。**

**找到 `init()` 里：**
```go
	rawCmd.Flags().StringArrayVar(&rawFlags.header, "header", []string{}, "Custom header (key=value)")
```
**删掉。**

#### 19.4 `cmd/config/config.go`

**找到 `func init()`：**
```go
func init() {
	ConfigCmd.AddCommand(listCmd, getCmd, setCmd, unsetCmd)
	_ = os.Stderr // suppress unused import
}
```

**改为：**
```go
func init() {
	ConfigCmd.AddCommand(listCmd, getCmd, setCmd, unsetCmd)
}
```

然后从 import 块里删掉：
```go
	"os"
```

**先用 `go build` 确认是否真的可以删 `"os"`** — 如果其它地方用到 `os` 包就不能删。

#### 19.5 `cmd/issue/list.go`

**找到 `var listFlags struct {`，里面这一行：**
```go
repo      string
```
**删掉。**

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
```

### Commit
```
chore: remove leftover dead code and unused flags

修复 issue-fix-guide.md 第 19 节
```

---

## Fix 20 — 合并 `isNotFoundError` 重复实现

### 问题
`cmd/pr/list.go:123` 和 `cmd/gist/list.go:85` 各自定义了完全相同的 `isNotFoundError` 函数。

### 修改思路
把它移到 `internal/api/errors.go`，让两个地方都引用。

### 修改

#### 20.1 修改 `internal/api/errors.go`

在文件末尾追加：
```go

// IsNotFound reports whether err indicates an HTTP 404 response.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "404") && strings.Contains(msg, "not found")
}
```

然后在文件顶部 import 块里加：
```go
	"strings"
```

#### 20.2 修改 `cmd/pr/list.go`

**删除文件末尾的：**
```go
// isNotFoundError checks if the error is an HTTP 404 not found error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "404") && strings.Contains(errStr, "not found")
}
```

**找到调用方：**
```go
		if isNotFoundError(err) {
```
**改为：**
```go
		if api.IsNotFound(err) {
```

如果文件不再用 `strings` 包，把 `"strings"` 从 import 里删掉。**用 `go build` 确认。**

#### 20.3 修改 `cmd/gist/list.go`

**删除文件末尾的：**
```go
// isNotFoundError checks if the error is an HTTP 404 not found error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "404") && strings.Contains(errStr, "not found")
}
```

**找到调用方：**
```go
		if isNotFoundError(err) {
```
**改为：**
```go
		if api.IsNotFound(err) {
```

同样地，如果文件不再用 `strings` 包就删掉 import。

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
grep -rn 'func isNotFoundError' cmd/                    # 应为空
```

### Commit
```
refactor(api): consolidate isNotFoundError into api.IsNotFound

修复 issue-fix-guide.md 第 20 节
```

---

## Fix 21 — `--no-color` 应该尽早生效

### 问题
`cmd/root.go:80-82` 把 `color.NoColor = true` 放在 `NewAPIClient` 里，不调 API 的命令（version、config get、auth login 等）不会理会 `--no-color`。

### 文件
`cmd/root.go`

### 修改

Step 1：删掉 `NewAPIClient` 里的颜色处理。

**找到：**
```go
	if NoColor {
		color.NoColor = true
	}

	return api.NewClient(token, baseURL, GlobalVerbose, GlobalTimeout), nil
```

**改为：**
```go
	return api.NewClient(token, baseURL, GlobalVerbose, GlobalTimeout), nil
```

Step 2：在 `rootCmd` 的定义里加 `PersistentPreRunE`。

**找到：**
```go
var rootCmd = &cobra.Command{
	Use:   "gitee",
	Short: "Gitee CLI — manage Gitee from your terminal",
	Long: `gitee-cli is a OpenSource command-line tool for Gitee, Author: AndersHsueh
Work with repositories, issues, pull requests, gists and more.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}
```

**改为：**
```go
var rootCmd = &cobra.Command{
	Use:   "gitee",
	Short: "Gitee CLI — manage Gitee from your terminal",
	Long: `gitee-cli is a OpenSource command-line tool for Gitee, Author: AndersHsueh
Work with repositories, issues, pull requests, gists and more.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if NoColor {
			color.NoColor = true
		}
		return nil
	},
}
```

### 验证
```bash
go build ./... && go vet ./... && go test ./... -count=1
./gitee version --no-color    # 输出不应有 ANSI 颜色码
```

### Commit
```
fix: apply --no-color before any command runs

修复 issue-fix-guide.md 第 21 节
```

---

## 最终全量验证

所有 fix 完成后，跑：

```bash
# 1. 全量 lint / build / test
gofmt -l cmd internal pkg
go build ./...
go vet ./...
go test ./... -count=1

# 2. 重新构建 binary 并跑帮助
go build -o gitee ./cmd/gitee
./gitee --help
./gitee issue label --help
./gitee pr diff --help
./gitee repo clone --help

# 3. 跑现有的 bash 集成测试（如果有 token）
# bash test/run.sh
```

要求：
- `gofmt -l` 输出为空
- `go build` 无错误
- `go vet` 无错误
- `go test` 全部通过
- `./gitee --help` 正常显示所有子命令
- 各 `--help` 不再显示 `--delete-branch`、`--header`、`--yes`（已删除的 flag）

---

## 出错时的回退手册

如果某一步搞砸了：

1. **不要慌、不要乱删文件、不要 `git reset --hard`**。
2. 先 `git status` 看现在的工作区差异。
3. `git diff` 查看自己改动了什么。
4. 用 `git restore <file>` 撤销单个文件的未提交修改。
5. 如果已经 commit 了但发现错了：**新建一个 commit 来修正**，不要 `git commit --amend`，也不要 `git reset`。
6. 如果完全搞不定，停下来，把当前 `git status` 和 `git diff` 输出贴出来求助。

---

## 完成后请确认

- [ ] 21 个 fix 全部完成
- [ ] 21 个独立 commit 全部存在（`git log --oneline | head -25` 看一下）
- [ ] `gofmt -l cmd internal pkg` 输出为空
- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] `go test ./... -count=1` 通过
- [ ] 没有修改本文档没列出的任何文件

完工。
