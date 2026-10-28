# gitee-cli 系统设计文档（System Design Document）

> ⚠️ **状态提示（v0.13.2，2026-10 维护轮）**：本文档是早期设计稿，部分内容**未实现或已过期**
> （例如 Viper 配置、OAuth2 设备流、`internal/auth` / `printer` / `i18n` 模块、goreleaser 发布链）。
> 当前实现与行为以代码、`gitee --help`、`gitee <cmd> --agent-help`、`gitee help reference`
> 为准；与现状差异最大的章节已在文中修正。


版本：1.0.0（设计稿；现状修订见页首横幅）| 技术栈：Go 1.27+ / Cobra / Resty / 手写 JSON 配置

---

## 1. 技术栈选型

| 组件 | 技术选型 | 版本 | 选型理由 |
|------|---------|------|---------|
| 语言 | Go | 1.27+（go.mod `go 1.27.3`） | 交叉编译简单，二进制无依赖，生态成熟 |
| CLI 框架 | Cobra | v1.10 | gh/glab 同款，帮助/补全/Flag 生成完善 |
| 配置管理 | 手写 JSON（`encoding/json`） | stdlib | 结构简单，避免为几层配置引入 Viper；原子写 + 0600 |
| HTTP 客户端 | resty | v2.17 | 链式 API、自动重试、请求/响应中间件 |
| 认证 | 个人访问令牌（PAT） | — | 请求头 `Authorization: token <PAT>`；**无 OAuth2 / x/oauth2 依赖** |
| 输出表格 | `tabwriter` (stdlib) | — | 统一走 `internal/output.NewTableWriter` |
| 颜色 | `github.com/fatih/color` | v1.19 | 跨平台 ANSI 颜色，支持 `NO_COLOR` |
| 测试 | `testing` + `testify` | v1.12 | 标准 + 断言库 |
| Mock HTTP | `httptest` (stdlib) | — | 内置服务器，零依赖 |

---

## 2. 项目目录结构

```
gitee-cli/
├── cmd/                          # Cobra 命令定义（每个子命令一个文件）
│   ├── root.go                   # 根命令、全局 Flag 定义
│   ├── version.go                # --version 实现
│   ├── auth/
│   │   ├── auth.go              # auth 父命令
│   │   ├── login.go             # auth login
│   │   ├── logout.go            # auth logout
│   │   ├── status.go            # auth status
│   │   ├── refresh.go           # auth refresh
│   │   └── token.go             # auth token
│   ├── repo/
│   │   ├── repo.go
│   │   ├── list.go
│   │   ├── create.go
│   │   ├── clone.go
│   │   ├── fork.go
│   │   ├── view.go
│   │   ├── edit.go
│   │   ├── delete.go
│   │   └── browse.go
│   ├── issue/
│   │   ├── issue.go
│   │   ├── list.go
│   │   ├── create.go
│   │   ├── view.go
│   │   ├── edit.go
│   │   ├── close.go
│   │   ├── reopen.go
│   │   ├── comment.go
│   │   └── label.go
│   ├── pr/
│   │   ├── pr.go
│   │   ├── list.go
│   │   ├── create.go
│   │   ├── view.go
│   │   ├── merge.go
│   │   ├── close.go
│   │   ├── reopen.go
│   │   ├── review.go
│   │   ├── diff.go
│   │   └── checks.go
│   ├── gist/
│   │   ├── gist.go
│   │   ├── list.go
│   │   ├── create.go
│   │   ├── view.go
│   │   ├── edit.go
│   │   ├── delete.go
│   │   ├── star.go
│   │   └── clone.go
│   ├── org/
│   │   ├── org.go
│   │   ├── list.go
│   │   ├── view.go
│   │   ├── member.go
│   │   └── repo.go
│   ├── search/
│   │   ├── search.go
│   │   ├── repos.go
│   │   ├── issues.go
│   │   └── users.go
│   ├── api/
│   │   └── raw.go               # 原始 API 访问
│   ├── config/
│   │   ├── config.go
│   │   ├── list.go
│   │   ├── set.go
│   │   ├── get.go
│   │   └── unset.go
│   └── completion/
│       └── shell.go
├── internal/                     # 内部包（不对外暴露）
│   ├── api/                     # HTTP 客户端（resty）：client / endpoints / errors / pagination
│   ├── alias/                   # 用户别名解析与存储
│   ├── atomicfile/              # 原子文件写（temp + chmod + rename）
│   ├── cmdkit/                  # 列表命令骨架（RunList）与通用选项
│   ├── config/                  # 手写 JSON 配置（settings.json，0600 / 0700）
│   ├── domain/                  # 跨命令共享业务逻辑（label / milestone / notification / webhook）
│   ├── extension/               # 动态扩展加载与存储
│   ├── output/                  # 统一 tabwriter（NewTableWriter）与 PrintJSON
│   ├── terminal/                # 终端交互（prompt）
│   └── utils/                   # 通用工具（repo 解析、flag、git、校验）
├── pkg/                         # 可复用公开包
│   └── types/
│       ├── repo.go              # Repository 类型
│       ├── issue.go             # Issue / Label / Milestone 类型
│       ├── pr.go                # PullRequest / Review 类型
│       ├── gist.go              # Gist / GistFile 类型
│       ├── org.go               # Organization / Member 类型
│       ├── user.go              # User / Email 类型
│       └── search.go            # SearchResult 类型
├── cmd/gitee/
│   └── main.go                  # 程序入口
├── go.mod
├── go.sum
├── scripts/                      # ci.sh / release.sh / publish-release.sh
└── README.md
```

---

### 2.1 实际目录结构（v0.13.x 现状）

上面的设计稿目录与当前代码**有差异**，以下为实测结构（`git ls-files` / `ls` 可复现）：

```
cmd/            # 每组一个包：agent alias api auth completion config extension gist issue
                # label milestone notification org pr release repo search ssh-key status
                # user webhook help_topics + root.go / runtime.go / cmdkit 辅助
internal/       # api alias atomicfile cmdkit config domain extension output terminal utils
                # （没有 auth/、printer/、i18n/；认证逻辑在 cmd/auth + internal/config）
pkg/types/      # API 数据模型
docs/           # 本设计文档 + philosophy.md + 多语 README
scripts/        # ci.sh / release.sh / publish-release.sh（发布全靠脚本，无 Makefile/goreleaser）
test/           # 功能测试 bash 脚本 + 校验脚本
```

---

## 3. 核心模块设计

### 3.1 API Client（`internal/api/client.go`）

```
┌─────────────────────────────────────────┐
│            Cobra Commands               │
│   (repo list, issue create, pr merge)   │
└──────────────────┬──────────────────────┘
                   │ 调用
┌──────────────────▼──────────────────────┐
│              API Client                  │
│  • Authorization header 自动注入         │
│  • 请求/响应日志（--verbose 模式）        │
│  • 5xx 自动重试（3次，指数退避）           │
│  • context.Context 超时控制              │
│  • 统一错误类型映射                       │
└──────────────────┬──────────────────────┘
                   │ HTTPS
┌──────────────────▼──────────────────────┐
│          Gitee REST API v5               │
│     https://gitee.com/api/v5/           │
└─────────────────────────────────────────┘
```

**接口定义：**

```go
// internal/api/client.go
type Client interface {
    // 通用请求（返回原始 Response）
    Do(ctx context.Context, method, path string, body interface{}) (*Response, error)
    // 语义化快捷方法
    Get(ctx context.Context, path string, result interface{}) error
    Post(ctx context.Context, path string, body, result interface{}) error
    Put(ctx context.Context, path string, body, result interface{}) error
    Patch(ctx context.Context, path string, body, result interface{}) error
    Delete(ctx context.Context, path string) error
    // 自动分页列表（遍历所有页，追加到 result slice）
    List(ctx context.Context, path string, opts *PageOptions, result interface{}) error
}

type Response struct {
    StatusCode int
    Headers    http.Header
    Body       []byte
}

type PageOptions struct {
    Page    int // 起始页（1-based）
    PerPage int // 每页数量（默认 30，最大 100）
    Limit   int // 总条数上限（0=不限制）
}
```

**关键实现细节：**

```go
// 自动注入 Authorization 头
client.SetHeader("Authorization", "token "+accessToken)

// 重试配置（resty）
client.SetRetryCount(3).
    SetRetryWaitTime(2 * time.Second).
    SetRetryMaxWaitTime(8 * time.Second).
    AddRetryCondition(func(r *resty.Response, err error) bool {
        return r.StatusCode() >= 500 || err != nil
    })

// 超时
client.SetTimeout(time.Duration(timeout) * time.Second)
```

---

### 3.2 认证模块（`cmd/auth/` + `internal/config/`）

> 设计稿原写作 `internal/auth/` + OAuth2 Device Flow；**实际没有该包，也没有 OAuth2**。
> 认证 = 个人访问令牌（PAT）：读取（`--token` > `GITEE_TOKEN` > `settings.json`）→
> `GET /user` 校验 → 可选写回 `settings.json`。

**凭证存储位置：**

```
~/.gitee-cli/
└── settings.json      # 统一存放所有用户设置与凭证（文件权限 0600，目录 0700）
```

**`~/.gitee-cli/settings.json` 实际结构（字段以 `internal/config/config.go` 为准）：**

```json
{
  "git": {
    "user": { "name": "<your-name>", "email": "<your-email>" }
  },
  "hosts": {
    "gitee.com": {
      "token": "<your-access-token>",
      "username": "<your-username>",
      "protocol": "https",
      "scopes": ["user_info", "projects", "pull_requests", "issues", "gists"]
    }
  },
  "preferences": {
    "output": "table",
    "editor": "",
    "browser": "",
    "prompt": "enabled",
    "pager": "",
    "protocol": "",
    "active_host": "gitee.com"
  }
}
```

> 注：`expires_at` 不在 `HostConfig` 结构体中，写进文件会被忽略。

> ⚠️ **安全须知**
> - 文件创建时强制 `0600`（仅属主可读写）；读取时若发现权限过宽会自动收紧并告警
> - `token` 字段为 Gitee Access Token，**绝对不可提交到版本控制**
> - CI/CD 中通过环境变量 `GITEE_TOKEN` 注入，而非写入文件
> - `--token` 会出现在 `ps` 与 shell history，一次性脚本之外不建议使用

**认证时序（真实实现）：**

```
Client (gitee-cli)                 Gitee API v5
      |                                  |
      | 1. 解析 token 优先级              |
      |    --token > GITEE_TOKEN > settings.json
      |                                  |
      | 2. GET /user                     |
      |    Authorization: token <PAT>    |
      |--------------------------------->|
      |    200 {login, id, ...}          |
      |<---------------------------------|
      |                                  |
      | 3. 仅 --token 显式传入时          |
      |    settings.json ← token (0600)  |
      |    目录 0700 / 文件 0600          |
```

`gitee auth refresh` 是占位实现：打印「Token refresh is not yet supported. Please re-authenticate with 'gitee auth login'.」后退出 0，不刷新也不重新校验；**不是** OAuth `refresh_token` 交换。

---

### 3.3 配置模块（`internal/config/`）

**三层配置优先级（从高到低）：**

```
1. CLI Flag（--token, --output, --api-url ...）
2. 环境变量（GITEE_TOKEN, GITEE_OUTPUT ...）
3. 配置文件（~/.gitee-cli/settings.json）
4. 内置默认值
```

**配置初始化（`internal/config/`）：**

```go
func Init() {
    // settings.json 路径：~/.gitee-cli/settings.json
    configDir := filepath.Join(os.Getenv("HOME"), ".gitee-cli")
    os.MkdirAll(configDir, 0700)

    viper.SetConfigName("settings")
    viper.SetConfigType("json")
    viper.AddConfigPath(configDir)
    viper.SetEnvPrefix("GITEE")
    viper.AutomaticEnv()

    // 默认值
    viper.SetDefault("preferences.output", "table")
    viper.SetDefault("preferences.prompt", "enabled")
    viper.SetDefault("hosts.gitee\\.com.protocol", "https")
    viper.SetDefault("api_url", "https://gitee.com/api/v5")
    viper.SetDefault("timeout", 30)
}

// 读取当前 host 的 token
func GetToken(hostname string) string {
    key := fmt.Sprintf("hosts.%s.token", hostname)
    return viper.GetString(key)
}

// 读取 git 全局用户信息（首次初始化时从 git config 同步）
func SyncGitConfig() {
    name, _ := exec.Command("git", "config", "--global", "user.name").Output()
    email, _ := exec.Command("git", "config", "--global", "user.email").Output()
    if n := strings.TrimSpace(string(name)); n != "" {
        viper.Set("git.user.name", n)
    }
    if e := strings.TrimSpace(string(email)); e != "" {
        viper.Set("git.user.email", e)
    }
    viper.WriteConfig()
}
```

---

### 3.4 输出模块（`internal/output/`）

> 设计稿原写作 `internal/printer/` + `Printer` 接口；**实际没有该包**。现状：
> `internal/output.NewTableWriter()` 统一创建 tabwriter，`output.PrintJSON()` 输出 JSON；
> `-o` 只支持 `table` / `json`（**没有 csv**，非法值报错退出）。

**表格输出规则：**
- 表头加粗（`fatih/color`，`NO_COLOR` / `--no-color` 时降级为纯文本）
- 时间字段按格式显示（部分是相对时间）
- 长字符串截断并添加 `…`（`utils.Truncate`）
- 统一 `text/tabwriter` 对齐，管道输出时自动退化为纯文本

---

### 3.5 分页处理（`internal/api/pagination.go`）

Gitee API v5 在响应头中返回分页信息（遵循 RFC 5988 Link Header）：

```
Link: <https://gitee.com/api/v5/user/repos?page=2>; rel="next",
      <https://gitee.com/api/v5/user/repos?page=10>; rel="last"
X-Total-Count: 287
```

**分页解析逻辑：**

```go
type PaginationInfo struct {
    NextPage  int
    LastPage  int
    TotalCount int
}

func ParseLinkHeader(header string) *PaginationInfo {
    // 解析 Link header，提取 next/last 页码
    // 解析 X-Total-Count
}

// 自动遍历所有分页
func (c *client) List(ctx context.Context, path string, opts *PageOptions, result interface{}) error {
    page := opts.Page
    for {
        url := fmt.Sprintf("%s?page=%d&per_page=%d", path, page, opts.PerPage)
        resp, err := c.Get(ctx, url, &pageResult)
        if err != nil {
            return err
        }
        // 追加结果到 result
        // 检查 Link header
        info := ParseLinkHeader(resp.Headers.Get("Link"))
        if info.NextPage == 0 || (opts.Limit > 0 && len(*result) >= opts.Limit) {
            break
        }
        page = info.NextPage
    }
    return nil
}
```

---

### 3.6 错误处理（`internal/api/errors.go`）

**错误类型层级：**

```go
// API 错误（HTTP 4xx/5xx）
type APIError struct {
    StatusCode int      `json:"-"`
    Message    string   `json:"message"`
    Errors     []FieldError `json:"errors,omitempty"`
}

type FieldError struct {
    Field    string `json:"field"`
    Code     string `json:"code"`
    Message  string `json:"message"`
}

func (e *APIError) Error() string {
    return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// HTTP 状态码 → 用户友好错误映射
func WrapHTTPError(statusCode int, body []byte) error {
    switch statusCode {
    case 401:
        return fmt.Errorf("authentication failed. Run 'gitee auth login'")
    case 403:
        return fmt.Errorf("insufficient permissions: %s", parseMessage(body))
    case 404:
        return fmt.Errorf("resource not found")
    case 422:
        return parseValidationError(body)
    default:
        return parseGenericError(statusCode, body)
    }
}
```

**命令层统一错误处理：**

```go
// cmd/root.go
func handleError(err error) {
    if err == nil {
        return
    }
    fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
    os.Exit(1)
}
```

---

## 4. 数据模型（`pkg/types/`）

```go
// pkg/types/repo.go
type Repository struct {
    ID              int64     `json:"id"`
    Name            string    `json:"name"`
    FullName        string    `json:"full_name"`
    Description     string    `json:"description"`
    Private         bool      `json:"private"`
    Internal        bool      `json:"internal"`
    Fork            bool      `json:"fork"`
    URL             string    `json:"url"`
    HTMLURL         string    `json:"html_url"`
    CloneURL        string    `json:"clone_url"`
    SSHURL          string    `json:"ssh_url"`
    Language        string    `json:"language"`
    StargazersCount int       `json:"stargazers_count"`
    ForksCount      int       `json:"forks_count"`
    OpenIssuesCount int       `json:"open_issues_count"`
    DefaultBranch   string    `json:"default_branch"`
    Owner           User      `json:"owner"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
    PushedAt        time.Time `json:"pushed_at"`
}

// pkg/types/issue.go
type Issue struct {
    Number      int         `json:"number"`
    ID          int64       `json:"id"`
    Title       string      `json:"title"`
    State       string      `json:"state"`
    Body        string      `json:"body"`
    HTMLURL     string      `json:"html_url"`
    Labels      []Label     `json:"labels"`
    Assignees   []User      `json:"assignees"`
    Milestone   *Milestone  `json:"milestone"`
    Comments    int         `json:"comments"`
    CreatedAt   time.Time   `json:"created_at"`
    UpdatedAt   time.Time   `json:"updated_at"`
    ClosedAt    *time.Time  `json:"closed_at"`
    User        User        `json:"user"`
}

type Label struct {
    ID    int64  `json:"id"`
    Name  string `json:"name"`
    Color string `json:"color"`
    URL   string `json:"url"`
}

// pkg/types/pr.go
type PullRequest struct {
    Number          int         `json:"number"`
    ID              int64       `json:"id"`
    Title           string      `json:"title"`
    State           string      `json:"state"`
    Body            string      `json:"body"`
    HTMLURL         string      `json:"html_url"`
    Merged          bool        `json:"merged"`
    MergedAt        *time.Time  `json:"merged_at"`
    Head            PRBranch    `json:"head"`
    Base            PRBranch    `json:"base"`
    User            User        `json:"user"`
    Assignees       []User      `json:"assignees"`
    Labels          []Label     `json:"labels"`
    CreatedAt       time.Time   `json:"created_at"`
    UpdatedAt       time.Time   `json:"updated_at"`
    Commits         int         `json:"commits"`
    Additions       int         `json:"additions"`
    Deletions       int         `json:"deletions"`
    ChangedFiles    int         `json:"changed_files"`
}

type PRBranch struct {
    Label  string `json:"label"`
    Ref    string `json:"ref"`
    SHA    string `json:"sha"`
    Repo   Repository `json:"repo"`
}

// pkg/types/user.go
type User struct {
    ID        int64  `json:"id"`
    Login     string `json:"login"`
    Name      string `json:"name"`
    Email     string `json:"email"`
    AvatarURL string `json:"avatar_url"`
    HTMLURL   string `json:"html_url"`
    Type      string `json:"type"`
    SiteAdmin bool   `json:"site_admin"`
}
```

---

## 5. 构建与发布

> 设计稿原写作 Makefile + `.goreleaser.toml`；**实际两者都不存在**，发布链路是 shell 脚本 +
> 两个 CI 后端（GitHub Actions / Gitee Go）共用 `scripts/ci.sh`。

### 5.1 构建与本地开发

```bash
# 安装到 $GOPATH/bin（日常开发用这个）
go install ./cmd/gitee

# 一次性本地构建（不装到 PATH）
go build -o gitee ./cmd/gitee

# 与 CI 完全一致的门禁
bash scripts/ci.sh     # gofmt / vet / golangci-lint / build / test -race / 交叉编译 / govulncheck

# 手动交叉编译
GOOS=linux  GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee
GOOS=windows GOARCH=amd64 go build -o dist/gitee-windows-amd64.exe ./cmd/gitee
```

### 5.2 发布链

```bash
git tag -a v0.13.2 -m v0.13.2 && git push origin v0.13.2
GITEE_TOKEN=... bash scripts/release.sh --publish
```

- `scripts/release.sh`：构建 5 平台产物（`gitee-<os>-<arch>.tar.gz` / Windows `.zip`）、
  生成 `SHA256SUMS.txt`、调用 `publish-release.sh` 创建 Gitee release 并上传资产（幂等）
- `scripts/publish-release.sh`：只做「创建 release + 上传资产」，复用 `gitee release create/upload`
- 版本号由 ldflags 注入；README 徽章由测试 `TestReadmeVersionMatchesVersionVar` 锁定
- 无 Homebrew tap、无 goreleaser

---

## 6. 安全设计

| 安全措施 | 实现方式 |
|---------|---------|
| 设置文件权限 | `~/.gitee-cli/settings.json` 创建时强制 `0600`（`os.OpenFile(..., 0600)`） |
| Token 不写入日志 | `--verbose` 模式下 Authorization header 脱敏输出为 `token ***` |
| Token 不提交 | `.gitignore_global` 中排除 `~/.gitee-cli/`；工具初始化时提示用户 |
| HTTPS | 默认只连 `https://` 站点；没有 `--insecure` 开关（不提供跳过证书校验的入口） |
| 证书验证 | 使用系统 CA bundle 与标准 `net/http` 校验 |
| 输入验证 | 仓库名、用户名通过正则 `^[a-zA-Z0-9_.-]+$` 校验 |
| CI/CD 安全 | Token 通过环境变量 `GITEE_TOKEN` 注入，不依赖文件 |
| 依赖安全 | CI 中运行 `govulncheck ./...` |
| SSRF 防护 | `--api-url` 只允许 `https://` 前缀 |

---

## 7. 扩展性设计

### 7.1 插件系统（v2.0 规划）

```go
// 命令注册时检查 PATH 中的 gitee-* 可执行文件（类似 gh extensions）
func RegisterExternalCommands(root *cobra.Command) {
    for _, ext := range discoverExtensions() {
        root.AddCommand(wrapExtensionCommand(ext))
    }
}
```

### 7.2 企业私有部署支持

通过 `--hostname` flag 或 `GITEE_HOSTNAME` 环境变量切换 API 基址：

```go
apiURL := fmt.Sprintf("https://%s/api/v5", hostname)
```

### 7.3 多账号支持（v2.0 规划）

`settings.json` 的 `hosts` 字段原生支持多 hostname：

```json
{
  "hosts": {
    "gitee.com": {
      "token": "<public-token>",
      "username": "<public-username>"
    },
    "enterprise.gitee.com": {
      "token": "<enterprise-token>",
      "username": "<enterprise-username>"
    }
  }
}
```
