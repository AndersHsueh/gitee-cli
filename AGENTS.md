# AGENTS.md

## 项目概述

**gitee-cli** — Gitee 官方命令行工具，功能对标 GitHub `gh`。用于在终端操作 Gitee 仓库、Issue、PR、代码片段等资源。

- **语言**: Go 1.25+
- **CLI 框架**: Cobra
- **HTTP 客户端**: go-resty/resty v2
- **API 版本**: Gitee API v5
- **当前版本**: v0.2.0

### 架构分层

```
cmd/           → Cobra 命令定义（每功能一个子目录，如 cmd/repo/）
internal/api/  → Gitee API v5 客户端封装（Client + endpoints + errors + pagination）
internal/config/ → 配置管理（~/.gitee-cli/settings.json 读写）
internal/utils/  → 通用工具函数（Truncate, ParseOwnerRepo）
pkg/types/     → API 数据模型（Repository, Issue, PR, User 等结构体）
cmd/gitee/     → 程序入口（main.go 注册所有子命令）
```

### 命令注册模式

`cmd/gitee/main.go` 调用 `cmd.RegisterCommands()` 将各模块的顶层 `*cobra.Command` 注册到 rootCmd。每个模块（如 repo、issue、pr）在自己的包内通过 `init()` 将子命令挂载到模块命令下。

## 构建与命令

```bash
# 构建
go build -o gitee ./cmd/gitee

# 运行单元测试
go test ./... -v -count=1

# 运行功能测试（需先编译并登录）
./test/run.sh              # 全部模块
./test/run.sh repo issue   # 指定模块

# Lint
golangci-lint run ./...

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee

# 依赖管理
go mod tidy
```

## 代码风格

- **Commit 规范**: Conventional Commits — `feat:`, `fix:`, `docs:`, `test:`, `chore:`, `refactor:`
- **Go 命名**: 遵循 Go 标准约定 — 包名小写单词、导出用 PascalCase、未导出用 camelCase
- **命令组织**: 每个功能模块在 `cmd/` 下独立目录，每个子命令一个文件（如 `cmd/repo/list.go`）
- **错误处理**: 命令层使用 `RunE` 返回 error，由 rootCmd 统一输出红色错误信息
- **输出格式**: 支持 `--output table|json|csv`，table 用 `text/tabwriter`，json 用 `json.MarshalIndent`
- **API 路径**: 在 `internal/api/endpoints.go` 中用函数构建（如 `ReposList(owner)`, `PRsGet(owner, repo, number)`）

## 测试

### 单元测试

- 框架: `testing` + `github.com/stretchr/testify`（assert/require）
- 位置: 与源码同包，`*_test.go` 文件（如 `internal/api/client_test.go`）
- HTTP mock: 使用 `net/http/httptest` 创建测试服务器
- 运行: `go test ./... -v -count=1`

### 功能测试

- 位置: `test/` 目录，每个模块一个 bash 脚本（`test/repo.sh`, `test/auth.sh` 等）
- 入口: `test/run.sh`，支持 `GITEE_CLI_BINARY` 环境变量指定二进制路径
- 前置条件: 需要先编译二进制并完成 `gitee auth login`
- 风格: 使用 pass/fail/skip 计数器，彩色输出结果

## 安全

- **Token 存储**: `~/.gitee-cli/settings.json`，文件权限 `0600`，目录权限 `0700`
- **Token 解析优先级**: `--token` 参数 > `GITEE_TOKEN` 环境变量 > settings 文件
- **认证头格式**: `Authorization: token <access_token>`（注意不是 Bearer）
- **危险操作**: `repo delete` 等命令需安全确认（`--yes` 参数）
- **API 错误处理**: 401 自动提示 `gitee auth login`，403/404/422 有专门错误消息

## 配置

### 配置文件

- 路径: `~/.gitee-cli/settings.json`
- 示例: `settings.json.sample`
- 结构: `git.user`（用户名/邮箱）、`hosts`（多 host 支持，含 token/username/protocol/scopes）、`preferences`（output/editor/browser/pager 等偏好）

### 全局 Flag

| Flag | 说明 |
|------|------|
| `--token` | 覆盖配置的 access token |
| `--hostname` | Gitee 主机名（默认 gitee.com，支持企业私有部署） |
| `--api-url` | API 基础 URL（默认 https://{hostname}/api/v5） |
| `--output/-o` | 输出格式 table/json/csv |
| `--verbose/-v` | 显示 HTTP 请求/响应详情 |
| `--timeout` | HTTP 超时秒数（默认 30） |
| `--repo` | 目标仓库 owner/repo |
| `--no-color` | 禁用彩色输出 |
| `--paginate/-P` | 遍历所有分页结果 |
| `--page/--per-page/--limit` | 分页控制 |

### API 客户端

- 通过 `cmd.NewAPIClient()` 创建，自动解析 token 和 baseURL
- 内置重试: 3 次，等待 2-8 秒，仅对 5xx 或网络错误重试
- 分页: 解析 Link header（`ParseLinkHeader`）和 X-Total-Count header（`GetTotalCount`）
