# gitee-cli 開発者ガイド

バージョン：1.0.0 | 対象：貢献者 / チーム開発メンバー

---

## 1. 環境構築

### 1.1 必需ツール

```bash
# Go 1.23+（必須）
go version    # 1.23 以上が必要

# Cobra CLI（コマンドテンプレートジェネレーター）
go install github.com/spf13/cobra-cli@latest

# golangci-lint（コード品質チェック）
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin v1.59.0

# goreleaser（リリース自動化）
go install github.com/goreleaser/goreleaser@latest

# govulncheck（脆弱性スキャン）
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### 1.2 推奨ツール

```bash
# goimports（import 整理）
go install golang.org/x/tools/cmd/goimports@latest

# mockgen（Mock 生成）
go install go.uber.org/mock/mockgen@latest
```

### 1.3 クローンと初期化

```bash
git clone https://gitee.com/<your-user>/gitee-cli.git
cd gitee-cli
go mod download
go build -o bin/gitee ./cmd/gitee
./bin/gitee --version
```

---

## 2. プロジェクト初期化（新規プロジェクト用）

```bash
# Go モジュール初期化（モジュールパスは gitee.com/<org>/gitee-cli に従う）
go mod init gitee.com/gitee/gitee-cli

# 依存関係を追加
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/go-resty/resty/v2
go get golang.org/x/oauth2
go get github.com/fatih/color
go get github.com/stretchr/testify

# Cobra CLI でスケルトンを生成
cobra-cli init --viper --author "Gitee Team" --license apache
cobra-cli add auth
cobra-cli add repo
cobra-cli add issue
cobra-cli add pr
cobra-cli add gist
cobra-cli add org
cobra-cli add search
cobra-cli add api
cobra-cli add config
cobra-cli add completion
```

---

## 3. 開発フロー

### 3.1 新規サブコマンドの追加

例：`gitee issue list` の追加

**Step 1：コマンドファイル作成**

```go
// cmd/issue/list.go
package issue

import (
    "context"
    "fmt"

    "github.com/spf13/cobra"
    "gitee.com/gitee/gitee-cli/internal/api"
    "gitee.com/gitee/gitee-cli/internal/printer"
    "gitee.com/gitee/gitee-cli/pkg/types"
)

var listFlags struct {
    State    string
    Labels   string
    Assignee string
    Repo     string
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "Issue の一覧表示",
    Long:  "現在のリポジトリまたは指定リポジトリの Issue を一覧表示。状態、ラベル、担当者でフィルター可能。",
    RunE:  runList,
}

func init() {
    f := listCmd.Flags()
    f.StringVarP(&listFlags.State, "state", "s", "open", "Issue の状態: open/closed/all")
    f.StringVarP(&listFlags.Labels, "labels", "l", "", "ラベルフィルター（カンマ区切り）")
    f.StringVarP(&listFlags.Assignee, "assignee", "a", "", "担当者フィルター")
    f.StringVarP(&listFlags.Repo, "repo", "R", "", "リポジトリ（owner/name 形式）")

    IssueCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    client, err := api.NewClientFromConfig()
    if err != nil {
        return err
    }

    owner, repo, err := resolveRepo(listFlags.Repo)
    if err != nil {
        return err
    }

    var issues []types.Issue
    path := fmt.Sprintf("/repos/%s/%s/issues?state=%s", owner, repo, listFlags.State)
    if listFlags.Labels != "" {
        path += "&labels=" + listFlags.Labels
    }
    if err := client.List(ctx, path, nil, &issues); err != nil {
        return err
    }

    outputFmt, _ := cmd.Flags().GetString("output")
    p := printer.New(outputFmt, cmd.OutOrStdout())
    p.PrintHeader("#", "TITLE", "STATE", "LABELS", "ASSIGNEE", "UPDATED")
    for _, i := range issues {
        p.PrintRow(
            fmt.Sprintf("#%d", i.Number),
            truncate(i.Title, 60),
            i.State,
            joinLabels(i.Labels, 3),
            firstAssignee(i.Assignees),
            humanTime(i.UpdatedAt),
        )
    }
    return p.Flush()
}
```

**Step 2：親コマンドに登録**

```go
// cmd/issue/issue.go
var IssueCmd = &cobra.Command{
    Use:   "issue",
    Short: "Issue 管理",
}

// cmd/root.go
func init() {
    rootCmd.AddCommand(issue.IssueCmd)
}
```

**Step 3：API エンドポイント追加（必要な場合）**

```go
// internal/api/endpoints.go
const (
    EndpointIssues = "/repos/%s/%s/issues"
    EndpointIssue  = "/repos/%s/%s/issues/%d"
    EndpointIssueComments = "/repos/%s/%s/issues/%d/comments"
)
```

**Step 4：型定義の追加/更新**

```go
// pkg/types/issue.go — SDD で定義済み、必要に応じてフィールドを拡張
```

**Step 5：テスト作成（第 4 章参照）**

---

### 3.2 新規 API エンドポイント呼び出しの追加

```go
// internal/api/client.go にセマンティックメソッドを追加
func (c *client) ListIssues(ctx context.Context, owner, repo, state string) ([]types.Issue, error) {
    var issues []types.Issue
    path := fmt.Sprintf("/repos/%s/%s/issues?state=%s", owner, repo, state)
    err := c.List(ctx, path, &PageOptions{PerPage: 30}, &issues)
    return issues, err
}

func (c *client) CreateIssue(ctx context.Context, owner, repo string, body types.IssueCreateRequest) (*types.Issue, error) {
    var issue types.Issue
    path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
    err := c.Post(ctx, path, body, &issue)
    return &issue, err
}
```

---

### 3.3 リポジトリコンテキスト検出

多くのコマンドは git remote から解析された現在のリポジトリを自動検出する必要があります（`gh` と同様）：

```go
// internal/git/remote.go
func DetectRepo() (owner, repo string, err error) {
    // 1. --repo フラグをチェック
    // 2. git remote origin URL を解析
    //    - https://gitee.com/owner/repo.git
    //    - git@gitee.com:owner/repo.git
    // 3. owner, repo を返す
    remoteURL, err := exec.Command("git", "remote", "get-url", "origin").Output()
    // ... 解析ロジック
}
```

---

## 4. テスト基準

### 4.1 ユニットテスト

各 `*.go` ファイルに対応する `*_test.go` があり、`httptest` で HTTP をモックします：

```go
// internal/api/client_test.go
func TestClient_ListIssues(t *testing.T) {
    // モックサーバーを設定
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "GET", r.Method)
        assert.Contains(t, r.URL.Path, "/issues")
        assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("X-Total-Count", "2")
        json.NewEncoder(w).Encode([]types.Issue{
            {Number: 1, Title: "First Issue", State: "open"},
            {Number: 2, Title: "Second Issue", State: "closed"},
        })
    }))
    defer ts.Close()

    client := api.NewClient(
        api.WithBaseURL(ts.URL),
        api.WithToken("test-token"),
    )

    issues, err := client.ListIssues(context.Background(), "owner", "repo", "all")
    require.NoError(t, err)
    require.Len(t, issues, 2)
    assert.Equal(t, 1, issues[0].Number)
    assert.Equal(t, "First Issue", issues[0].Title)
}
```

### 4.2 コマンドラインテスト（出力検証）

```go
// cmd/issue/list_test.go
func TestIssueListCommand(t *testing.T) {
    // モック API を設定
    ts := setupMockServer(t, mockIssuesResponse)
    defer ts.Close()

    // stdout をキャプチャ
    buf := &bytes.Buffer{}
    cmd := newTestRootCmd(ts.URL)
    cmd.SetOut(buf)
    cmd.SetArgs([]string{"issue", "list", "--output", "json"})

    err := cmd.Execute()
    require.NoError(t, err)

    var result []map[string]interface{}
    require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
    assert.Len(t, result, 2)
}
```

### 4.3 テスト実行

```bash
go test ./...                    # 全テストを実行
go test ./cmd/... -v             # コマンド層の詳細出力
go test ./internal/api/... -v   # API 層の詳細出力
go test ./... -cover             # カバレッジ
go test ./... -race              # レース検出（CI 必須）
go test ./... -count=1           # キャッシュ無効化（CI 使用）
```

### 4.4 カバレッジ要件

| モジュール | 最低カバレッジ |
|-----------|--------------|
| `internal/api/` | 80% |
| `internal/auth/` | 70% |
| `internal/printer/` | 90% |
| `pkg/types/` | 強制なし |
| `cmd/` | 60% |

---

## 5. コード基準

### 5.1 Lint 設定（`.golangci.yml`）

```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - gosec
    - revive
    - exhaustive

linters-settings:
  goimports:
    local-prefixes: gitee.com/gitee/gitee-cli
```

実行：

```bash
golangci-lint run ./...
golangci-lint run --fix ./...   # 自動修正
```

### 5.2 整形

```bash
go fmt ./...        # コードを整形
goimports -w .      # import を整理（グループ化：stdlib / 外部 / 内部）
```

### 5.3 コミット基準（Conventional Commits）

形式：`<type>(<scope>): <description>`

| タイプ | 適用場面 |
|--------|---------|
| `feat` | 新機能追加 |
| `fix` | バグ修正 |
| `docs` | ドキュメント変更 |
| `test` | テスト追加/変更 |
| `refactor` | リファクタリング（機能変更なし） |
| `chore` | 依存関係更新、ビルドスクリプトなど |
| `perf` | パフォーマンス最適化 |
| `ci` | CI/CD 設定変更 |

例：

```
feat(repo): add repo fork command
fix(auth): handle expired token gracefully
docs(readme): add installation guide for macOS
test(api): add client pagination test coverage
refactor(printer): extract interface for output formats
chore(deps): bump resty to v2.12.0
```

### 5.4 コードレビューセルフチェックリスト

```
提出前：
[ ] go fmt && goimports を実行済み
[ ] golangci-lint で警告なし
[ ] すべての API 呼び出しにエラー処理あり（error を無視しない）
[ ] ネットワークリクエストに context.Context を渡す
[ ] Token がログ、stdout、または任意の可視出力に表示されない
[ ] 新機能に対応するユニットテストあり
[ ] テストは happy path + error path をカバー
[ ] --help 出力の説明が正確
[ ] 新フラグにデフォルト値の説明あり
```

---

## 6. デバッグtips

### 6.1 詳細ログを有効にする

```bash
gitee --verbose repo list
# 出力：
# → GET https://gitee.com/api/v5/user/repos?page=1&per_page=30
# ← HTTP 200 (245ms) [3 items]
```

### 6.2 `gitee api` コマンドでエンドポイントを直接テスト

```bash
# 現在のユーザー情報を取得
gitee api GET /user

# リポジトリを一覧表示（ページネーション付き）
gitee api GET /user/repos?type=owner&per_page=5

# テストリポジトリを作成
gitee api POST /user/repos --field name=test-cli-repo --field private=true --field auto_init=true

# テストリポジトリを削除
gitee api DELETE /repos/owner/test-cli-repo
```

### 6.3 API アドレスをオーバーライド（プライベートデプロイメントをテスト）

```bash
gitee --api-url "https://your-enterprise.gitee.com/api/v5" repo list
# または
GITEE_API_URL="https://your-enterprise.gitee.com/api/v5" gitee repo list
```

### 6.4 Token を直接使用（OAuth をスキップ）

```bash
gitee auth login --with-token
# stdin から token を読み取り：
echo "your_access_token" | gitee auth login --with-token
```

---

## 7. ビルドとリリース

### 7.1 ローカルビルド

```bash
make build          # bin/gitee にコンパイル
make install        # $GOPATH/bin にインストール
```

### 7.2 クロスコンパイル

```bash
make cross-build
# 出力：
# dist/gitee-linux-amd64
# dist/gitee-linux-arm64
# dist/gitee-darwin-amd64
# dist/gitee-darwin-arm64
# dist/gitee-windows-amd64.exe
```

### 7.3 リリース

```bash
# 先にタグを打つ
git tag v0.1.0
git push origin v0.1.0

# goreleaser が自動的にビルド + アップロード
make release
```

---

## 8. CI/CD パイプライン

### 8.1 GitHub Actions / Gitee Go CI の例

```yaml
# .gitee/workflows/ci.yml
name: CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go mod download
      - run: go vet ./...
      - run: golangci-lint run ./...
      - run: go test ./... -v -race -count=1
      - run: govulncheck ./...

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: make cross-build
```

---

## 9. プロジェクトマイルストーン

| フェーズ | ターゲットコマンド | 説明 |
|---------|------------------|------|
| **M0** | `gitee auth` 一式 | OAuth2 Device Flow、Token ストレージ、`auth status/logout/token` |
| **M1** | `gitee repo` 一式 + `gitee api` | リポジトリ CRUD、生 API アクセス、設定モジュール |
| **M2** | `gitee issue` 一式 | Issue CRUD、コメント、ラベル管理 |
| **M3** | `gitee pr` 一式 | PR CRUD、マージ、レビュー、diff/checks |
| **M4** | `gitee gist` + `gitee search` | Gist 管理、3 種類の検索 |
| **M5** | `gitee org` + シェル補完 + i18n | 組織管理、Bash/Zsh/Fish 補完、zh/en 切り替え |
| **M6** | エンタープライズデプロイ + マルチアカウント + プラグインシステム | `--hostname` 拡張、credentials マルチホスト、外部コマンド検出 |

---

## 10. 参考資料

| リソース | リンク |
|---------|-------|
| Gitee API v5 Swagger | https://gitee.com/api/v5/swagger |
| Gitee OAuth2 ドキュメント | https://gitee.com/api/v5/oauth_doc |
| Cobra 公式ドキュメント | https://pkg.go.dev/github.com/spf13/cobra |
| Viper 公式ドキュメント | https://pkg.go.dev/github.com/spf13/viper |
| Resty HTTP クライアント | https://github.com/go-resty/resty |
| glab プロジェクト（最良の参照実装） | https://github.com/gitlab-org/cli |
| gh CLI プロジェクト | https://github.com/cli/cli |
| Go プロジェクトレイアウト標準 | https://github.com/golang-standards/project-layout |
| Conventional Commits | https://www.conventionalcommits.org |
| golangci-lint ルールドキュメント | https://golangci-lint.run/usage/linters/ |