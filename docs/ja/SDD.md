# gitee-cli システム設計書（System Design Document）

> ⚠️ **ステータス通知（v0.13.2、2026-10 メンテナンス）**：本文書は初期の設計ドラフトであり、
> **未実装または古くなった**記述を含みます（Viper 設定、OAuth2 デバイスフロー、
> `internal/auth` / `printer` / `i18n` モジュール、goreleaser リリースなど）。現在の実装と
> 挙動はコード、`gitee --help`、`gitee <cmd> --agent-help`、`gitee help reference` が正です。
> 現状と大きく乖離する章は本文中で修正しています。


バージョン：1.0.0（設計ドラフト。現状は冒頭通知を参照）| 技術スタック：Go 1.27+ / Cobra / Resty / 手書き JSON 設定

---

## 1. 技術選択

| コンポーネント | 技術選択 | バージョン | 選択理由 |
|-------------|---------|-----------|---------|
| 言語 | Go | 1.27+（go.mod は `go 1.27.3`） | クロスコンパイルが簡単、静的バイナリ、成熟したエコシステム |
| CLI フレームワーク | Cobra | v1.10 | gh/glab と同じ、ヘルプ/補完/フラグ生成が優秀 |
| 設定管理 | 手書き JSON（`encoding/json`） | stdlib | 構造が単純で Viper 不要；アトミック書き込み + 0600 |
| HTTP クライアント | resty | v2.17 | チェーン可能 API、自動リトライ、リクエスト/レスポンスミドルウェア |
| 認証 | パーソナルアクセストークン（PAT） | — | ヘッダー `Authorization: token <PAT>`；**OAuth2 / x/oauth2 依存なし** |
| テーブル出力 | `tabwriter` (stdlib) | — | `internal/output.NewTableWriter` に統一 |
| カラー | `github.com/fatih/color` | v1.19 | クロスプラットフォーム ANSI カラー、`NO_COLOR` をサポート |
| テスト | `testing` + `testify` | v1.12 | 標準 + アサーションライブラリ |
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
├── internal/                     # 内部パッケージ（非公開）
│   ├── api/                     # HTTP クライアント（resty）：client / endpoints / errors / pagination
│   ├── alias/                   # ユーザー別名の解決と保存
│   ├── atomicfile/              # アトミックなファイル書き込み（temp + chmod + rename）
│   ├── cmdkit/                  # リストコマンド骨格（RunList）と共通オプション
│   ├── config/                  # 手書き JSON 設定（settings.json、0600 / 0700）
│   ├── domain/                  # コマンド横断の業務ロジック（label / milestone / notification / webhook）
│   ├── extension/               # 動的拡張の読み込みと保存
│   ├── output/                  # 統一 tabwriter（NewTableWriter）と PrintJSON
│   ├── terminal/                # ターミナル対話（prompt）
│   └── utils/                   # 共通ユーティリティ（repo 解決、flag、git、検証）
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
├── scripts/                      # ci.sh / release.sh / publish-release.sh
└── README.md
```

---

### 2.1 実際のディレクトリ構造（v0.13.x）

上記の設計稿のツリーは**現在のコードと差異があります**。以下が実測構造です
（`git ls-files` / `ls` で再現可能）：

```
cmd/            # グループごとに 1 パッケージ：agent alias api auth completion config extension
                # gist issue label milestone notification org pr release repo search ssh-key
                # status user webhook help_topics + root.go / runtime.go / cmdkit ヘルパー
internal/       # api alias atomicfile cmdkit config domain extension output terminal utils
                # （auth/、printer/、i18n/ は存在しない。認証は cmd/auth + internal/config）
pkg/types/      # API データモデル
docs/           # 本設計書 + philosophy.md + 多言語 README
scripts/        # ci.sh / release.sh / publish-release.sh（Makefile / goreleaser はない）
test/           # 機能テスト bash スクリプト + ドキュメント検証スクリプト
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

### 3.2 認証モジュール（`cmd/auth/` + `internal/config/`）

> 設計稿は `internal/auth/` + OAuth2 Device Flow としていましたが、**そのパッケージは存在せず
> OAuth2 も使いません**。認証 = パーソナルアクセストークン（PAT）：解決（`--token` >
> `GITEE_TOKEN` > `settings.json`）→ `GET /user` で検証 → 必要なら `settings.json` へ保存。

**資格情報の保存先：**

```
~/.gitee-cli/
└── settings.json      # 設定と資格情報を一元管理（ファイル 0600、ディレクトリ 0700）
```

**`~/.gitee-cli/settings.json` の実際の構造（`internal/config/config.go` 準拠）：**

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

> 注：`expires_at` は `HostConfig` に存在せず、ファイルに書いても無視されます。

> ⚠️ **セキュリティ注意**
> - 作成時は `0600`。読み込み時に権限が広すぎる場合は自動で絞り込み警告を出します
> - `token` は Gitee のアクセストークンであり、**絶対にコミットしない**こと
> - CI/CD では `GITEE_TOKEN` を環境変数で注入し、ファイルに書かない
> - `--token` は `ps` とシェル履歴に残るため、使い捨て以外は `GITEE_TOKEN` を推奨

**認証シーケンス（実際の実装）：**

```
Client (gitee-cli)                 Gitee API v5
      |                                  |
      | 1. トークンの優先順位を解決      |
      |    --token > GITEE_TOKEN > settings.json
      |                                  |
      | 2. GET /user                     |
      |    Authorization: token <PAT>    |
      |--------------------------------->|
      |    200 {login, id, ...}          |
      |<---------------------------------|
      |                                  |
      | 3. --token 明示時のみ            |
      |    settings.json ← token (0600)  |
      |    ディレクトリ 0700 / ファイル 0600
```

`gitee auth refresh` はスタブ実装で、「Token refresh is not yet supported. Please
re-authenticate with 'gitee auth login'.」と表示して終了コード 0 になります。更新も再検証もせず、
OAuth の `refresh_token` 交換でもありません。

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

### 3.4 出力モジュール（`internal/output/`）

> 設計稿は `internal/printer/` + `Printer` インターフェースでしたが、**そのパッケージは
> 存在しません**。現状：`internal/output.NewTableWriter()` が tabwriter を生成し、
> `output.PrintJSON()` が JSON を出力。`-o` は `table` / `json` のみ
> （**csv なし**、不正値はエラー終了）。

**テーブル出力の規則：**
- ヘッダーは太字（`fatih/color`、`NO_COLOR` / `--no-color` でプレーンテキストへ）
- 時刻フィールドは安定した形式で表示（一部は相対時刻）
- 長い文字列は `…` で切り詰め（`utils.Truncate`）
- `text/tabwriter` で整列、パイプ出力時はプレーンテキストへ縮退

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

> 設計稿は Makefile + `.goreleaser.toml` でしたが、**どちらも存在しません**。リリースは
> shell スクリプト + 2 つの CI バックエンド（GitHub Actions / Gitee Go）が `scripts/ci.sh` を
> 共有する構成です。

### 5.1 ビルドとローカル開発

```bash
# $GOPATH/bin へインストール（日常開発はこちら）
go install ./cmd/gitee

# 一度きりのローカルビルド（PATH には入らない）
go build -o gitee ./cmd/gitee

# CI と完全に同じゲート
bash scripts/ci.sh     # gofmt / vet / golangci-lint / build / test -race / クロスビルド / govulncheck

# 手動クロスコンパイル
GOOS=linux   GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee
GOOS=windows GOARCH=amd64 go build -o dist/gitee-windows-amd64.exe ./cmd/gitee
```

### 5.2 リリースチェーン

```bash
git tag -a v0.13.2 -m v0.13.2 && git push origin v0.13.2
GITEE_TOKEN=... bash scripts/release.sh --publish
```

- `scripts/release.sh`：5 プラットフォームの成果物（`gitee-<os>-<arch>.tar.gz`、Windows は
  `.zip`）をビルドし `SHA256SUMS.txt` を生成、`publish-release.sh` で Gitee release を作成し
  アセットをアップロード（冪等）
- `scripts/publish-release.sh`：「release 作成 + アセットアップロード」のみを担当し、
  `gitee release create/upload` を再利用
- バージョンは ldflags で注入、README バッジは `TestReadmeVersionMatchesVersionVar` が固定
- Homebrew tap なし、goreleaser なし

---

## 6. セキュリティ設計

| セキュリティ対策 | 実装方式 |
|----------------|---------|
| 設定ファイル権限 | `~/.gitee-cli/settings.json` は作成時に必須の `0600` を設定（`os.OpenFile(..., 0600)`） |
| Token がログに記録されない | `--verbose` モードでは Authorization ヘッダーを `token ***` としてサニタイズ出力 |
| Token がコミットされない | `.gitignore_global` で `~/.gitee-cli/` を除外；ツール初期化時にユーザーにプロンプト |
| HTTPS | `https://` のエンドポイントのみ使用。TLS 検証を飛ばす `--insecure` は存在しない |
| 証明書検証 | システムの CA バンドル + 標準 `net/http` 検証 |
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