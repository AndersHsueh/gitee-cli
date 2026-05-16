# gitee-cli システム設計書（System Design Document）

バージョン：1.0.0 | 技術スタック：Go 1.23+ / Cobra / Viper / Resty

---

## 1. 技術選択

| コンポーネント | 技術選択 | バージョン | 選択理由 |
|-------------|---------|-----------|---------|
| 言語 | Go | 1.23+ | クロスコンパイルが簡単、静的バイナリ、成熟したエコシステム |
| CLI フレームワーク | Cobra | v1.8+ | gh/glab と同じ、ヘルプ/補完/フラグ生成が優秀 |
| 設定管理 | Viper | v1.18+ | YAML/ENV/フラグ三層設定、Cobra との完璧な統合 |
| HTTP クライアント | resty | v2.x | チェーン可能 API、自动リトライ、リクエスト/レスポンスミドルウェア |
| OAuth2 | `golang.org/x/oauth2` | latest | 標準実装、Device Flow をサポート |
| テーブル出力 | `tabwriter` (stdlib) | — | 追加依存なし、軽量 |
| カラー | `github.com/fatih/color` | v1.x | クロスプラットフォーム ANSI カラー、`NO_COLOR` をサポート |
| テスト | `testing` + `testify` | v1.x | 標準 + アサーションライブラリ |
| HTTP モック | `httptest` (stdlib) | — | 組み込みサーバー、ゼロ依存 |

---

## 2. プロジェクトディレクトリ構造

```
gitee-cli/
├── cmd/                          # Cobra コマンド定義（サブコマンドごとに 1 ファイル）
│   ├── root.go                   # ルートコマンド、グローバルフラグ定義
│   ├── version.go                # --version 実装
│   ├── auth/
│   │   ├── auth.go              # auth 親コマンド
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
│   │   └── raw.go               # 生 API アクセス
│   ├── config/
│   │   ├── config.go
│   │   ├── list.go
│   │   ├── set.go
│   │   ├── get.go
│   │   └── unset.go
│   └── completion/
│       └── shell.go
├── internal/                     # 内部パッケージ（外部にエクスポートしない）
│   ├── api/
│   │   ├── client.go            # HTTP クライアントラッパー（resty + ミドルウェア）
│   │   ├── endpoints.go         # API エンドポイントパス定数
│   │   ├── errors.go            # API エラータイプ + HTTP ステータスコードマッピング
│   │   └── pagination.go        # ページネーション Link Header 解析
│   ├── auth/
│   │   ├── token.go             # 認証情報読み書き（settings.json）
│   │   ├── oauth.go             # OAuth2 Device Flow 実装
│   │   └── config.go            # Token ストレージ設定
│   ├── config/
│   │   ├── config.go            # Viper 設定ラッパー
│   │   └── migrate.go           # 設定移行（バージョン互換性）
│   ├── printer/
│   │   ├── printer.go           # Printer インターフェース定義
│   │   ├── table.go             # tabwriter テーブル出力
│   │   ├── json.go              # JSON 整形出力
│   │   └── csv.go               # CSV 出力
│   ├── terminal/
│   │   ├── prompt.go            # インタラクティブプロンプト（確認/質問）
│   │   ├── spinner.go           # ローディングアニメーション
│   │   └── browser.go           # ブラウザ開く（クロスプラットフォーム）
│   └── i18n/
│       ├── i18n.go              # 翻訳関数 T()
│       ├── zh_CN.go             # 中国語言語パック
│       └── en_US.go             # 英語言語パック
├── pkg/                         # 再利用可能公開パッケージ
│   └── types/
│       ├── repo.go              # Repository タイプ
│       ├── issue.go             # Issue / Label / Milestone タイプ
│       ├── pr.go                # PullRequest / Review タイプ
│       ├── gist.go              # Gist / GistFile タイプ
│       ├── org.go               # Organization / Member タイプ
│       ├── user.go              # User / Email タイプ
│       └── search.go            # SearchResult タイプ
├── cmd/gitee/
│   └── main.go                  # プログラムエントリ
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.toml
└── README.md
```

---

## 3. コアモジュール設計

### 3.1 API Client（`internal/api/client.go`）

```
┌─────────────────────────────────────────┐
│            Cobra Commands               │
│   (repo list, issue create, pr merge)   │
└──────────────────┬──────────────────────┘
                   │ 呼び出し
┌──────────────────▼──────────────────────┐
│              API Client                  │
│  • Authorization ヘッダー自動注入         │
│  • リクエスト/レスポンスログ（--verbose） │
│  • 5xx 自動リトライ（3 回、指数バックオフ）│
│  • context.Context タイムアウト制御       │
│  • 統一エラータイプマッピング             │
└──────────────────┬──────────────────────┘
                   │ HTTPS
┌──────────────────▼──────────────────────┐
│          Gitee REST API v5               │
│     https://gitee.com/api/v5/           │
└─────────────────────────────────────────┘
```

**インターフェース定義：**

```go
// internal/api/client.go
type Client interface {
    // 汎用リクエスト（生 Response を返す）
    Do(ctx context.Context, method, path string, body interface{}) (*Response, error)
    // セマンティックショートカットメソッド
    Get(ctx context.Context, path string, result interface{}) error
    Post(ctx context.Context, path string, body, result interface{}) error
    Put(ctx context.Context, path string, body, result interface{}) error
    Patch(ctx context.Context, path string, body, result interface{}) error
    Delete(ctx context.Context, path string) error
    // 自動ページネーションリスト（全ページを走査、result スライスに追加）
    List(ctx context.Context, path string, opts *PageOptions, result interface{}) error
}

type Response struct {
    StatusCode int
    Headers    http.Header
    Body       []byte
}

type PageOptions struct {
    Page    int // 開始ページ（1 から）
    PerPage int // 1 ページあたりの件数（デフォルト 30、最大 100）
    Limit   int // 最大合計件数（0=無制限）
}
```

**主要な実装詳細：**

```go
// Authorization ヘッダーを自動注入
client.SetHeader("Authorization", "token "+accessToken)

// リトライ設定（resty）
client.SetRetryCount(3).
    SetRetryWaitTime(2 * time.Second).
    SetRetryMaxWaitTime(8 * time.Second).
    AddRetryCondition(func(r *resty.Response, err error) bool {
        return r.StatusCode() >= 500 || err != nil
    })

// タイムアウト
client.SetTimeout(time.Duration(timeout) * time.Second)
```

---

### 3.2 認証モジュール（`internal/auth/`）

**認証情報保存場所：**

```
~/.gitee-cli/
└── settings.json      # すべてのユーザー設定と認証情報を統合保存（ファイル権限 0600）
```

> 旧パス `~/.config/gitee/` は廃止、ツール起動時に自動移行。

**`~/.gitee-cli/settings.json` 完全構造：**

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

> ⚠️ **セキュリティ注意**
> - ファイル作成時に必須権限 `0600` を強制（所有者の読み書きのみ）
> - `token` フィールドは Gitee Access Token、**絶対にバージョン管理にコミットしない**
> - `~/.gitee-cli/` を `~/.gitignore_global` に追加
> - CI/CD で使用する場合は、環境変数 `GITEE_TOKEN` で注入、ファイルには書き込まない

**マルチホストサポート（エンタープライズプライベートデプロイ）：**

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

**`git` フィールドの説明：**

`settings.json` の `git.user` はシステムの `git config --global` と同期を維持し、コミット作成時に自動的に署名情報を入力するために使用されます。ツールの初回実行時にグローバル git 設定を読み取ってこのフィールドに書き込みます。`gitee config` コマンドで個別に変更することもできます。

**OAuth2 Device Flow シーケンス：**

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
      |  user_code をユーザーに表示  |
      |  ブラウザを開く            |                        |
      |-------------------------------------------------------->|
      |                          |    ユーザーが ABCD-1234 を入力   |
      |                          |<------------------------|
      |                          |    ユーザーが授權をクリック   |
      |                          |<------------------------|
      |                          |                        |
      |  [5 秒ごとにポーリング]     |                        |
      |  POST /oauth/device/token |                        |
      |  {device_code, grant_type}                        |
      |------------------------->|                        |
      |  {access_token,          |                        |
      |   token_type: "bearer"   |                        |
      |   scope, refresh_token}  |                        |
      |<-------------------------|                        |
      |                          |                        |
      |  settings.json に書き込み  |                        |
      |  GET /user → ユーザー名を検証 |                        |
```

---

### 3.3 設定モジュール（`internal/config/`）

**三層設定優先度（高い順）：**

```
1. CLI フラグ（--token, --output, --api-url ...）
2. 環境変数（GITEE_TOKEN, GITEE_OUTPUT ...）
3. 設定ファイル（~/.gitee-cli/settings.json）
4. 組み込みデフォルト値
```

**設定初期化（`internal/config/`）：**

```go
func Init() {
    // settings.json パス：~/.gitee-cli/settings.json
    configDir := filepath.Join(os.Getenv("HOME"), ".gitee-cli")
    os.MkdirAll(configDir, 0700)

    viper.SetConfigName("settings")
    viper.SetConfigType("json")
    viper.AddConfigPath(configDir)
    viper.SetEnvPrefix("GITEE")
    viper.AutomaticEnv()

    // デフォルト値
    viper.SetDefault("preferences.output", "table")
    viper.SetDefault("preferences.prompt", "enabled")
    viper.SetDefault("hosts.gitee\\.com.protocol", "https")
    viper.SetDefault("api_url", "https://gitee.com/api/v5")
    viper.SetDefault("timeout", 30)
}

// 現在のホストの token を読み取る
func GetToken(hostname string) string {
    key := fmt.Sprintf("hosts.%s.token", hostname)
    return viper.GetString(key)
}

// git グローバルユーザー情報を読み取る（初回初期化時に git config から同期）
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

### 3.4 出力モジュール（`internal/printer/`）

**Printer インターフェース：**

```go
type Printer interface {
    PrintHeader(fields ...string)
    PrintRow(values ...string)
    PrintFooter(msg string)
    Flush() error
}

// --output フラグに基づくファクトリメソッド
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

**テーブル出力ルール：**
- 表頭：`color.New(color.Bold).Sprint(field)`
- 時間フィールド：`humanize.Time(t)` で相対時間を出力
- 長い文字列：60 文字で切り捨てて `…` を追加
- ターミナル幅を自動検出（`term.GetSize()`）

---

### 3.5 ページネーション（`internal/api/pagination.go`）

Gitee API v5 はレスポンスヘッダーにページネーション情報を返します（RFC 5988 Link Header）：

```
Link: <https://gitee.com/api/v5/user/repos?page=2>; rel="next",
      <https://gitee.com/api/v5/user/repos?page=10>; rel="last"
X-Total-Count: 287
```

**ページネーション解析ロジック：**

```go
type PaginationInfo struct {
    NextPage  int
    LastPage  int
    TotalCount int
}

func ParseLinkHeader(header string) *PaginationInfo {
    // Link ヘッダーを解析、next/last ページ番号を抽出
    // X-Total-Count を解析
}

// 全ページを自動走査
func (c *client) List(ctx context.Context, path string, opts *PageOptions, result interface{}) error {
    page := opts.Page
    for {
        url := fmt.Sprintf("%s?page=%d&per_page=%d", path, page, opts.PerPage)
        resp, err := c.Get(ctx, url, &pageResult)
        if err != nil {
            return err
        }
        // 結果を result に追加
        // Link ヘッダーをチェック
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

### 3.6 エラー処理（`internal/api/errors.go`）

**エラタイプ階層：**

```go
// API エラー（HTTP 4xx/5xx）
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

// HTTP ステータスコード → ユーザー向けエラーメッセージマッピング
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

**コマンド層統一エラー処理：**

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

## 4. データモデル（`pkg/types/`）

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

## 5. ビルドとリリース

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

### 5.2 `.goreleaser.toml` コア設定

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

## 6. セキュリティ設計

| セキュリティ対策 | 実装方式 |
|----------------|---------|
| 設定ファイル権限 | `~/.gitee-cli/settings.json` は作成時に必須の `0600` を設定（`os.OpenFile(..., 0600)`） |
| Token がログに記録されない | `--verbose` モードでは Authorization ヘッダーを `token ***` としてサニタイズ出力 |
| Token がコミットされない | `.gitignore_global` で `~/.gitee-cli/` を除外；ツール初期化時にユーザーにプロンプト |
| HTTPS を強制 | デフォルトで有効、`--insecure` は明示的に設定する必要がある |
| 証明書検証 | システムの CA バンドルを使用、`--insecure` でスキップ |
| 入力検証 | リポジトリ名、ユーザー名は正規表現 `^[a-zA-Z0-9_.-]+$` で検証 |
| CI/CD セキュリティ | Token は環境変数 `GITEE_TOKEN` で注入、ファイルには依存しない |
| 依存関係セキュリティ | CI で `govulncheck ./...` を実行 |
| SSRF 防护 | `--api-url` は `https://` 接頭辞のみ許可 |

---

## 7. 拡張性設計

### 7.1 プラグインシステム（v2.0 プラン）

```go
// コマンド登録時に PATH の gitee-* 実行ファイルをチェック（gh extensions と同様）
func RegisterExternalCommands(root *cobra.Command) {
    for _, ext := range discoverExtensions() {
        root.AddCommand(wrapExtensionCommand(ext))
    }
}
```

### 7.2 エンタープライズプライベートデプロイサポート

`--hostname` フラグまたは `GITEE_HOSTNAME` 環境変数で API ベースを切り替え：

```go
apiURL := fmt.Sprintf("https://%s/api/v5", hostname)
```

### 7.3 マルチアカウントサポート（v2.0 プラン）

`settings.json` の `hosts` フィールドは複数のホスト名をネイティブサポート：

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