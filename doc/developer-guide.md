# gitee-cli 开发指导文档

## 1. 环境准备

### 1.1 必需工具

```bash
# Go 1.23+
go version  # 需 >= 1.23

# Cobra CLI（生成命令模板）
go install github.com/spf13/cobra-cli@latest

# golangci-lint（代码检查）
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# goreleaser（发版工具）
go install github.com/goreleaser/goreleaser@latest
```

### 1.2 克隆与依赖安装

```bash
git clone https://gitee.com/<your-user>/gitee-cli.git
cd gitee-cli
go mod download
```

---

## 2. 项目初始化

### 2.1 使用 Cobra 生成命令骨架

```bash
# 初始化 go.mod
go mod init gitee.com/<your-user>/gitee-cli

# 生成根命令
cobra-cli init --viper

# 生成子命令示例
cobra-cli add auth
cobra-cli add repo
cobra-cli add issue
```

---

## 3. 开发流程

### 3.1 新增一个子命令

以 `gitee issue list` 为例：

```bash
# 1. 创建命令文件
touch cmd/issue/list.go
```

```go
// cmd/issue/list.go
package issue

import (
    "github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "列出 Issue",
    RunE:  runList,
}

var listFlags = struct {
    State    string
    Labels   string
    Assignee string
}{}

func init() {
    listCmd.Flags().StringVarP(&listFlags.State, "state", "s", "open", "Issue 状态: open/closed/all")
    listCmd.Flags().StringVarP(&listFlags.Labels, "labels", "l", "", "标签筛选（逗号分隔）")
    listCmd.Flags().StringVarP(&listFlags.Assignee, "assignee", "a", "", "指派人筛选")

    issueCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
    // 获取 API Client
    client, err := api.NewClient()
    if err != nil {
        return err
    }

    // 调用 API
    issues, err := client.Issue.List(api.IssueListOptions{
        State:    listFlags.State,
        Labels:   listFlags.Labels,
        Assignee: listFlags.Assignee,
    })
    if err != nil {
        return err
    }

    // 输出
    printer := printer.New(outputFormat)
    printer.PrintHeader("编号", "标题", "状态", "指派")
    for _, i := range issues {
        printer.PrintRow(
            fmt.Sprintf("#%d", i.Number),
            i.Title,
            i.State,
            i.Assignee,
        )
    }
    printer.Flush()
    return nil
}
```

### 3.2 新增 API 端点

```go
// internal/api/issues.go
func (c *Client) ListIssues(owner, repo string, opts IssueListOptions) ([]Issue, error) {
    var issues []Issue
    path := fmt.Sprintf("/repos/%s/%s/issues?state=%s", owner, repo, opts.State)
    err := c.list(path, &issues)
    return issues, err
}
```

### 3.3 添加类型定义

```go
// pkg/types/issue.go
package types

type Issue struct {
    Number    int           `json:"number"`
    Title     string        `json:"title"`
    State     string        `json:"state"`
    Body      string        `json:"body"`
    Labels    []Label       `json:"labels"`
    Assignees []User        `json:"assignees"`
    Milestone *Milestone    `json:"milestone"`
    CreatedAt time.Time     `json:"created_at"`
    UpdatedAt time.Time     `json:"updated_at"`
}
```

---

## 4. 测试规范

### 4.1 单元测试

每个 `*.go` 文件对应一个 `*_test.go` 文件：

```go
// internal/api/client_test.go
func TestClient_ListIssues(t *testing.T) {
    // 使用 httptest 模拟 Gitee API
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode([]Issue{
            {Number: 1, Title: "Test Issue"},
        })
    }))
    defer ts.Close()

    client := NewClient(WithBaseURL(ts.URL))
    issues, err := client.ListIssues("owner", "repo", IssueListOptions{})
    require.NoError(t, err)
    require.Len(t, issues, 1)
}
```

### 4.2 运行测试

```bash
go test ./...                # 运行所有测试
go test ./cmd/... -v         # 查看详细输出
go test ./... -cover         # 覆盖率报告
go test ./... -race          # 竞态检测
```

---

## 5. 代码规范

### 5.1 Lint 检查

```bash
golangci-lint run ./...
```

### 5.2 格式化

```bash
go fmt ./...     # 格式化
goimports -w .   # 整理 import
```

### 5.3 Commit 规范

使用 [Conventional Commits](https://www.conventionalcommits.org/)：

```
feat(repo): add repo fork command
fix(auth): handle expired token gracefully
docs(readme): add installation guide
test(api): add client test coverage
refactor(printer): extract interface for output formats
```

### 5.4 代码审查自检清单

- [ ] 所有 API 调用有错误处理
- [ ] 所有用户输入有验证
- [ ] Token 不写入日志
- [ ] 网络请求有超时
- [ ] 测试覆盖新代码路径

---

## 6. 构建与发布

### 6.1 本地构建

```bash
go build -o gitee ./cmd/gitee
./gitee --version
```

### 6.2 交叉编译

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee

# Windows
GOOS=windows GOARCH=amd64 go build -o dist/gitee-windows-amd64.exe ./cmd/gitee
```

### 6.3 使用 Makefile

```bash
make build      # 本地构建
make test       # 运行测试
make lint       # Lint 检查
make install    # 安装到 $GOPATH/bin
make cross-build  # 交叉编译所有平台
```

---

## 7. 调试技巧

### 7.1 开启调试模式

```bash
gitee --verbose repo list
# 输出：HTTP Request/Response 详情
```

### 7.2 使用 `gitee api` 测试

```bash
gitee api GET /user
gitee api POST /user/repos --field name=test
```

### 7.3 测试 OAuth 流程

```bash
gitee auth login --scopes repo,user,gist
```

---

## 8. 依赖管理

### 8.1 添加依赖

```bash
go get github.com/fatih/color
```

### 8.2 更新依赖

```bash
go get -u ./...
go mod tidy
```

---

## 9. 项目里程碑

| 阶段 | 命令 | 目标 |
|------|------|------|
| **M0** | `gitee auth login` | 完成 OAuth 认证流程 |
| **M1** | `gitee repo list/create/view` | 仓库管理（CRUD） |
| **M2** | `gitee issue list/create/view` | Issue 管理 |
| **M3** | `gitee pr list/create/view/merge` | PR 管理 |
| **M4** | `gitee search` + `gitee api` | 搜索和原始 API |
| **M5** | Shell 补全 + 配置管理 | 完整 UX 打磨 |

---

## 10. 参考资料

- [Gitee API v5 文档](https://gitee.com/api/v5/swagger)
- [Gitee OAuth2 文档](https://gitee.com/api/v5/oauth_doc)
- [Cobra 官方文档](https://pkg.go.dev/github.com/spf13/cobra)
- [Viper 官方文档](https://pkg.go.dev/github.com/spf13/viper)
- [glab 项目源码](https://github.com/cli/go-gitlab) — 最佳参考实现
- [gh CLI 项目源码](https://github.com/cli/cli)