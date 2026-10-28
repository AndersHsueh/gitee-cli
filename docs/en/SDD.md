# gitee-cli System Design Document

> ⚠️ **Status notice (v0.13.2, 2026-10 maintenance round)**: This document is an early design
> draft; parts of it are **not implemented or out of date** (e.g. Viper config, OAuth2 device
> flow, `internal/auth` / `printer` / `i18n` modules, goreleaser release chain). The authoritative
> implementation and behavior are the code, `gitee --help`, `gitee <cmd> --agent-help` and
> `gitee help reference`. Sections that diverge most are corrected inline.


Version: 1.0.0 (design draft; see banner) | Tech Stack: Go 1.27+ / Cobra / Resty / hand-written JSON config

---

## 1. Technology Selection

| Component | Technology | Version | Reason |
|-----------|------------|---------|--------|
| Language | Go | 1.27+ (`go 1.27.3` in go.mod) | Easy cross-compilation, zero binary dependencies, mature ecosystem |
| CLI Framework | Cobra | v1.10 | Same as gh/glab, excellent help/completion/flag generation |
| Config Management | hand-written JSON (`encoding/json`) | stdlib | Simple structure; avoids Viper for a few layers; atomic write + 0600 |
| HTTP Client | resty | v2.17 | Chainable API, auto-retry, request/response middleware |
| Auth | Personal access token (PAT) | — | `Authorization: token <PAT>` header; **no OAuth2 / x/oauth2 dependency** |
| Table Output | `tabwriter` (stdlib) | — | Unified through `internal/output.NewTableWriter` |
| Colors | `github.com/fatih/color` | v1.19 | Cross-platform ANSI colors, supports `NO_COLOR` |
| Testing | `testing` + `testify` | v1.12 | Standard + assertion library |
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
│   ├── api/                     # HTTP client (resty): client / endpoints / errors / pagination
│   ├── alias/                   # User alias resolution and storage
│   ├── atomicfile/              # Atomic file write (temp + chmod + rename)
│   ├── cmdkit/                  # List-command skeleton (RunList) + shared options
│   ├── config/                  # Hand-written JSON config (settings.json, 0600 / 0700)
│   ├── domain/                  # Cross-command business logic (label / milestone / notification / webhook)
│   ├── extension/               # Dynamic extension loading and storage
│   ├── output/                  # Unified tabwriter (NewTableWriter) and PrintJSON
│   ├── terminal/                # Terminal interaction (prompt)
│   └── utils/                   # Shared helpers (repo resolution, flags, git, validation)
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
├── scripts/                      # ci.sh / release.sh / publish-release.sh
└── README.md
```

---

### 2.1 Actual Directory Structure (v0.13.x)

The design draft tree above **differs from the current code**. Measured structure
(reproducible with `git ls-files` / `ls`):

```
cmd/            # one package per group: agent alias api auth completion config extension gist
                # issue label milestone notification org pr release repo search ssh-key status
                # user webhook help_topics + root.go / runtime.go / cmdkit helpers
internal/       # api alias atomicfile cmdkit config domain extension output terminal utils
                # (no auth/, printer/, i18n/; auth logic lives in cmd/auth + internal/config)
pkg/types/      # API data models
docs/           # this design doc + philosophy.md + multilingual READMEs
scripts/        # ci.sh / release.sh / publish-release.sh (no Makefile / goreleaser)
test/           # functional bash tests + doc-example verifier
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

### 3.2 Authentication Module (`cmd/auth/` + `internal/config/`)

> The draft said `internal/auth/` + OAuth2 Device Flow; **no such package exists and there is no
> OAuth2**. Auth = personal access token (PAT): resolve (`--token` > `GITEE_TOKEN` >
> `settings.json`) → verify with `GET /user` → optionally persist back to `settings.json`.

**Credential storage location:**

```
~/.gitee-cli/
└── settings.json      # unified storage for user settings and credentials (file 0600, dir 0700)
```

**Actual `~/.gitee-cli/settings.json` structure (fields per `internal/config/config.go`):**

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

> Note: `expires_at` is not part of `HostConfig`; it is ignored on load.

> ⚠️ **Security notes**
> - Files are created with `0600`; overly permissive files are tightened with a warning on load
> - `token` is a Gitee access token and **must never be committed**
> - CI/CD should inject `GITEE_TOKEN` instead of writing files
> - `--token` shows up in `ps` and shell history; prefer `GITEE_TOKEN` outside one-off scripts

**Authentication sequence (real implementation):**

```
Client (gitee-cli)                 Gitee API v5
      |                                  |
      | 1. Resolve token priority        |
      |    --token > GITEE_TOKEN > settings.json
      |                                  |
      | 2. GET /user                     |
      |    Authorization: token <PAT>    |
      |--------------------------------->|
      |    200 {login, id, ...}          |
      |<---------------------------------|
      |                                  |
      | 3. Only when --token was passed  |
      |    settings.json <- token (0600) |
      |    dir 0700 / file 0600          |
```

`gitee auth refresh` is a stub: it prints "Token refresh is not yet supported. Please
re-authenticate with 'gitee auth login'." and exits 0. It refreshes nothing and is **not** an
OAuth `refresh_token` exchange.

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

### 3.4 Output Module (`internal/output/`)

> The draft said `internal/printer/` + a `Printer` interface; **no such package exists**. Current
> state: `internal/output.NewTableWriter()` creates the tabwriter, `output.PrintJSON()` emits JSON;
> `-o` only accepts `table` / `json` (**no csv**; invalid values error out).

**Table output rules:**
- Header is bold (`fatih/color`, degraded to plain text with `NO_COLOR` / `--no-color`)
- Time fields are rendered in a stable format (some relative)
- Long strings are truncated with `…` (`utils.Truncate`)
- `text/tabwriter` alignment; piped output degrades to plain text

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

> The draft said Makefile + `.goreleaser.toml`; **neither exists**. The release chain is shell
> scripts plus two CI backends (GitHub Actions / Gitee Go) sharing `scripts/ci.sh`.

### 5.1 Build and local development

```bash
# Install into $GOPATH/bin (normal development)
go install ./cmd/gitee

# One-off local build (not on PATH)
go build -o gitee ./cmd/gitee

# Exact CI gate
bash scripts/ci.sh     # gofmt / vet / golangci-lint / build / test -race / cross-build / govulncheck

# Manual cross-compilation
GOOS=linux   GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee
GOOS=windows GOARCH=amd64 go build -o dist/gitee-windows-amd64.exe ./cmd/gitee
```

### 5.2 Release chain

```bash
git tag -a v0.13.2 -m v0.13.2 && git push origin v0.13.2
GITEE_TOKEN=... bash scripts/release.sh --publish
```

- `scripts/release.sh`: builds 5 platform artifacts (`gitee-<os>-<arch>.tar.gz`, Windows `.zip`),
  writes `SHA256SUMS.txt`, calls `publish-release.sh` to create the Gitee release and upload
  assets (idempotent)
- `scripts/publish-release.sh`: only creates the release and uploads assets, reusing
  `gitee release create/upload`
- Version is injected via ldflags; README badges are locked by `TestReadmeVersionMatchesVersionVar`
- No Homebrew tap, no goreleaser

---

## 6. Security Design

| Security Measure | Implementation |
|------------------|-----------------|
| Config file permissions | `~/.gitee-cli/settings.json` created with mandatory `0600` (`os.OpenFile(..., 0600)`) |
| Token not in logs | `--verbose` mode sanitizes Authorization header as `token ***` |
| Token not committed | `~/.gitee-cli/` excluded in `.gitignore_global`; tool prompts user on init |
| HTTPS | Only talks to `https://` endpoints; there is no `--insecure` switch to skip TLS verification |
| Certificate validation | System CA bundle + standard `net/http` verification |
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