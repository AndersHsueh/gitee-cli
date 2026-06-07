# gitee-cli System Design Document

Version: 1.0.0 | Tech Stack: Go 1.23+ / Cobra / Viper / Resty

---

## 1. Technology Selection

| Component | Technology | Version | Reason |
|-----------|------------|---------|--------|
| Language | Go | 1.23+ | Easy cross-compilation, zero binary dependencies, mature ecosystem |
| CLI Framework | Cobra | v1.8+ | Same as gh/glab, excellent help/completion/flag generation |
| Config Management | Viper | v1.18+ | YAML/ENV/Flag three-layer config, perfect Cobra integration |
| HTTP Client | resty | v2.x | Chainable API, auto-retry, request/response middleware |
| OAuth2 | `golang.org/x/oauth2` | latest | Standard implementation, supports Device Flow |
| Table Output | `tabwriter` (stdlib) | — | No extra dependencies, lightweight |
| Colors | `github.com/fatih/color` | v1.x | Cross-platform ANSI colors, supports `NO_COLOR` |
| Testing | `testing` + `testify` | v1.x | Standard + assertion library |
| Mock HTTP | `httptest` (stdlib) | — | Built-in server, zero dependencies |

---

## 2. Project Directory Structure

```
gitee-cli/
├── cmd/                          # Cobra command definitions (one file per subcommand)
│   ├── root.go                   # Root command, global flag definitions
│   ├── version.go                # --version implementation
│   ├── auth/
│   │   ├── auth.go              # auth parent command
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
│   │   └── raw.go               # Raw API access
│   ├── config/
│   │   ├── config.go
│   │   ├── list.go
│   │   ├── set.go
│   │   ├── get.go
│   │   └── unset.go
│   └── completion/
│       └── shell.go
├── internal/                     # Internal packages (not exported)
│   ├── api/
│   │   ├── client.go            # HTTP client wrapper (resty + middleware)
│   │   ├── endpoints.go         # API endpoint path constants
│   │   ├── errors.go            # API error types + HTTP status code mapping
│   │   └── pagination.go        # Pagination Link Header parsing
│   ├── auth/
│   │   ├── token.go             # Credential read/write (settings.json)
│   │   ├── oauth.go             # OAuth2 Device Flow implementation
│   │   └── config.go            # Token storage config
│   ├── config/
│   │   ├── config.go            # Viper config wrapper
│   │   └── migrate.go           # Config migration (version compatibility)
│   ├── printer/
│   │   ├── printer.go           # Printer interface definition
│   │   ├── table.go             # tabwriter table output
│   │   ├── json.go              # JSON pretty output
│   │   └── csv.go               # CSV output
│   ├── terminal/
│   │   ├── prompt.go            # Interactive prompts (ask/confirm)
│   │   ├── spinner.go           # Loading animation
│   │   └── browser.go           # Browser open (cross-platform)
│   └── i18n/
│       ├── i18n.go              # Translation function T()
│       ├── zh_CN.go             # Chinese language pack
│       └── en_US.go             # English language pack
├── pkg/                         # Reusable public packages
│   └── types/
│       ├── repo.go              # Repository type
│       ├── issue.go             # Issue / Label / Milestone types
│       ├── pr.go                # PullRequest / Review types
│       ├── gist.go              # Gist / GistFile types
│       ├── org.go               # Organization / Member types
│       ├── user.go              # User / Email types
│       └── search.go            # SearchResult type
├── cmd/gitee/
│   └── main.go                  # Program entry
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.toml
└── README.md
```

---

## 3. Core Module Design

### 3.1 API Client (`internal/api/client.go`)

```
┌─────────────────────────────────────────┐
│            Cobra Commands               │
│   (repo list, issue create, pr merge)   │
└──────────────────┬──────────────────────┘
                   │ call
┌──────────────────▼──────────────────────┐
│              API Client                  │
│  • Authorization header auto-injection   │
│  • Request/response logging (--verbose) │
│  • 5xx auto-retry (3x, exponential)     │
│  • context.Context timeout control       │
│  • Unified error type mapping           │
└──────────────────┬──────────────────────┘
                   │ HTTPS
┌──────────────────▼──────────────────────┐
│          Gitee REST API v5               │
│     https://gitee.com/api/v5/           │
└─────────────────────────────────────────┘
```

**Interface Definition:**

```go
// internal/api/client.go
type Client interface {
    // Generic request (returns raw Response)
    Do(ctx context.Context, method, path string, body interface{}) (*Response, error)
    // Semantic shortcut methods
    Get(ctx context.Context, path string, result interface{}) error
    Post(ctx context.Context, path string, body, result interface{}) error
    Put(ctx context.Context, path string, body, result interface{}) error
    Patch(ctx context.Context, path string, body, result interface{}) error
    Delete(ctx context.Context, path string) error
    // Auto-paginate list (traverse all pages, append to result slice)
    List(ctx context.Context, path string, opts *PageOptions, result interface{}) error
}

type Response struct {
    StatusCode int
    Headers    http.Header
    Body       []byte
}

type PageOptions struct {
    Page    int // Start page (1-based)
    PerPage int // Items per page (default 30, max 100)
    Limit   int // Max total items (0=unlimited)
}
```

**Key Implementation Details:**

```go
// Auto-inject Authorization header
client.SetHeader("Authorization", "token "+accessToken)

// Retry config (resty)
client.SetRetryCount(3).
    SetRetryWaitTime(2 * time.Second).
    SetRetryMaxWaitTime(8 * time.Second).
    AddRetryCondition(func(r *resty.Response, err error) bool {
        return r.StatusCode() >= 500 || err != nil
    })

// Timeout
client.SetTimeout(time.Duration(timeout) * time.Second)
```

---

### 3.2 Authentication Module (`internal/auth/`)

**Credential Storage Location:**

```
~/.gitee-cli/
└── settings.json      # Unified storage for all user settings and credentials (file mode 0600)
```

> Legacy path `~/.config/gitee/` is deprecated, auto-migrated on tool startup.

**`~/.gitee-cli/settings.json` Full Structure:**

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

> ⚠️ **Security Notice**
> - File created with mandatory permissions `0600` (owner read/write only)
> - `token` field is Gitee Access Token, **never commit to version control**
> - Add `~/.gitee-cli/` to `~/.gitignore_global`
> - For CI/CD, inject via environment variable `GITEE_TOKEN`, not file

**Multi-Host Support (Enterprise Private Deployment):**

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

**`git` Field Description:**

`git.user` in `settings.json` stays in sync with system `git config --global`, used to auto-fill commit authorship when creating commits. On first run, the tool reads global git config and writes to this field; can also be modified separately via `gitee config` command.

**OAuth2 Device Flow Sequence:**

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
      |  Display user_code to user|
      |  Open browser             |                        |
      |-------------------------------------------------------->|
      |                          |    User enters ABCD-1234   |
      |                          |<------------------------|
      |                          |    User clicks authorize   |
      |                          |<------------------------|
      |                          |                        |
      |  [Poll every 5s]         |                        |
      |  POST /oauth/device/token |                        |
      |  {device_code, grant_type}                        |
      |------------------------->|                        |
      |  {access_token,          |                        |
      |   token_type: "bearer"   |                        |
      |   scope, refresh_token}  |                        |
      |<-------------------------|                        |
      |                          |                        |
      |  Write to settings.json  |                        |
      |  GET /user → verify username |                        |
```

---

### 3.3 Configuration Module (`internal/config/`)

**Three-Layer Config Priority (high to low):**

```
1. CLI Flag (--token, --output, --api-url ...)
2. Environment Variable (GITEE_TOKEN, GITEE_OUTPUT ...)
3. Config File (~/.gitee-cli/settings.json)
4. Built-in defaults
```

**Config Initialization (`internal/config/`):**

```go
func Init() {
    // settings.json path: ~/.gitee-cli/settings.json
    configDir := filepath.Join(os.Getenv("HOME"), ".gitee-cli")
    os.MkdirAll(configDir, 0700)

    viper.SetConfigName("settings")
    viper.SetConfigType("json")
    viper.AddConfigPath(configDir)
    viper.SetEnvPrefix("GITEE")
    viper.AutomaticEnv()

    // Defaults
    viper.SetDefault("preferences.output", "table")
    viper.SetDefault("preferences.prompt", "enabled")
    viper.SetDefault("hosts.gitee\\.com.protocol", "https")
    viper.SetDefault("api_url", "https://gitee.com/api/v5")
    viper.SetDefault("timeout", 30)
}

// Read current host's token
func GetToken(hostname string) string {
    key := fmt.Sprintf("hosts.%s.token", hostname)
    return viper.GetString(key)
}

// Read git global user info (sync from git config on first init)
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

### 3.4 Output Module (`internal/printer/`)

**Printer Interface:**

```go
type Printer interface {
    PrintHeader(fields ...string)
    PrintRow(values ...string)
    PrintFooter(msg string)
    Flush() error
}

// Factory method based on --output flag
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

**Table Output Rules:**
- Header: `color.New(color.Bold).Sprint(field)`
- Time fields: call `humanize.Time(t)` for relative time
- Long strings: truncate at 60 chars with `…`
- Auto-detect terminal width (`term.GetSize()`)

---

### 3.5 Pagination (`internal/api/pagination.go`)

Gitee API v5 returns pagination info in response headers (RFC 5988 Link Header):

```
Link: <https://gitee.com/api/v5/user/repos?page=2>; rel="next",
      <https://gitee.com/api/v5/user/repos?page=10>; rel="last"
X-Total-Count: 287
```

**Pagination Parsing Logic:**

```go
type PaginationInfo struct {
    NextPage  int
    LastPage  int
    TotalCount int
}

func ParseLinkHeader(header string) *PaginationInfo {
    // Parse Link header, extract next/last page numbers
    // Parse X-Total-Count
}

// Auto-traverse all pages
func (c *client) List(ctx context.Context, path string, opts *PageOptions, result interface{}) error {
    page := opts.Page
    for {
        url := fmt.Sprintf("%s?page=%d&per_page=%d", path, page, opts.PerPage)
        resp, err := c.Get(ctx, url, &pageResult)
        if err != nil {
            return err
        }
        // Append results to result
        // Check Link header
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

### 3.6 Error Handling (`internal/api/errors.go`)

**Error Type Hierarchy:**

```go
// API Error (HTTP 4xx/5xx)
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

// HTTP status code → user-friendly error mapping
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

**Command Layer Unified Error Handling:**

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

## 4. Data Models (`pkg/types/`)

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

## 5. Build and Release

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

### 5.2 `.goreleaser.toml` Core Config

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

## 6. Security Design

| Security Measure | Implementation |
|------------------|-----------------|
| Config file permissions | `~/.gitee-cli/settings.json` created with mandatory `0600` (`os.OpenFile(..., 0600)`) |
| Token not in logs | `--verbose` mode sanitizes Authorization header as `token ***` |
| Token not committed | `~/.gitee-cli/` excluded in `.gitignore_global`; tool prompts user on init |
| HTTPS enforced | Enabled by default, `--insecure` must be explicitly set |
| Certificate validation | Use system CA bundle, `--insecure` skips |
| Input validation | Repo name, username validated via regex `^[a-zA-Z0-9_.-]+$` |
| CI/CD security | Token injected via environment variable `GITEE_TOKEN`, not file |
| Dependency security | `govulncheck ./...` runs in CI |
| SSRF protection | `--api-url` only allows `https://` prefix |

---

## 7. Extensibility Design

### 7.1 Plugin System (v2.0 Plan)

```go
// Check PATH for gitee-* executables on command registration (similar to gh extensions)
func RegisterExternalCommands(root *cobra.Command) {
    for _, ext := range discoverExtensions() {
        root.AddCommand(wrapExtensionCommand(ext))
    }
}
```

### 7.2 Enterprise Private Deployment Support

Switch API base via `--hostname` flag or `GITEE_HOSTNAME` environment variable:

```go
apiURL := fmt.Sprintf("https://%s/api/v5", hostname)
```

### 7.3 Multi-Account Support (v2.0 Plan)

`hosts` field in `settings.json` natively supports multiple hostnames:

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