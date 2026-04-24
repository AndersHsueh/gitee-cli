# gitee-cli 系统设计文档

## 1. 技术栈

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| 语言 | Go 1.23+ | 推荐 `glab` 项目为架构参考 |
| CLI 框架 | Cobra | 命令解析、帮助生成、补全 |
| 配置管理 | Viper | YAML 配置、Flag 绑定、环境变量 |
| HTTP 客户端 | `net/http`（stdlib）+ `resty` | 兼顾简单性和高级特性 |
| 认证 | `golang.org/x/oauth2` | OAuth2 设备码流程 |
| 输出 | `tabwriter` + 自定义 Printer | 表格输出 |
| 颜色 | `github.com/fatih/color` | 跨平台 ANSI 颜色 |
| 测试 | `testing` + `testify` | 单元测试和 Mock |

---

## 2. 项目结构

```
gitee-cli/
├── cmd/                          # Cobra 命令定义
│   ├── root.go                   # 根命令、全局 Flag
│   ├── auth/
│   │   ├── login.go             # auth login
│   │   ├── logout.go            # auth logout
│   │   └── status.go            # auth status
│   ├── repo/
│   │   ├── list.go
│   │   ├── create.go
│   │   ├── clone.go
│   │   ├── fork.go
│   │   ├── view.go
│   │   ├── edit.go
│   │   └── delete.go
│   ├── issue/
│   │   ├── list.go
│   │   ├── create.go
│   │   ├── view.go
│   │   ├── edit.go
│   │   ├── close.go
│   │   ├── reopen.go
│   │   └── comment.go
│   ├── pr/
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
│   │   ├── list.go
│   │   ├── create.go
│   │   ├── view.go
│   │   ├── edit.go
│   │   ├── delete.go
│   │   ├── star.go
│   │   └── clone.go
│   ├── org/
│   │   ├── list.go
│   │   ├── view.go
│   │   ├── member.go
│   │   └── repo.go
│   ├── search/
│   │   ├── repos.go
│   │   ├── issues.go
│   │   └── users.go
│   ├── api/
│   │   └── raw.go              # 原始 API 访问
│   ├── config/
│   │   ├── list.go
│   │   ├── set.go
│   │   └── get.go
│   └── completion/
│       └── shell.go
├── internal/                     # 内部包（不对外暴露）
│   ├── api/                     # Gitee API 客户端
│   │   ├── client.go           # HTTP 客户端封装
│   │   ├── endpoints.go        # API 端点定义
│   │   ├── errors.go           # 错误类型定义
│   │   └── pagination.go       # 分页处理
│   ├── auth/                    # 认证相关
│   │   ├── token.go            # Token 读写
│   │   ├── oauth.go            # OAuth2 设备码流程
│   │   └── config.go           # 凭证配置
│   ├── config/                  # 配置文件管理
│   │   ├── config.go           # Viper 配置封装
│   │   └── migrate.go          # 配置迁移
│   ├── printer/                # 输出格式化
│   │   ├── table.go            # 表格输出
│   │   ├── json.go             # JSON 输出
│   │   ├── csv.go              # CSV 输出
│   │   └── printer.go          # 接口定义
│   ├── terminal/               # 终端交互
│   │   ├── prompt.go           # 交互式提示
│   │   ├── spinner.go          # 加载动画
│   │   └── browser.go          # 浏览器打开
│   └── i18n/                   # 国际化
│       ├── zh_CN.go
│       └── en_US.go
├── pkg/                         # 可复用的公开包
│   └── types/                  # 公共类型定义
│       ├── repo.go
│       ├── issue.go
│       ├── pr.go
│       └── user.go
├── go.mod
├── go.sum
├── Makefile                    # 交叉编译脚本
├── .goreleaser.toml            # Release 自动化配置
└── README.md
```

---

## 3. 核心模块设计

### 3.1 API Client（`internal/api/client.go`）

```
                    ┌─────────────────────────────────┐
                    │          Cobra Commands          │
                    │  (repo list, issue create ...)   │
                    └──────────────┬──────────────────┘
                                   │
                    ┌──────────────▼──────────────────┐
                    │          API Client              │
                    │  · 自动注入 Authorization header │
                    │  · 处理分页（HasMore → NextPage）│
                    │  · 统一错误处理                  │
                    │  · 请求重试（3次，指数退避）      │
                    └──────────────┬──────────────────┘
                                   │
                    ┌──────────────▼──────────────────┐
                    │        Gitee REST API v5         │
                    │   https://gitee.com/api/v5/      │
                    └─────────────────────────────────┘
```

**接口设计：**
```go
type Client interface {
    Do(method, path string, body interface{}) (*Response, error)
    List(path string, result interface{}) error   // 自动分页
    Get(path string, result interface{}) error
    Post(path string, body, result interface{}) error
    Put(path string, body, result interface{}) error
    Delete(path string) error
}
```

**关键特性：**
- 每个请求自动注入 `Authorization: token <access_token>` header
- 读取 `X-Total-Count` 和 `Link` header 自动处理分页
- HTTP 4xx/5xx 映射为 `*api.Error` 类型（带状态码和消息）
- 所有网络调用支持 `context.Context` 取消

---

### 3.2 认证模块（`internal/auth/`）

```
~/.gitee-cli/
├── config.yml         # Viper 配置（协议、编辑器等）
└── credentials.yml    # 加密存储的 Token
```

**Token 存储安全：**
- 凭证文件权限设置为 `0600`
- Token 明文存储（OAuth2 无刷新 Token 时使用）
- 可选：使用 `keyring` 系统钥匙串（macOS Keychain / Linux secret-tool）

**OAuth2 设备码流程：**
```
1. POST /oauth/authorizations/device/code
   → 获取 device_code, user_code, verification_url

2. 打开浏览器（或显示 device_code）

3. 轮询 POST /oauth/authorizations/device/token
   → 直到 authorized=true

4. 获得 access_token，写入 ~/.gitee-cli/settings.json
```

---

### 3.3 配置模块（`internal/config/`）

使用 Viper 管理三层配置（优先级从高到低）：

1. **Flag / CLI 参数**（优先级最高）
2. **环境变量**（`GITEE_TOKEN`, `GITEE_HOSTNAME`, `GITEE_OUTPUT`）
3. **配置文件**（`~/.gitee-cli/settings.json`）

默认配置路径：`~/.gitee-cli/settings.json`

---

### 3.4 输出模块（`internal/printer/`）

实现统一的 `Printer` 接口，支持多格式切换：

```go
type Printer interface {
    PrintHeader(...string)
    PrintRow(...string)
    PrintFooter()
    Flush()
}
```

实现类：`TablePrinter`、`JsonPrinter`、`CsvPrinter`

---

## 4. 数据模型（`pkg/types/`）

所有 API 响应模型定义在 `pkg/types/`，对标 Gitee API v5 返回字段的 JSON 结构：

```go
// pkg/types/repo.go
type Repository struct {
    ID       int64   `json:"id"`
    Name     string  `json:"name"`
    FullName string  `json:"full_name"`
    Owner    User    `json:"owner"`
    Private  bool    `json:"private"`
    // ...
}

// pkg/types/issue.go
type Issue struct {
    Number    int      `json:"number"`
    Title     string   `json:"title"`
    State     string   `json:"state"`
    Labels    []Label  `json:"labels"`
    Assignees []User   `json:"assignees"`
    Body      string   `json:"body"`
    // ...
}
```

---

## 5. 错误处理设计

```
internal/api/errors.go

type Error struct {
    StatusCode int
    Message    string      // API 返回的错误消息
    Errors     []FieldError // 字段级错误（表单验证失败时）
}

type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}
```

命令层统一使用 `factory.Check(err)` 包装：
```go
func Check(err error) {
    if err != nil {
        ui.Error(err.Error())
        os.Exit(1)
    }
}
```

---

## 6. 分页处理

```go
// internal/api/pagination.go
type PageOptions struct {
    Page    int
    PerPage int    // 默认 30，最大 100
}

func (c *Client) List(path string, opts PageOptions, result interface{}) error {
    url := fmt.Sprintf("%s?page=%d&per_page=%d", path, opts.Page, opts.PerPage)
    // 自动处理 Link header 中的 next/prev/last
    // 将结果追加到 result slice
}
```

---

## 7. 构建与发布（Makefile）

```makefile
.PHONY: build test lint install cross-build release

build:
    go build -o bin/gitee ./cmd/gitee

test:
    go test ./... -v -count=1

lint:
    golangci-lint run

cross-build:
    GOOS=linux GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee
    GOOS=darwin GOARCH=amd64 go build -o dist/gitee-darwin-amd64 ./cmd/gitee
    GOOS=windows GOARCH=amd64 go build -o dist/gitee-windows-amd64.exe ./cmd/gitee

install:
    go install ./cmd/gitee

release:
    goreleaser release --snapshot --clean
```

---

## 8. 扩展性设计

### 8.1 命令注册机制

使用 Cobra 的 `AddCommand()` 动态注册子命令，支持插件化扩展：

```go
// 未来可扩展为插件加载
func RegisterPlugins(root *cobra.Command) {
    for _, p := range plugin.Plugins() {
        root.AddCommand(p.Commands()...)
    }
}
```

### 8.2 API 版本兼容性

- 所有 API 调用通过 `ApiBaseUrl` 配置项统一管理
- 默认 `https://gitee.com/api/v5`
- 支持 Gitee 企业私有部署（配置 `GITEE_HOSTNAME`）

---

## 9. 安全设计

| 安全措施 | 实现 |
|---------|------|
| Token 存储 | 文件权限 `0600`，不写入日志 |
| HTTPS | 强制使用 HTTPS，忽略证书错误需显式 `--insecure` |
| 输入验证 | 所有用户输入经过 `flag-validation` 库校验 |
| 依赖安全 | 使用 `go mod verify` + `govulncheck` 扫描漏洞 |