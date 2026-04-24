# gitee-cli 开发指导文档（Developer Guide）

版本：1.0.0 | 面向：贡献者 / 团队开发成员

---

## 1. 环境准备

### 1.1 必需工具

```bash
# Go 1.23+（必须）
go version    # 需 >= 1.23

# Cobra CLI（命令模板生成器）
go install github.com/spf13/cobra-cli@latest

# golangci-lint（代码质量检查）
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin v1.59.0

# goreleaser（发版自动化）
go install github.com/goreleaser/goreleaser@latest

# govulncheck（漏洞扫描）
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### 1.2 推荐工具

```bash
# goimports（import 整理）
go install golang.org/x/tools/cmd/goimports@latest

# mockgen（生成 Mock）
go install go.uber.org/mock/mockgen@latest
```

### 1.3 克隆与初始化

```bash
git clone https://gitee.com/<your-user>/gitee-cli.git
cd gitee-cli
go mod download
go build -o bin/gitee ./cmd/gitee
./bin/gitee --version
```

---

## 2. 项目初始化（新建时）

```bash
# 初始化 Go 模块（模块路径遵循 gitee.com/<org>/gitee-cli）
go mod init gitee.com/gitee/gitee-cli

# 添加依赖
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/go-resty/resty/v2
go get golang.org/x/oauth2
go get github.com/fatih/color
go get github.com/stretchr/testify

# 使用 Cobra CLI 生成骨架
cobra-cli init --viper --author "Gitee Team" --license apache
cobra-cli add auth
cobra-cli add repo
cobra-cli add issue
cobra-cli add pr
cobra-cli add gist
cobra-cli add org
cobra-cli add search
cobra-cli add api
cobra-cli add config
cobra-cli add completion
```

---

## 3. 开发流程

### 3.1 新增子命令

以添加 `gitee issue list` 为例：

**Step 1：创建命令文件**

```go
// cmd/issue/list.go
package issue

import (
    "context"
    "fmt"

    "github.com/spf13/cobra"
    "gitee.com/gitee/gitee-cli/internal/api"
    "gitee.com/gitee/gitee-cli/internal/printer"
    "gitee.com/gitee/gitee-cli/pkg/types"
)

var listFlags struct {
    State    string
    Labels   string
    Assignee string
    Repo     string
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "列出 Issue",
    Long:  "列出当前仓库或指定仓库的 Issue，支持按状态、标签、指派人筛选。",
    RunE:  runList,
}

func init() {
    f := listCmd.Flags()
    f.StringVarP(&listFlags.State, "state", "s", "open", "Issue 状态: open/closed/all")
    f.StringVarP(&listFlags.Labels, "labels", "l", "", "标签筛选（逗号分隔）")
    f.StringVarP(&listFlags.Assignee, "assignee", "a", "", "指派人筛选")
    f.StringVarP(&listFlags.Repo, "repo", "R", "", "仓库（owner/name 格式）")

    IssueCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    client, err := api.NewClientFromConfig()
    if err != nil {
        return err
    }

    owner, repo, err := resolveRepo(listFlags.Repo)
    if err != nil {
        return err
    }

    var issues []types.Issue
    path := fmt.Sprintf("/repos/%s/%s/issues?state=%s", owner, repo, listFlags.State)
    if listFlags.Labels != "" {
        path += "&labels=" + listFlags.Labels
    }
    if err := client.List(ctx, path, nil, &issues); err != nil {
        return err
    }

    outputFmt, _ := cmd.Flags().GetString("output")
    p := printer.New(outputFmt, cmd.OutOrStdout())
    p.PrintHeader("#", "TITLE", "STATE", "LABELS", "ASSIGNEE", "UPDATED")
    for _, i := range issues {
        p.PrintRow(
            fmt.Sprintf("#%d", i.Number),
            truncate(i.Title, 60),
            i.State,
            joinLabels(i.Labels, 3),
            firstAssignee(i.Assignees),
            humanTime(i.UpdatedAt),
        )
    }
    return p.Flush()
}
```

**Step 2：在父命令注册**

```go
// cmd/issue/issue.go
var IssueCmd = &cobra.Command{
    Use:   "issue",
    Short: "管理 Issue",
}

// cmd/root.go
func init() {
    rootCmd.AddCommand(issue.IssueCmd)
}
```

**Step 3：添加 API 端点（如需新端点）**

```go
// internal/api/endpoints.go
const (
    EndpointIssues = "/repos/%s/%s/issues"
    EndpointIssue  = "/repos/%s/%s/issues/%d"
    EndpointIssueComments = "/repos/%s/%s/issues/%d/comments"
)
```

**Step 4：添加/更新类型定义**

```go
// pkg/types/issue.go — 已在 SDD 中定义，按需扩展字段
```

**Step 5：编写测试（见第 4 节）**

---

### 3.2 新增 API 端点调用

```go
// internal/api/client.go 中添加语义化方法
func (c *client) ListIssues(ctx context.Context, owner, repo, state string) ([]types.Issue, error) {
    var issues []types.Issue
    path := fmt.Sprintf("/repos/%s/%s/issues?state=%s", owner, repo, state)
    err := c.List(ctx, path, &PageOptions{PerPage: 30}, &issues)
    return issues, err
}

func (c *client) CreateIssue(ctx context.Context, owner, repo string, body types.IssueCreateRequest) (*types.Issue, error) {
    var issue types.Issue
    path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
    err := c.Post(ctx, path, body, &issue)
    return &issue, err
}
```

---

### 3.3 仓库上下文检测

很多命令需要自动检测当前仓库（类似 `gh`），从 git remote 解析：

```go
// internal/git/remote.go
func DetectRepo() (owner, repo string, err error) {
    // 1. 检查 --repo flag
    // 2. 解析 git remote origin URL
    //    - https://gitee.com/owner/repo.git
    //    - git@gitee.com:owner/repo.git
    // 3. 返回 owner, repo
    remoteURL, err := exec.Command("git", "remote", "get-url", "origin").Output()
    // ... 解析逻辑
}
```

---

## 4. 测试规范

### 4.1 单元测试

每个 `*.go` 文件对应 `*_test.go`，使用 `httptest` Mock HTTP：

```go
// internal/api/client_test.go
func TestClient_ListIssues(t *testing.T) {
    // 构造 Mock 服务器
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "GET", r.Method)
        assert.Contains(t, r.URL.Path, "/issues")
        assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("X-Total-Count", "2")
        json.NewEncoder(w).Encode([]types.Issue{
            {Number: 1, Title: "First Issue", State: "open"},
            {Number: 2, Title: "Second Issue", State: "closed"},
        })
    }))
    defer ts.Close()

    client := api.NewClient(
        api.WithBaseURL(ts.URL),
        api.WithToken("test-token"),
    )

    issues, err := client.ListIssues(context.Background(), "owner", "repo", "all")
    require.NoError(t, err)
    require.Len(t, issues, 2)
    assert.Equal(t, 1, issues[0].Number)
    assert.Equal(t, "First Issue", issues[0].Title)
}
```

### 4.2 命令层测试（输出验证）

```go
// cmd/issue/list_test.go
func TestIssueListCommand(t *testing.T) {
    // 构造 Mock API
    ts := setupMockServer(t, mockIssuesResponse)
    defer ts.Close()

    // 捕获 stdout
    buf := &bytes.Buffer{}
    cmd := newTestRootCmd(ts.URL)
    cmd.SetOut(buf)
    cmd.SetArgs([]string{"issue", "list", "--output", "json"})

    err := cmd.Execute()
    require.NoError(t, err)

    var result []map[string]interface{}
    require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
    assert.Len(t, result, 2)
}
```

### 4.3 运行测试

```bash
go test ./...                    # 运行所有测试
go test ./cmd/... -v             # 命令层详细输出
go test ./internal/api/... -v   # API 层详细输出
go test ./... -cover             # 覆盖率
go test ./... -race              # 竞态检测（CI 必须）
go test ./... -count=1           # 禁用缓存（CI 使用）
```

### 4.4 覆盖率要求

| 模块 | 最低覆盖率 |
|------|---------|
| `internal/api/` | 80% |
| `internal/auth/` | 70% |
| `internal/printer/` | 90% |
| `pkg/types/` | 不强制 |
| `cmd/` | 60% |

---

## 5. 代码规范

### 5.1 Lint 配置（`.golangci.yml`）

```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - gosec
    - revive
    - exhaustive

linters-settings:
  goimports:
    local-prefixes: gitee.com/gitee/gitee-cli
```

运行：

```bash
golangci-lint run ./...
golangci-lint run --fix ./...   # 自动修复
```

### 5.2 格式化

```bash
go fmt ./...        # 格式化代码
goimports -w .      # 整理 import（分组：stdlib / 外部包 / 内部包）
```

### 5.3 Commit 规范（Conventional Commits）

格式：`<type>(<scope>): <description>`

| 类型 | 适用场景 |
|------|---------|
| `feat` | 新增功能 |
| `fix` | Bug 修复 |
| `docs` | 文档变更 |
| `test` | 添加/修改测试 |
| `refactor` | 重构（非功能变更） |
| `chore` | 依赖更新、构建脚本等 |
| `perf` | 性能优化 |
| `ci` | CI/CD 配置变更 |

示例：

```
feat(repo): add repo fork command
fix(auth): handle expired token gracefully
docs(readme): add installation guide for macOS
test(api): add client pagination test coverage
refactor(printer): extract interface for output formats
chore(deps): bump resty to v2.12.0
```

### 5.4 代码审查自检清单

```
开发者提交前：
[ ] go fmt && goimports 已运行
[ ] golangci-lint 无警告
[ ] 所有 API 调用有错误处理（不能忽略 error）
[ ] 网络请求传入 context.Context
[ ] Token 不出现在日志、stdout、任何可见输出
[ ] 新功能有对应单元测试
[ ] 测试覆盖 happy path + error path
[ ] --help 输出描述准确
[ ] 新 flag 有默认值说明
```

---

## 6. 调试技巧

### 6.1 开启详细日志

```bash
gitee --verbose repo list
# 输出：
# → GET https://gitee.com/api/v5/user/repos?page=1&per_page=30
# ← HTTP 200 (245ms) [3 items]
```

### 6.2 使用 `gitee api` 命令直接测试端点

```bash
# 获取当前用户信息
gitee api GET /user

# 列出仓库（带分页）
gitee api GET /user/repos?type=owner&per_page=5

# 创建测试仓库
gitee api POST /user/repos --field name=test-cli-repo --field private=true --field auto_init=true

# 删除测试仓库
gitee api DELETE /repos/owner/test-cli-repo
```

### 6.3 覆盖 API 地址（测试私有部署）

```bash
gitee --api-url "https://your-enterprise.gitee.com/api/v5" repo list
# 或
GITEE_API_URL="https://your-enterprise.gitee.com/api/v5" gitee repo list
```

### 6.4 使用 Token 直接测试（跳过 OAuth）

```bash
gitee auth login --with-token
# 从 stdin 读取 token：
echo "your_access_token" | gitee auth login --with-token
```

---

## 7. 构建与发布

### 7.1 本地构建

```bash
make build          # 编译到 bin/gitee
make install        # 安装到 $GOPATH/bin
```

### 7.2 交叉编译

```bash
make cross-build
# 输出：
# dist/gitee-linux-amd64
# dist/gitee-linux-arm64
# dist/gitee-darwin-amd64
# dist/gitee-darwin-arm64
# dist/gitee-windows-amd64.exe
```

### 7.3 发布 Release

```bash
# 先打标签
git tag v0.1.0
git push origin v0.1.0

# goreleaser 自动构建 + 上传
make release
```

---

## 8. CI/CD 流程

### 8.1 GitHub Actions / Gitee Go CI 示例

```yaml
# .gitee/workflows/ci.yml
name: CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go mod download
      - run: go vet ./...
      - run: golangci-lint run ./...
      - run: go test ./... -v -race -count=1
      - run: govulncheck ./...

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: make cross-build
```

---

## 9. 项目里程碑

| 阶段 | 目标命令 | 说明 |
|------|---------|------|
| **M0** | `gitee auth` 全套 | 完成 OAuth2 Device Flow，Token 存储，`auth status/logout/token` |
| **M1** | `gitee repo` 全套 + `gitee api` | 仓库 CRUD，原始 API 访问，配置模块 |
| **M2** | `gitee issue` 全套 | Issue CRUD，评论，标签管理 |
| **M3** | `gitee pr` 全套 | PR CRUD，Merge，Review，diff/checks |
| **M4** | `gitee gist` + `gitee search` | Gist 管理，三类搜索 |
| **M5** | `gitee org` + Shell 补全 + i18n | 组织管理，Bash/Zsh/Fish 补全，中英文切换 |
| **M6** | 企业部署支持 + 多账号 + 插件系统 | `--hostname` 扩展，credentials 多 host，外部命令发现 |

---

## 10. 参考资料

| 资源 | 链接 |
|------|------|
| Gitee API v5 Swagger | https://gitee.com/api/v5/swagger |
| Gitee OAuth2 文档 | https://gitee.com/api/v5/oauth_doc |
| Cobra 官方文档 | https://pkg.go.dev/github.com/spf13/cobra |
| Viper 官方文档 | https://pkg.go.dev/github.com/spf13/viper |
| Resty HTTP 客户端 | https://github.com/go-resty/resty |
| glab 项目（最佳参考实现） | https://github.com/gitlab-org/cli |
| gh CLI 项目 | https://github.com/cli/cli |
| Go 项目布局规范 | https://github.com/golang-standards/project-layout |
| Conventional Commits | https://www.conventionalcommits.org |
| golangci-lint 规则文档 | https://golangci-lint.run/usage/linters/ |
