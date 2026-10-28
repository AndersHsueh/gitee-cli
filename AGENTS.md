# AGENTS.md

## 项目概述

**gitee-cli** — Gitee 官方命令行工具，功能对标 GitHub `gh`。用于在终端操作 Gitee 仓库、Issue、PR、代码片段等资源。

- **语言**: Go 1.27+（`go.mod` 为 `go 1.27.3`）
- **CLI 框架**: Cobra v1.10
- **HTTP 客户端**: go-resty/resty v2
- **API 版本**: Gitee API v5
- **当前版本**: 以 `cmd/version.go` 为唯一来源（本维护轮为 v0.13.x）
- **认证**: 个人访问令牌（PAT），请求头 `Authorization: token <PAT>`；无 OAuth2

### 架构分层

```
cmd/            → Cobra 命令定义（每个命令组一个子包，如 cmd/repo/），另有
                  root.go/runtime.go（全局 flag 与 Runtime 注入）、cmdkit 辅助、
                  agent_help.go（--agent-help 矩阵）、help_topics/（help 主题）
internal/api/   → Gitee API v5 客户端（client + endpoints + errors + pagination）
internal/config/→ 手写 JSON 配置（~/.gitee-cli/settings.json，0600）
internal/cmdkit/→ 列表命令骨架 RunList（分页/输出/空结果统一）
internal/domain/→ 跨命令共享业务逻辑（label/milestone/notification/webhook）
internal/output/→ 统一 tabwriter（NewTableWriter）与 PrintJSON
internal/utils/ → 通用工具函数（Truncate, ParseOwnerRepo, ResolveRepo）
internal/alias|extension|atomicfile|terminal → 别名、动态扩展、原子写、终端交互
pkg/types/      → API 数据模型（Repository, Issue, PR, User 等结构体）
cmd/gitee/      → 程序入口（main.go 注册所有子命令）
```

### 命令注册模式

`cmd/gitee/main.go` 调用 `cmd.RegisterCommands(...)` 将各模块顶层 `*cobra.Command` 注册到 rootCmd；`cmd.RegisterDynamicCommands` 再挂载 `~/.gitee-cli/extensions/` 里的外部扩展（`--no-extensions`/`GITEE_NO_EXTENSIONS` 可跳过）；最后由 `help_topics.RegisterHelpTopics` 注册 help 主题。每个模块在自己的包内通过 `init()` 把子命令挂到组命令下。

命令层用 `cmd.RuntimeFromCmd(cobraCmd)` 取运行时上下文（`Runtime` 显式注入，替代旧的 `cmd.Global*`），再用 `cmd.NewAPIClientFrom(rt)` / `cmd.NewCmdkitOptionsFrom(rt, client)` 构造依赖。

## 构建与命令

```bash
# 构建并安装到 $GOPATH/bin（日常开发用这个）
go install ./cmd/gitee

# 一次性本地构建（不进 PATH）
go build -o gitee ./cmd/gitee

# 与 CI 完全一致的质量门禁（gofmt/vet/lint/build/test -race/交叉编译/govulncheck）
bash scripts/ci.sh          # 全量
bash scripts/ci.sh --fast   # 跳过交叉编译 + govulncheck

# 运行单元测试（httptest mock，不依赖真实网络）
go test ./... -race -count=1

# 运行功能测试（需先编译并登录）
./test/run.sh              # 全部模块
./test/run.sh repo issue   # 指定模块

# Lint（.golangci.yml 是 v2 schema）
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
- **输出格式**: 只支持 `--output table|json`（**无 csv**，非法值报错退出）；table 统一用 `internal/output.NewTableWriter()`，json 用 `output.PrintJSON`
- **API 路径**: 在 `internal/api/endpoints.go` 中用函数构建（如 `ReposList(owner)`, `PRsGet(owner, repo, number)`）
- **分页**: 列表命令统一走 `cmdkit.RunList` / `api.FetchListOrAll`，不要手写 paginate 分支或 per_page clamp
- **agent-help**: 新增命令必须写 11 维度 `--agent-help` 文档并加入 `cmd/agent_help.go` 矩阵；契约测试会失败
- **配置/认证**: 配置文件固定 `~/.gitee-cli/settings.json`（手写 JSON，无 Viper）；认证是 PAT（无 OAuth2 设备流）

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

- **Token 存储**: `~/.gitee-cli/settings.json`，文件权限 `0600`，目录权限 `0700`（读取时若权限过宽会自动收紧并告警）
- **Token 解析优先级**: `--token` 参数 > `GITEE_TOKEN` 环境变量 > settings 文件
- **避免 `--token` 明文**: 命令行 token 会进 `ps` 与 shell history，脚本里优先用 `GITEE_TOKEN`
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
| `--token` | 覆盖配置的 access token（建议用 `GITEE_TOKEN`） |
| `--hostname` | Gitee 主机名（默认 gitee.com，支持企业私有部署） |
| `--api-url` | API 基础 URL（默认 https://{hostname}/api/v5） |
| `--output/-o` | 输出格式 table/json（无 csv，非法值报错） |
| `--agent-help` | 输出 Agent 向结构化长文档（部分命令；`-o json` 可结构化） |
| `--verbose/-v` | 显示 HTTP 请求/响应详情（stderr） |
| `--timeout` | HTTP 超时秒数（默认 30） |
| `-R, --repo` | 目标仓库 owner/repo |
| `--no-color` | 禁用彩色输出（也认 `NO_COLOR`） |
| `--no-extensions` | 跳过外部扩展加载 |
| `--paginate/-P` | 遍历所有分页结果 |
| `--page/--per-page/--limit` | 分页控制（`--limit` 默认 30，`0` 表示不限） |

环境变量共 13 个（`GITEE_TOKEN` / `GITEE_HOSTNAME` / `GITEE_API_URL` / `GITEE_OUTPUT` /
`GITEE_VERBOSE` / `GITEE_TIMEOUT` / `GITEE_LIMIT` / `GITEE_PAGINATE` /
`GITEE_NO_EXTENSIONS` / `NO_COLOR` / `GITEE_ALIASES_FILE` / `GITEE_EXT_DIR` / `BROWSER`），
以 `gitee help environment` 为准。

### API 客户端

- 优先通过 `cmd.NewAPIClientFrom(rt)` 创建（`Runtime` 携带 token/baseURL/超时）；`cmd.NewAPIClient()` 是兼容旧路径
- 内置重试: 3 次，等待 2-8 秒，**仅 GET** 对 5xx/网络错误重试（写操作不重试，避免重复创建）
- 分页: `api.FetchAllPages` / `api.FetchListOrAll` 按 page/per_page 循环，不解析 Link header
- 写路径注意: 部分 Gitee v5 端点只接受 formData（如 issue close/reopen/edit），用 `Client.PatchForm`
