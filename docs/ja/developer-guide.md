# gitee-cli 開発者ガイド

> ⚠️ **ステータス通知（v0.13.2、2026-10 メンテナンス）**：本文書は初期の設計ドラフトであり、
> 一部の例は**現在のコードと一致しません**（モジュールパス、`internal/printer`、
> Viper/oauth2、Makefile/goreleaser など）。事実はコード、`gitee --help`、
> `gitee <cmd> --agent-help`、`bash scripts/ci.sh` が正です。乖離の大きい節は本文中で修正済み。


バージョン：1.0.0（設計ドラフト。現状は冒頭通知を参照）| 対象：貢献者 / チーム開発メンバー

---

## 1. 環境構築

### 1.1 必需ツール

```bash
# Go 1.27+（必須。go.mod は `go 1.27.3`）
go version    # 1.27 以上が必要

# golangci-lint（コード品質チェック。CI は v2.15.0、設定は v2 schema）
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

# govulncheck（脆弱性スキャン。go.mod の tool 依存でバージョン固定）
go install golang.org/x/vuln/cmd/govulncheck@latest

# リリースと CI はリポジトリ内スクリプト。Makefile / goreleaser は不要
bash scripts/ci.sh --fast   # ローカル品質ゲート
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
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go mod download
go install ./cmd/gitee      # $GOPATH/bin にインストール（日常開発はこちら）
gitee version
```

---

## 2. リポジトリの初期化（現状）

本リポジトリは**既に作成済み**で、`go mod init` / `cobra-cli` は不要です。要点：

- モジュールパスは `gitee-cli`（`gitee.com/<org>/gitee-cli` ではない）なので、内部 import は
  `gitee-cli/internal/api`、`gitee-cli/cmd` と書きます。
- 依存は Cobra、resty v2、fatih/color、testify（+ 間接依存）のみ。
  **Viper も golang.org/x/oauth2 もありません**。
- 設定は手書き JSON（`internal/config`）で Viper 不使用、認証は PAT で OAuth ではありません。

```bash
# go.mod に従って依存とツールを取得
go mod download
go build ./...              # コンパイル自己チェック
bash scripts/ci.sh --fast   # gofmt + vet + build + test（ローカル高速ゲート）
```

---

## 3. 開発フロー

### 3.1 新規サブコマンドの追加

例：`gitee issue list` の追加

**Step 1：コマンドファイル作成**

実際の `cmd/issue/list.go` を参照（ここでは簡略化）：

```go
// cmd/issue/list.go
package issue

import (
    "net/url"

    "gitee-cli/cmd"
    "gitee-cli/internal/api"
    "gitee-cli/internal/cmdkit"
    "gitee-cli/internal/utils"
    "gitee-cli/pkg/types"

    "github.com/spf13/cobra"
)

var listFlags struct {
    state    string
    labels   string
    assignee string
}

var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List issues in a repository",
    Example: `  gitee issue list --repo owner/repo
  gitee issue list --repo owner/repo --state closed --labels bug`,
    PreRunE: func(c *cobra.Command, args []string) error {
        if cmd.RuntimeFromCmd(c).AgentHelp {
            cmd.PrintAgentHelp(issueListAgentHelp)   // 任意：このコマンドの agent-help 文言
        }
        return nil
    },
    RunE: runList,
}

func init() {
    f := listCmd.Flags()
    f.StringVar(&listFlags.state, "state", "open", "Filter by state: open|closed|all")
    f.StringVar(&listFlags.labels, "labels", "", "Filter by labels (comma-separated)")
    f.StringVar(&listFlags.assignee, "assignee", "", "Filter by assignee username")
    // --limit / --per-page / --paginate は root の永続フラグ。ここで再登録しない。
    IssueCmd.AddCommand(listCmd)
}

func runList(cobraCmd *cobra.Command, args []string) error {
    rt := cmd.RuntimeFromCmd(cobraCmd)
    client, err := cmd.NewAPIClientFrom(rt)
    if err != nil {
        return err
    }

    // list コマンドは cmdkit.RunList 骨格に統一：ページネーション、テーブル、JSON、
    // 空結果メッセージを一箇所で処理。
    opts := cmd.NewCmdkitOptionsFrom(rt, client)
    return cmdkit.RunList[types.Issue](opts, cmdkit.ListSpec[types.Issue]{
        ResolveOwnerRepo: func() (string, string, error) {
            return utils.ResolveRepo(rt.Repo, "", nil)
        },
        Path: func(owner, repoName string) string {
            qv := url.Values{}
            qv.Set("state", listFlags.state)
            if listFlags.labels != "" {
                qv.Set("labels", listFlags.labels)
            }
            if listFlags.assignee != "" {
                qv.Set("assignee", listFlags.assignee)
            }
            return api.IssuesList(owner, repoName) + "?" + qv.Encode()
        },
        Columns: []string{"#", "TITLE", "STATE", "LABELS", "UPDATED"},
        Row: func(i types.Issue) []string {
            return []string{i.Number, utils.Truncate(i.Title, 60), i.State, "-", i.UpdatedAt.Format("2006-01-02")}
        },
        EmptyText: func(o, r string) string { return "No issues found in " + o + "/" + r },
        ErrCtx:    "listing issues",
    })
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

エンドポイントは `internal/api/endpoints.go` の builder 関数だけで組み立てます。
**cmd 層で `fmt.Sprintf` による URL 結合は禁止**：

```go
// internal/api/endpoints.go
func IssuesList(owner, repo string) string {
    return "/repos/" + owner + "/" + repo + "/issues"
}

func IssueGet(owner, repo string, number int) string {
    return IssuesList(owner, repo) + "/" + strconv.Itoa(number)
}
```

**Step 4：型定義の追加/更新**

```go
// pkg/types/issue.go — SDD で定義済み、必要に応じてフィールドを拡張
```

**Step 5：テスト作成（第 4 章参照）**

---

### 3.2 新規 API エンドポイント呼び出しの追加

生成済みの Client メソッド（`Get/Post/Patch/PatchForm/Put/Delete/DeleteWithBody/UploadFile`）を
再利用し、`internal/api` に業務メソッドを増やさない方針です：

```go
// 読み取り：result のポインタを渡す
var issue types.Issue
err := client.Get(ctx, api.IssueGet(owner, repo, number), &issue)

// 書き込み：JSON body
err := client.Post(ctx, api.IssuesCreate(owner, repo), req, &created)

// 書き込み：Gitee v5 の一部 write パスは formData のみ（issue close/reopen/edit）
err := client.PatchForm(ctx, api.IssuesEdit(owner, number), url.Values{
    "repo":  {repo},
    "state": {"closed"},
})

// 一覧：FetchListOrAll / cmdkit.RunList を優先し、paginate 分岐を手書きしない
issues, err := api.FetchListOrAll[types.Issue](ctx, client, path, paginate, perPage, limit, page, "listing issues")
```

---

### 3.3 リポジトリコンテキスト検出

リポジトリ解決は `internal/utils` に統一し、コマンド内で git remote を解析しない：

```go
// utils.ParseOwnerRepo("owner/repo") → owner, repo
// utils.ResolveRepo(owner, repo, args) → --repo / 位置引数 / git remote の順で解決
owner, repoName, err := utils.ResolveRepo(rt.Repo, "", args)
```

- `--repo` / `-R`（一部コマンド）が指定されれば最優先；
- 次に位置引数（例：`gitee repo view owner/repo`）；
- 最後にカレントディレクトリの git remote（`git remote get-url origin`）。

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

    client := api.NewClient("test-token", ts.URL, false, 30)  // token, baseURL, verbose, timeout

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

| モジュール | 推奨カバレッジ |
|-----------|--------------|
| `internal/api/` | 高（HTTP クライアントとエラーマッピング） |
| `internal/config/` | 高（権限、アトミック書き込み、優先順位） |
| `internal/domain/` | 中（現在の弱点。新しい write 経路は httptest を添える） |
| `pkg/types/` | 強制なし |
| `cmd/` | 中（agent-help 契約テストが文言を担保） |

> 現状：リポジトリ全体で約 30%（歴史的に低い）。CI はカバレッジのハードゲートを持たないため、
> 新機能は httptest を自前で持参してください。

---

## 5. コード基準

### 5.1 Lint 設定（`.golangci.yml`）

リポジトリの `.golangci.yml` は **v2 schema**（`version: "2"`）で、デフォルトの linter 集
（errcheck / govet / ineffassign / staticcheck / unused）に加え、テストファイルの未チェック
戻り値に対する限定的な除外を設定しています。gofmt は `scripts/ci.sh` が別途担保します。

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

### 6.4 Token を直接使用

```bash
# 推奨：環境変数（ディスクに残らず、シェル履歴にも残らない）
export GITEE_TOKEN=your_access_token
gitee auth status

# 明示的にトークンでログイン（~/.gitee-cli/settings.json に保存）
gitee auth login --token your_access_token
```

---

## 7. ビルドとリリース

### 7.1 ローカルビルド

```bash
go install ./cmd/gitee           # $GOPATH/bin にインストール（日常開発）
go build -o gitee ./cmd/gitee    # 一度きりのローカルビルド（PATH には入らない）
```

### 7.2 クロスコンパイル

```bash
# 手動クロスビルド（リリーススクリプトも同様）
GOOS=linux   GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee
GOOS=linux   GOARCH=arm64 go build -o dist/gitee-linux-arm64 ./cmd/gitee
GOOS=darwin  GOARCH=amd64 go build -o dist/gitee-darwin-amd64 ./cmd/gitee
GOOS=darwin  GOARCH=arm64 go build -o dist/gitee-darwin-arm64 ./cmd/gitee
GOOS=windows GOARCH=amd64 go build -o dist/gitee-windows-amd64.exe ./cmd/gitee
```

### 7.3 リリース

```bash
# 先にタグを打って push
git tag -a v0.13.2 -m v0.13.2 && git push origin v0.13.2

# 5 プラットフォーム成果物 + SHA256SUMS + Gitee release 作成 + アセットアップロード（冪等）
GITEE_TOKEN=... bash scripts/release.sh --publish
```

> Makefile も goreleaser もありません。リリースは `scripts/release.sh`（ビルド/パッケージ/
> チェックサム）と `scripts/publish-release.sh`（release 作成 + アセットアップロード、
> `gitee release create/upload` を再利用）で行います。

---

## 8. CI/CD パイプライン

**2 つの CI バックエンド、1 つのゲート定義**：

- `.github/workflows/ci.yml`（GitHub Actions ミラー）
- `.workflow/ci.yml`（Gitee Go）
- どちらも `scripts/ci.sh` を呼ぶ：gofmt → go vet → golangci-lint（任意）→ build →
  `go test -race -count=1` → クロスビルド → govulncheck

ローカルで CI を再現：

```bash
bash scripts/ci.sh          # 全部（クロスビルド + govulncheck）
bash scripts/ci.sh --fast   # クロスビルド + govulncheck を省略した高速ループ
```

---

## 9. プロジェクトマイルストーン

| フェーズ | ターゲットコマンド | 説明 |
|---------|------------------|------|
| **M0** | `gitee auth` 一式 | PAT 認証（OAuth2 デバイスフローなし）、Token 保存、`auth status/logout/token` |
| **M1** | `gitee repo` 一式 + `gitee api` | リポジトリ CRUD、生 API アクセス、設定モジュール |
| **M2** | `gitee issue` 一式 | Issue CRUD、コメント、ラベル管理 |
| **M3** | `gitee pr` 一式 | PR CRUD、マージ、レビュー、diff/checks |
| **M4** | `gitee gist` + `gitee search` | Gist 管理、3 種類の検索 |
| **M5** | `gitee org` + シェル補完 | 組織管理、Bash/Zsh/Fish/PowerShell 補完（i18n は未実装） |
| **M6** | エンタープライズデプロイ + マルチアカウント + プラグインシステム | `--hostname` 拡張、credentials マルチホスト、外部コマンド検出 |

---

## 10. 参考資料

| リソース | リンク |
|---------|-------|
| Gitee API v5 Swagger | https://gitee.com/api/v5/swagger |
| Gitee パーソナルアクセストークン | https://gitee.com/profile/personal_access_tokens |
| Cobra 公式ドキュメント | https://pkg.go.dev/github.com/spf13/cobra |
| Resty HTTP クライアント | https://github.com/go-resty/resty |
| glab プロジェクト（最良の参照実装） | https://github.com/gitlab-org/cli |
| gh CLI プロジェクト | https://github.com/cli/cli |
| Go プロジェクトレイアウト標準 | https://github.com/golang-standards/project-layout |
| Conventional Commits | https://www.conventionalcommits.org |
| golangci-lint ルールドキュメント | https://golangci-lint.run/usage/linters/ |