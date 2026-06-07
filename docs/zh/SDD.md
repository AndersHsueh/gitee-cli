# gitee-cli 系统设计文档（System Design Document）

版本：1.0.0 | 技术栈：Go 1.23+ / Cobra / Viper / Resty

---

## 1. 技术栈选型

| 组件 | 技术选型 | 版本 | 选型理由 |
|------|---------|------|---------|
| 语言 | Go | 1.23+ | 交叉编译简单，二进制无依赖，生态成熟 |
| CLI 框架 | Cobra | v1.8+ | gh/glab 同款，帮助/补全/Flag 生成完善 |
| 配置管理 | Viper | v1.18+ | YAML/ENV/Flag 三层配置，与 Cobra 完美集成 |
| HTTP 客户端 | resty | v2.x | 链式 API、自动重试、请求/响应中间件 |
| OAuth2 | `golang.org/x/oauth2` | latest | 标准实现，支持 Device Flow |
| 输出表格 | `tabwriter` (stdlib) | — | 无额外依赖，轻量 |
| 颜色 | `github.com/fatih/color` | v1.x | 跨平台 ANSI 颜色，支持 `NO_COLOR` |
| 测试 | `testing` + `testify` | v1.x | 标准 + 断言库 |
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
│   ├── api/
│   │   ├── client.go            # HTTP 客户端封装（resty + 中间件）
│   │   ├── endpoints.go         # API 端点路径常量
│   │   ├── errors.go            # API 错误类型 + HTTP 状态码映射
│   │   └── pagination.go        # 分页 Link Header 解析
│   ├── auth/
│   │   ├── token.go             # 凭证读写（settings.json）
│   │   ├── oauth.go             # OAuth2 Device Flow 实现
│   │   └── config.go            # Token 存储配置
│   ├── config/
│   │   ├── config.go            # Viper 配置封装
│   │   └── migrate.go           # 配置迁移（版本兼容）
│   ├── printer/
│   │   ├── printer.go           # Printer 接口定义
│   │   ├── table.go             # tabwriter 表格输出
│   │   ├── json.go              # JSON 美化输出
│   │   └── csv.go               # CSV 输出
│   ├── terminal/
│   │   ├── prompt.go            # 交互式提示（询问/确认）
│   │   ├── spinner.go           # 加载动画
│   │   └── browser.go           # 浏览器打开（跨平台）
│   └── i18n/
│       ├── i18n.go              # 翻译函数 T()
│       ├── zh_CN.go             # 中文语言包
│       └── en_US.go             # 英文语言包
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
├── Makefile
├── .goreleaser.toml
└── README.md
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

### 3.2 认证模块（`internal/auth/`）

**凭证存储位置：**

```
~/.gitee-cli/
└── settings.json      # 统一存放所有用户设置与凭证（文件权限 0600）
```

> 旧路径 `~/.config/gitee/` 已废弃，工具启动时自动迁移。

**`~/.gitee-cli/settings.json` 完整结构：**

```json
{
  "git": {
    "user": {
      "name": "<your-name>",
      "email": "<your-email>"
    }
  },
  "hosts": {
    "gitee.com": {
      "token": "<your-access-token>",
      "username": "<your-username>",
      "protocol": "https",
      "scopes": [
        "user_info",
        "projects",
        "pull_requests",
        "issues",
        "gists"
      ],
      "expires_at": ""
    }
  },
  "preferences": {
    "output": "table",
    "editor": "",
    "browser": "",
    "prompt": "enabled",
    "pager": ""
  }
}
```

> ⚠️ **安全须知**
> - 文件创建时强制设置权限 `0600`（仅属主可读写）
> - `token` 字段为 Gitee Access Token，**绝对不可提交到版本控制**
> - 请将 `~/.gitee-cli/` 加入 `~/.gitignore_global`
> - 如需在 CI/CD 中使用，应通过环境变量 `GITEE_TOKEN` 注入，而非写入文件

**多 Host 支持（企业私有部署）：**

```json
{
  "hosts": {
    "gitee.com": {
      "token": "<public-token>",
      "username": "<public-username>"
    },
    "your-company.gitee.com": {
      "token": "<enterprise-token>",
      "username": "<enterprise-username>",
      "protocol": "ssh"
    }
  }
}
```

**`git` 字段说明：**

`settings.json` 中的 `git.user` 与系统 `git config --global` 保持同步，用于在创建 commit 时自动填充署名信息。工具首次运行时会读取全局 git 配置并写入此字段；也可通过 `gitee config` 命令单独修改。

**OAuth2 Device Flow 时序：**

```
Client (gitee-cli)          Gitee Server           User Browser
      |                          |                        |
      |  POST /oauth/device/code |                        |
      |  {client_id, scope}      |                        |
      |------------------------->|                        |
      |  {device_code,           |                        |
      |   user_code: "ABCD-1234" |                        |
      |   verification_uri,      |                        |
      |   expires_in: 900,       |                        |
      |   interval: 5}           |                        |
      |<-------------------------|                        |
      |                          |                        |
      | 显示 user_code 给用户     |                        |
      | 打开浏览器                |                        |
      |-------------------------------------------------------->|
      |                          |    用户输入 ABCD-1234   |
      |                          |<------------------------|
      |                          |    用户点击授权          |
      |                          |<------------------------|
      |                          |                        |
      |  [轮询每 5s]              |                        |
      |  POST /oauth/device/token|                        |
      |  {device_code, grant_type}                        |
      |------------------------->|                        |
      |  {access_token,          |                        |
      |   token_type: "bearer"   |                        |
      |   scope, refresh_token}  |                        |
      |<-------------------------|                        |
      |                          |                        |
      | 写入 settings.json       |                        |
      | GET /user → 验证用户名   |                        |
```

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

### 3.4 输出模块（`internal/printer/`）

**Printer 接口：**

```go
type Printer interface {
    PrintHeader(fields ...string)
    PrintRow(values ...string)
    PrintFooter(msg string)
    Flush() error
}

// 根据 --output flag 工厂方法
func New(format string, w io.Writer) Printer {
    switch format {
    case "json":
        return &JsonPrinter{w: w}
    case "csv":
        return &CsvPrinter{w: w}
    default:
        return &TablePrinter{w: tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)}
    }
}
```

**表格输出规则：**
- 表头：`color.New(color.Bold).Sprint(field)`
- 时间字段：调用 `humanize.Time(t)` 输出相对时间
- 长字符串：超过 60 字符截断并添加 `…`
- 自动检测终端宽度（`term.GetSize()`）

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

### 5.1 Makefile

```makefile
.PHONY: build test lint install cross-build release clean

BINARY_NAME := gitee
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -ldflags="-s -w -X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/gitee

test:
	go test ./... -v -count=1 -race

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) ./cmd/gitee

cross-build:
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/gitee-linux-amd64 ./cmd/gitee
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o dist/gitee-linux-arm64 ./cmd/gitee
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/gitee-darwin-amd64 ./cmd/gitee
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/gitee-darwin-arm64 ./cmd/gitee
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/gitee-windows-amd64.exe ./cmd/gitee

release:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin/ dist/ coverage.out coverage.html
```

### 5.2 `.goreleaser.toml` 核心配置

```toml
project_name = "gitee-cli"

builds:
  - id: gitee
    main: ./cmd/gitee
    binary: gitee
    env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - "-s -w -X main.version={{.Version}}"

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

brews:
  - name: gitee
    homepage: "https://gitee.com/gitee-cli"
    tap:
      owner: gitee
      name: homebrew-tap
```

---

## 6. 安全设计

| 安全措施 | 实现方式 |
|---------|---------|
| 设置文件权限 | `~/.gitee-cli/settings.json` 创建时强制 `0600`（`os.OpenFile(..., 0600)`） |
| Token 不写入日志 | `--verbose` 模式下 Authorization header 脱敏输出为 `token ***` |
| Token 不提交 | `.gitignore_global` 中排除 `~/.gitee-cli/`；工具初始化时提示用户 |
| HTTPS 强制 | 默认启用，`--insecure` 需显式开启 |
| 证书验证 | 使用系统 CA bundle，`--insecure` 跳过 |
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
