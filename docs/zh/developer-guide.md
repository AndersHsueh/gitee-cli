# gitee-cli 开发指导文档（Developer Guide）

> ⚠️ **状态提示（v0.13.2，2026-10 维护轮）**：本文档是早期设计稿，部分示例**已与当前代码不一致**
> （模块路径、`internal/printer`、Viper/oauth2、Makefile/goreleaser 等）。当前事实以代码、
> `gitee --help`、`gitee <cmd> --agent-help`、`bash scripts/ci.sh` 为准；差异最大的段落已修正。


版本：1.0.0（设计稿；现状修订见页首横幅）| 面向：贡献者 / 团队开发成员

---

## 1. 环境准备

### 1.1 必需工具

```bash
# Go 1.27+（必须；go.mod 为 `go 1.27.3`）
go version    # 需 >= 1.27

# golangci-lint（代码质量检查；CI 使用 v2.15.0，配置是 v2 schema）
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

# govulncheck（漏洞扫描；go.mod 以 tool 依赖固定版本）
go install golang.org/x/vuln/cmd/govulncheck@latest

# 发布与 CI 都是仓库内脚本，不需要 Makefile / goreleaser
bash scripts/ci.sh --fast   # 本地质量门禁
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
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go mod download
go install ./cmd/gitee      # 装到 $GOPATH/bin，日常开发用这个
gitee version
```

---

## 2. 仓库初始化（现状）

本仓库**已经建好**，不需要再跑 `go mod init` / `cobra-cli`。关键事实：

- 模块路径是 `gitee-cli`（不是 `gitee.com/<org>/gitee-cli`），
  所以内部 import 写作 `gitee-cli/internal/api`、`gitee-cli/cmd`。
- 依赖只有 Cobra、resty v2、fatih/color、testify（+ 间接依赖）；
  **没有 Viper、没有 golang.org/x/oauth2**。
- 配置是手写 JSON（`internal/config`），不是 Viper；认证是 PAT，不是 OAuth。

```bash
# 拉依赖 + 装工具链（照 go.mod）
go mod download
go build ./...              # 语法/编译自检
bash scripts/ci.sh --fast   # gofmt + vet + build + test（本地快速门禁）
```

---

## 3. 开发流程

### 3.1 新增子命令

以添加 `gitee issue list` 为例：

**Step 1：创建命令文件**

真实写法参考 `cmd/issue/list.go`（这里做了简化）：

```go
// cmd/issue/list.go
package issue

import (
    "net/url"

    "gitee-cli/cmd"
    "gitee-cli/internal/api"
    "gitee-cli/internal/cmdkit"
    "gitee-cli/internal/utils"
    "gitee-cli/pkg/types"

    "github.com/spf13/cobra"
)

var listFlags struct {
    state    string
    labels   string
    assignee string
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List issues in a repository",
    Example: `  gitee issue list --repo owner/repo
  gitee issue list --repo owner/repo --state closed --labels bug`,
    PreRunE: func(c *cobra.Command, args []string) error {
        if cmd.RuntimeFromCmd(c).AgentHelp {
            cmd.PrintAgentHelp(issueListAgentHelp)   // 可选：本命令的 agent-help 文案
        }
        return nil
    },
    RunE: runList,
}

func init() {
    f := listCmd.Flags()
    f.StringVar(&listFlags.state, "state", "open", "Filter by state: open|closed|all")
    f.StringVar(&listFlags.labels, "labels", "", "Filter by labels (comma-separated)")
    f.StringVar(&listFlags.assignee, "assignee", "", "Filter by assignee username")
    // --limit / --per-page / --paginate 是 root 的全局 flag，不要在这里重复注册。
    IssueCmd.AddCommand(listCmd)
}

func runList(cobraCmd *cobra.Command, args []string) error {
    rt := cmd.RuntimeFromCmd(cobraCmd)
    client, err := cmd.NewAPIClientFrom(rt)
    if err != nil {
        return err
    }

    // list 命令统一走 cmdkit.RunList 骨架：分页、表格、JSON、空结果提示一次搞定。
    opts := cmd.NewCmdkitOptionsFrom(rt, client)
    return cmdkit.RunList[types.Issue](opts, cmdkit.ListSpec[types.Issue]{
        ResolveOwnerRepo: func() (string, string, error) {
            return utils.ResolveRepo(rt.Repo, "", nil)
        },
        Path: func(owner, repoName string) string {
            qv := url.Values{}
            qv.Set("state", listFlags.state)
            if listFlags.labels != "" {
                qv.Set("labels", listFlags.labels)
            }
            if listFlags.assignee != "" {
                qv.Set("assignee", listFlags.assignee)
            }
            return api.IssuesList(owner, repoName) + "?" + qv.Encode()
        },
        Columns: []string{"#", "TITLE", "STATE", "LABELS", "UPDATED"},
        Row: func(i types.Issue) []string {
            return []string{i.Number, utils.Truncate(i.Title, 60), i.State, "-", i.UpdatedAt.Format("2006-01-02")}
        },
        EmptyText: func(o, r string) string { return "No issues found in " + o + "/" + r },
        ErrCtx:    "listing issues",
    })
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

端点只用 `internal/api/endpoints.go` 里的 builder 函数拼接，**禁止在 cmd 层 `fmt.Sprintf` 拼 URL**：

```go
// internal/api/endpoints.go
func IssuesList(owner, repo string) string {
    return "/repos/" + owner + "/" + repo + "/issues"
}

func IssueGet(owner, repo string, number int) string {
    return IssuesList(owner, repo) + "/" + strconv.Itoa(number)
}
```

**Step 4：添加/更新类型定义**

```go
// pkg/types/issue.go — 已在 SDD 中定义，按需扩展字段
```

**Step 5：编写测试（见第 4 节）**

---

### 3.2 新增 API 端点调用

优先复用已生成的 Client 方法（`Get/Post/Patch/PatchForm/Put/Delete/DeleteWithBody/UploadFile`），
不要再往 `internal/api` 里塞业务方法：

```go
// 读：直接传 result 指针
var issue types.Issue
err := client.Get(ctx, api.IssueGet(owner, repo, number), &issue)

// 写：JSON body
err := client.Post(ctx, api.IssuesCreate(owner, repo), req, &created)

// 写：Gitee v5 某些写路径只接受 formData（issue close/reopen/edit）
err := client.PatchForm(ctx, api.IssuesEdit(owner, number), url.Values{
    "repo":  {repo},
    "state": {"closed"},
})

// 列表：优先 FetchListOrAll / cmdkit.RunList，不要手写 paginate 分支
issues, err := api.FetchListOrAll[types.Issue](ctx, client, path, paginate, perPage, limit, page, "listing issues")
```

---

### 3.3 仓库上下文检测

仓库解析统一走 `internal/utils`，不要在命令里自己解析 git remote：

```go
// utils.ParseOwnerRepo("owner/repo") → owner, repo
// utils.ResolveRepo(owner, repo, args) → 无 flag 时按 --repo / 位置参数 / git remote 顺序解析
owner, repoName, err := utils.ResolveRepo(rt.Repo, "", args)
```

- `--repo` / `-R`（部分命令）显式指定时优先；
- 否则按位置参数（如 `gitee repo view owner/repo`）；
- 最后回退到当前目录的 git remote（`git remote get-url origin`）。

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

    client := api.NewClient("test-token", ts.URL, false, 30)  // token, baseURL, verbose, timeout

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

| 模块 | 建议覆盖率 |
|------|---------|
| `internal/api/` | 高（HTTP 客户端与错误映射） |
| `internal/config/` | 高（权限、原子写、优先级） |
| `internal/domain/` | 中（当前是薄弱区，新增写路径应带 httptest） |
| `pkg/types/` | 不强制 |
| `cmd/` | 中（`--agent-help` 契约测试已覆盖文案） |

> 现状：全仓库约 30%（历史上偏低），CI 目前不设覆盖率硬门禁；新增功能请自带 httptest。

---

## 5. 代码规范

### 5.1 Lint 配置（`.golangci.yml`）

仓库里的 `.golangci.yml` 是 **v2 schema**（`version: "2"`），使用默认 linter 集
（errcheck / govet / ineffassign / staticcheck / unused），并对测试文件的未检查返回值做了
定向豁免。gofmt 由 `scripts/ci.sh` 单独守住。

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

### 6.4 使用 Token 直接测试

```bash
# 推荐：环境变量（不落盘、不进 shell history）
export GITEE_TOKEN=your_access_token
gitee auth status

# 或指定 token 登录（会写入 ~/.gitee-cli/settings.json）
gitee auth login --token your_access_token
```

---

## 7. 构建与发布

### 7.1 本地构建

```bash
go install ./cmd/gitee    # 安装到 $GOPATH/bin（日常开发）
go build -o gitee ./cmd/gitee   # 一次性本地构建（不进 PATH）
```

### 7.2 交叉编译

```bash
# 手动交叉编译（发布脚本也这么干）
GOOS=linux   GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee
GOOS=linux   GOARCH=arm64 go build -o dist/gitee-linux-arm64 ./cmd/gitee
GOOS=darwin  GOARCH=amd64 go build -o dist/gitee-darwin-amd64 ./cmd/gitee
GOOS=darwin  GOARCH=arm64 go build -o dist/gitee-darwin-arm64 ./cmd/gitee
GOOS=windows GOARCH=amd64 go build -o dist/gitee-windows-amd64.exe ./cmd/gitee
```

### 7.3 发布 Release

```bash
# 先打标签并推送
git tag -a v0.13.2 -m v0.13.2 && git push origin v0.13.2

# 构建 5 平台产物 + SHA256SUMS + 创建 Gitee release + 上传资产（幂等）
GITEE_TOKEN=... bash scripts/release.sh --publish
```

> 没有 Makefile、没有 goreleaser。发布靠 `scripts/release.sh`（构建/打包/校验和）
> 与 `scripts/publish-release.sh`（创建 release + 上传资产，复用 `gitee release create/upload`）。

---

## 8. CI/CD 流程

CI 有**两个触发面、一份门禁定义**：

- `.github/workflows/ci.yml`（GitHub Actions 镜像）
- `.workflow/ci.yml`（Gitee Go）
- 两者都调用 `scripts/ci.sh`：gofmt → go vet → golangci-lint（可选）→ build →
  `go test -race -count=1` → 交叉编译 → govulncheck

本地复现 CI：

```bash
bash scripts/ci.sh          # 全量（含交叉编译 + govulncheck）
bash scripts/ci.sh --fast   # 跳过交叉编译 + govulncheck，本地快速回路
```

---

## 9. 项目里程碑

| 阶段 | 目标命令 | 说明 |
|------|---------|------|
| **M0** | `gitee auth` 全套 | PAT 认证（无 OAuth2 设备流），Token 存储，`auth status/logout/token` |
| **M1** | `gitee repo` 全套 + `gitee api` | 仓库 CRUD，原始 API 访问，配置模块 |
| **M2** | `gitee issue` 全套 | Issue CRUD，评论，标签管理 |
| **M3** | `gitee pr` 全套 | PR CRUD，Merge，Review，diff/checks |
| **M4** | `gitee gist` + `gitee search` | Gist 管理，三类搜索 |
| **M5** | `gitee org` + Shell 补全 | 组织管理，Bash/Zsh/Fish/PowerShell 补全（i18n 未实现） |
| **M6** | 企业部署支持 + 多账号 + 插件系统 | `--hostname` 扩展，credentials 多 host，外部命令发现 |

---

## 10. 参考资料

| 资源 | 链接 |
|------|------|
| Gitee API v5 Swagger | https://gitee.com/api/v5/swagger |
| Gitee 个人访问令牌 | https://gitee.com/profile/personal_access_tokens |
| Cobra 官方文档 | https://pkg.go.dev/github.com/spf13/cobra |
| Resty HTTP 客户端 | https://github.com/go-resty/resty |
| glab 项目（最佳参考实现） | https://github.com/gitlab-org/cli |
| gh CLI 项目 | https://github.com/cli/cli |
| Go 项目布局规范 | https://github.com/golang-standards/project-layout |
| Conventional Commits | https://www.conventionalcommits.org |
| golangci-lint 规则文档 | https://golangci-lint.run/usage/linters/ |
