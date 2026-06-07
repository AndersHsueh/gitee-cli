# gitee-cli 機能仕様書（Functional Specification Document）

バージョン：1.0.0 | 言語：Go 1.23+ | 参照：Gitee API v5 + GitHub `gh` CLI

---

## 1. プロジェクト概要

### 1.1 プロジェクト情報

| フィールド | 値 |
|-----------|-----|
| プロジェクト名 | `gitee-cli` |
| タイプ | コマンドラインツール（CLI） |
| 言語 | Go 1.23+ |
| フレームワーク | Cobra + Viper |
| 目標 | Gitee の公式 CLI ツール成為、GitHub `gh` に対抗 |
| API ベースURL | `https://gitee.com/api/v5` |
| 参照プロジェクト | [`gh`](https://github.com/cli/cli)、[`glab`](https://github.com/cli/go-gitlab) |

### 1.2 コア機能サマリー

```
gitee auth login|logout|status|refresh|token   # OAuth2 認証
gitee repo list|create|clone|fork|view|edit|delete|browse  # リポジトリ管理
gitee issue list|create|view|edit|close|reopen|comment     # Issue 管理
gitee pr list|create|view|merge|close|reopen|review|diff|checks  # Pull Request
gitee gist list|create|view|edit|delete|star|clone         # Gist
gitee org list|view|member|repo|issue                      # 組織
gitee search repos|issues|users                            # 検索
gitee api <method> <path> [--field key=val]                # 生 API アクセス
gitee config set|get|list|unset                            # 設定
gitee completion bash|zsh|fish|powershell                  # シェル補完
```

---

## 2. グローバルフラグ

| フラグ | タイプ | 説明 | デフォルト値 |
|--------|--------|------|-------------|
| `--hostname string` | string | Gitee サーバーアドレス（エンタープライズ/プライベートデプロイ） | `gitee.com` |
| `--token string` | string | OAuth Access Token（設定より優先） | 設定から読み込み |
| `-o, --output string` | string | 出力形式 (`table`\|`json`\|`csv`) | `table` |
| `-P, --paginate` | bool | 自動ページネーション、全結果を取得 | `false` |
| `--page int` | int | ページ番号 | `1` |
| `--per-page int` | int | 1 ページあたりの件数（最大 100） | `30` |
| `--limit int` | int | 結果の最大数（0 = 無制限） | `0` |
| `-c, --config string` | string | 設定ファイルパス | `~/.gitee-cli/settings.json` |
| `--api-url string` | string | API ルート URL | `https://gitee.com/api/v5` |
| `--timeout int` | int | ネットワークタイムアウト秒数 | `30` |
| `-v, --verbose` | bool | デバッグ情報出力（HTTP リクエスト/レスポンス詳細） | `false` |
| `--no-color` | bool | ANSI カラー出力を無効化 | `false` |
| `--no-interaction` | bool | サイレントモード、インタラクティブプロンプトを無効化（CI/CD） | `false` |
| `--insecure` | bool | 安全でない HTTPS 接続を許可 | `false` |
| `--lang string` | string | UI 言語 (`zh`\|`en`) | `zh` |

**環境変数マッピング（フラグより優先、設定より低い）：**

| 環境変数 | 対応フラグ |
|---------|-----------|
| `GITEE_TOKEN` | `--token` |
| `GITEE_HOSTNAME` | `--hostname` |
| `GITEE_OUTPUT` | `--output` |
| `GITEE_API_URL` | `--api-url` |
| `HTTPS_PROXY` / `HTTP_PROXY` | プロキシ設定 |
| `NO_COLOR` | `--no-color` |

---

## 3. コマンド詳細仕様

### 3.1 `gitee auth` — 認証管理

**対応 Gitee API：** OAuth2 Device Flow（`/oauth/authorizations/device/*`）

| サブコマンド | Gitee API エンドポイント | 説明 |
|-------------|------------------------|------|
| `gitee auth login` | `POST /oauth/authorizations/device/code` + ポーリング | OAuth2 デバイスフロー開始、Access Token を取得 |
| `gitee auth logout` | ローカル操作 | `~/.gitee-cli/settings.json` から Token を削除 |
| `gitee auth status` | `GET /user` | 現在のログイン状態を表示（ユーザー名、Token 有効期限） |
| `gitee auth refresh` | `POST /oauth/token?grant_type=refresh_token` | 期限切れ Token を更新 |
| `gitee auth token` | ローカル操作 | 現在の Access Token を出力（パイプラピング用） |

**`gitee auth login` フルフロー：**
```
1. POST /oauth/authorizations/device/code
   Body: { client_id, scope }
   → device_code, user_code, verification_uri, expires_in, interval

2. プロンプトを表示：
   ! 一回限りのコード XXXX-XXXX をコピーしてください
   - Enter を押して gitee.com/device をブラウザで開く

3. ブラウザを開く / または --no-interaction モードで URL を印刷

4. [5 秒ごとにポーリング]
   POST /oauth/authorizations/device/token
   Body: { client_id, device_code, grant_type="urn:ietf:params:oauth:grant-type:device_code" }
   → access_token, token_type, scope, refresh_token

5. access_token を ~/.gitee-cli/settings.json に書き込み（0600 権限）

6. GET /user → 検証して "✓ Logged in as <username>" を表示
```

**`gitee auth login` パラメータ：**

| パラメータ | 説明 |
|-----------|------|
| `--scopes string` | リクエストする OAuth 権限スコープ（デフォルト：`user_info,projects,pull_requests,issues,notes,gists`） |
| `--with-token` | 既存の Token を stdin から読み込んで保存（OAuth フローをスキップ） |

---

### 3.2 `gitee repo` — リポジトリ管理

**対応 Gitee API：** `/repos`, `/user/repos`, `/orgs/{org}/repos`

| サブコマンド | Gitee API | 説明 |
|-------------|-----------|------|
| `gitee repo list` | `GET /user/repos` | 現在のユーザーのリポジトリを一覧表示 |
| `gitee repo create` | `POST /user/repos` | 新規リポジトリを作成 |
| `gitee repo clone <repo>` | `git clone` を呼び出す | リポジトリをローカルにクローン |
| `gitee repo fork <repo>` | `POST /repos/{owner}/{repo}/forks` | リポジトリをフォーク |
| `gitee repo view <repo>` | `GET /repos/{owner}/{repo}` | リポジトリ詳細を表示 |
| `gitee repo edit <repo>` | `PATCH /repos/{owner}/{repo}` | リポジトリ設定を編集 |
| `gitee repo delete <repo>` | `DELETE /repos/{owner}/{repo}` | リポジトリを削除（`--yes` 確認が必要） |
| `gitee repo browse <repo>` | ブラウザを開く | ブラウザでリポジトリページを開く |

**`gitee repo list` 出力フィールド：**

| フィールド | API フィールド | 説明 |
|-----------|---------------|------|
| `NAME` | `full_name` | owner/repo 形式 |
| `VISIBILITY` | `private` + `internal` | public/private/internal |
| `LANGUAGE` | `language` | 主なプログラミング言語 |
| `STARS` | `stargazers_count` | スター数 |
| `FORKS` | `forks_count` | フォーク数 |
| `UPDATED` | `updated_at` | 最終更新時刻（相対時間） |

**`gitee repo list` パラメータ：**

| パラメータ | 説明 | デフォルト |
|-----------|------|-----------|
| `--owner string` | ユーザー/組織名を指定 | 現在のログインユーザー |
| `--visibility string` | `public`\|`private`\|`internal`\|`all` | `all` |
| `--sort string` | `created`\|`updated`\|`pushed`\|`full_name` | `updated` |
| `--direction string` | `asc`\|`desc` | `desc` |
| `--type string` | `owner`\|`member`\|`all` | `owner` |

**`gitee repo create` パラメータ：**

| パラメータ | 必須 | API フィールド | 説明 |
|-----------|------|---------------|------|
| `--name string` | ✅ | `name` | リポジトリ名 |
| `--description string` | ❌ | `description` | リポジトリの説明 |
| `--private` | ❌ | `private=true` | プライベートに設定 |
| `--auto-init` | ❌ | `auto_init=true` | README を初期化（デフォルト true） |
| `--gitignore string` | ❌ | `gitignore_template` | .gitignore テンプレート名 |
| `--license string` | ❌ | `license_template` | ライセンステンプレート名 |
| `--org string` | ❌ | — | 組織下で作成（`POST /orgs/{org}/repos` を使用） |

---

### 3.3 `gitee issue` — Issue 管理

**対応 Gitee API：** `/repos/{owner}/{repo}/issues`

| サブコマンド | Gitee API | 説明 |
|-------------|-----------|------|
| `gitee issue list` | `GET /repos/{owner}/{repo}/issues` | Issue を一覧表示 |
| `gitee issue create` | `POST /repos/{owner}/{repo}/issues` | Issue を作成 |
| `gitee issue view <number>` | `GET /repos/{owner}/{repo}/issues/{number}` | 詳細を表示 |
| `gitee issue edit <number>` | `PATCH /repos/{owner}/{repo}/issues/{number}` | Issue を編集 |
| `gitee issue close <number>` | `PATCH /repos/{owner}/{repo}/issues/{number}` body `state=closed` | Issue をクローズ |
| `gitee issue reopen <number>` | `PATCH /repos/{owner}/{repo}/issues/{number}` body `state=open` | Issue を再オープン |
| `gitee issue comment <number>` | `POST /repos/{owner}/{repo}/issues/{number}/comments` | コメントを追加 |
| `gitee issue label list` | `GET /repos/{owner}/{repo}/labels` | リポジトリラベルを一覧表示 |

**`gitee issue list` パラメータ：**

| パラメータ | API フィールド | デフォルト |
|-----------|---------------|-----------|
| `--state string` | `state` | `open` |
| `--labels string` | `labels`（カンマ区切り） | — |
| `--assignee string` | `assignee` | — |
| `--milestone string` | `milestone` | — |
| `--repo string` | `{owner}/{repo}` | git remote から検出 |

**`gitee issue list` 出力フィールド：**

| フィールド | API フィールド | 説明 |
|-----------|---------------|------|
| `#` | `number` | Issue 番号 |
| `TITLE` | `title` | タイトル（60 文字で切り捨て） |
| `STATE` | `state` | open/closed |
| `LABELS` | `labels[].name` | ラベル（最大 3 個表示） |
| `ASSIGNEE` | `assignee.login` | 担当者 |
| `UPDATED` | `updated_at` | 更新日（相対時間） |

**`gitee issue create` パラメータ：**

| パラメータ | API フィールド | 必須 |
|-----------|---------------|------|
| `--title string` | `title` | ✅ |
| `--body string` | `body` | ❌ |
| `--labels string` | `labels`（カンマ区切り） | ❌ |
| `--assignees string` | `assignees`（カンマ区切り） | ❌ |
| `--milestone int` | `milestone_number` | ❌ |
| `--repo string` | `{owner}/{repo}` | git remote から検出 |

---

### 3.4 `gitee pr` — Pull Request 管理

**対応 Gitee API：** `/repos/{owner}/{repo}/pulls`

| サブコマンド | Gitee API | 説明 |
|-------------|-----------|------|
| `gitee pr list` | `GET /repos/{owner}/{repo}/pulls` | PR を一覧表示 |
| `gitee pr create` | `POST /repos/{owner}/{repo}/pulls` | PR を作成 |
| `gitee pr view <number>` | `GET /repos/{owner}/{repo}/pulls/{number}` | PR 詳細を表示 |
| `gitee pr merge <number>` | `PUT /repos/{owner}/{repo}/pulls/{number}/merge` | PR をマージ |
| `gitee pr close <number>` | `PATCH /repos/{owner}/{repo}/pulls/{number}` `state=closed` | PR をクローズ |
| `gitee pr reopen <number>` | `PATCH /repos/{owner}/{repo}/pulls/{number}` `state=open` | PR を再オープン |
| `gitee pr review <number>` | `POST /repos/{owner}/{repo}/pulls/{number}/reviews` | PR をレビュー |
| `gitee pr diff <number>` | `GET /repos/{owner}/{repo}/pulls/{number}` Accept: diff | diff コンテンツを出力 |
| `gitee pr checks <number>` | `GET /repos/{owner}/{repo}/statuses/{sha}` | CI チェック状態 |

**`gitee pr list` パラメータ：**

| パラメータ | API フィールド | デフォルト |
|-----------|---------------|-----------|
| `--state string` | `state` | `open` |
| `--author string` | `author` | — |
| `--head string` | `head` | — |
| `--base string` | `base` | — |
| `--sort string` | `sort` | `created` |
| `--direction string` | `direction` | `desc` |

**`gitee pr merge` パラメータ：**

| パラメータ | API フィールド | 説明 |
|-----------|---------------|------|
| `--squash` | `merge_method=squash` | Squash マージ |
| `--rebase` | `merge_method=rebase` | Rebase マージ |
| `--merge-commit-message string` | `merge_commit_message` | マージコミットメッセージ |
| `--delete-branch` | — | マージ後にソースブランチを削除 |

---

### 3.5 `gitee gist` — Gist

**対応 Gitee API：** `/gists`

| サブコマンド | Gitee API | 説明 |
|-------------|-----------|------|
| `gitee gist list` | `GET /gists` | ユーザーの gist を一覧表示 |
| `gitee gist create [files...]` | `POST /gists` | Gist を作成 |
| `gitee gist view <id>` | `GET /gists/{id}` | Gist コンテンツを表示 |
| `gitee gist edit <id>` | `PATCH /gists/{id}` | Gist を編集 |
| `gitee gist delete <id>` | `DELETE /gists/{id}` | Gist を削除 |
| `gitee gist star <id>` | `PUT /gists/{id}/star` | Gist をスター |
| `gitee gist clone <id>` | `git clone` を呼び出す | Gist をローカル git リポジトリとしてクローン |

**`gitee gist create` パラメータ：**

| パラメータ | API フィールド | 説明 |
|-----------|---------------|------|
| `--description string` | `description` | Gist の説明 |
| `--public` | `public=true` | 公開に設定（デフォルト private） |
| `--from-stdin` | — | stdin からコンテンツを読み取る（`--filename` と共に使用） |
| `--filename string` | — | stdin モードでのファイル名 |

**`gitee gist list` 出力フィールド：**

| フィールド | API フィールド | 説明 |
|-----------|---------------|------|
| `ID` | `id` | Gist ID |
| `DESCRIPTION` | `description` | 説明（50 文字で切り捨て） |
| `FILES` | `files` 数 | 含まれるファイル数 |
| `PUBLIC` | `public` | 公開かどうか |
| `UPDATED` | `updated_at` | 更新時刻 |

---

### 3.6 `gitee org` — 組織管理

**対応 Gitee API：** `/orgs`, `/orgs/{org}`

| サブコマンド | Gitee API | 説明 |
|-------------|-----------|------|
| `gitee org list` | `GET /user/orgs` | ユーザーが属する組織を一覧表示 |
| `gitee org view <org>` | `GET /orgs/{org}` | 組織詳細を表示 |
| `gitee org member list <org>` | `GET /orgs/{org}/members` | 組織メンバーを一覧表示 |
| `gitee org repo list <org>` | `GET /orgs/{org}/repos` | 組織のリポジトリを一覧表示 |
| `gitee org issue list <org>` | `GET /orgs/{org}/issues` | 組織内の Issue を一覧表示 |

---

### 3.7 `gitee search` — 検索

**対応 Gitee API：** `/search/*`

| サブコマンド | Gitee API | 説明 |
|-------------|-----------|------|
| `gitee search repos <query>` | `GET /search/repositories` | リポジトリを検索 |
| `gitee search issues <query>` | `GET /search/issues` | Issue を検索 |
| `gitee search users <query>` | `GET /search/users` | ユーザーを検索 |

**`gitee search repos` パラメータ：**

| パラメータ | API フィールド | 説明 |
|-----------|---------------|------|
| `--language string` | `language` | プログラミング言語でフィルター |
| `--sort string` | `sort` | `stars`\|`forks`\|`updated` |
| `--order string` | `order` | `asc`\|`desc`（デフォルト `desc`） |
| `--limit int` | `per_page` | 結果の最大数 |

---

### 3.8 `gitee api` — 生 API アクセス

Gitee API v5 リクエストを直接実行、デバッグと自動化スクリプト用：

```bash
gitee api GET /user
gitee api GET /user/repos?type=owner
gitee api POST /user/repos --field name=my-repo --field private=true
gitee api DELETE /repos/owner/repo
gitee api PATCH /repos/owner/repo --field description="new desc"
```

**パラメータ：**

| パラメータ | 説明 |
|-----------|------|
| `<method>` | HTTP メソッド：GET / POST / PUT / PATCH / DELETE |
| `<path>` | API パス（例：`/user/repos`）、`--api-url` 接頭辞が自動追加 |
| `--field key=value` | リクエストボディパラメータ（JSON）、複数回使用可能 |
| `--header key=value` | カスタムリクエストヘッダー、複数回使用可能 |
| `--raw-field key=value` | 文字列型パラメータ（JSON 型推論なし） |
| `--input file` | ファイルからリクエストボディを読み取る（JSON） |
| `--include` | レスポンスヘッダーを出力 |
| `--paginate` | 自動ページネーションと JSON 配列のマージ |

---

### 3.9 `gitee config` — 設定管理

| サブコマンド | 説明 |
|-------------|------|
| `gitee config list` | 全設定を一覧表示（ソース含む：file/env/default） |
| `gitee config set <key> <value>` | 設定項目を設定（`~/.gitee-cli/settings.json` に書き込み） |
| `gitee config get <key>` | 特定の設定項目を読み取る |
| `gitee config unset <key>` | 設定項目を削除 |

**設定可能項目：**

| 設定キー | タイプ | デフォルト値 | 説明 |
|---------|--------|-------------|------|
| `protocol` | string | `https` | git プロトコル (`ssh`\|`https`) |
| `editor` | string | `$EDITOR` | コミットメッセージ/ボディエディタ |
| `browser` | string | システムデフォルト | ブラウザオープンコマンド |
| `prompt` | string | `enabled` | インタラクティブプロンプトスイッチ (`enabled`\|`disabled`) |
| `output` | string | `table` | デフォルト出力形式 |
| `pager` | string | `$PAGER` | ページャーコマンド |

---

### 3.10 `gitee completion` — シェル補完

| サブコマンド | 説明 |
|-------------|------|
| `gitee completion bash` | Bash 補完スクリプトを出力 |
| `gitee completion zsh` | Zsh 補完スクリプトを出力 |
| `gitee completion fish` | Fish 補完スクリプトを出力 |
| `gitee completion powershell` | PowerShell 補完スクリプトを出力 |

**インストール例：**
```bash
# Bash
gitee completion bash >> ~/.bashrc

# Zsh
gitee completion zsh > "${fpath[1]}/_gitee"

# Fish
gitee completion fish > ~/.config/fish/completions/gitee.fish
```

---

## 4. 出力形式仕様

### 4.1 テーブル形式（デフォルト `--output table`）

- 等幅列アライメント、UTF-8 エンコーディング
- 列幅はターミナル幅に自動適合、最大列は 60 文字で切り捨て（末尾 `...`）
- 表頭は大文字-bold（ANSI `\033[1m`）
- 時間フィールドは相対時間を表示（例：`2 days ago`）

### 4.2 JSON 形式（`--output json`）

- 整形 JSON 出力（2 スペースインデント）
- `list` コマンドは JSON 配列を返す
- `view` コマンドは単一の JSON オブジェクトを返す
- エラーメッセージは stderr に出力（パイプラに影響なし）

### 4.3 CSV 形式（`--output csv`）

- 最初の行は見出し（小文字フィールド名）
- 値はダブルクォートでラップ、内部のダブルクォートは `""` でエスケープ
- Excel/データベースへのインポートに適している

### 4.4 ページネーション出力

- `--paginate`：全ページを自動トラバース、単一結果セットにマージ
- `--page N`：ページ番号を指定（`1` 起始）
- `--per-page N`：1 ページあたりの件数（最大 100）
- `--limit N`：出力アイテムの合計を制限（ページネーションより優先、`0` = 無制限）

---

## 5. エラー処理仕様

| シナリオ | 終了コード | stderr 出力 |
|---------|-----------|------------|
| 未ログイン | 1 | `Error: not authenticated. Run 'gitee auth login' first.` |
| ネットワークタイムアウト | 1 | `Error: failed to connect to gitee.com: <detail>` |
| API が 401 を返す | 1 | `Error: authentication failed. Token may be expired. Run 'gitee auth refresh'.` |
| API が 403 を返す | 1 | `Error: insufficient permissions: <message>` |
| API が 404 を返す | 1 | `Error: not found: <resource>` |
| API が 422 を返す | 1 | `Error: validation failed: <field errors>` |
| API が 5xx を返す | 1 | `Error: Gitee server error (HTTP <code>): <message>` |
| パラメータエラー | 2 | `Error: <description>` + 使用法ヘルプ |
| Token 期限切れ | 1 | `gitee auth refresh` または `gitee auth login` をプロンプト |
| 操作がキャンセルされた | 1 | `Aborted.` |

---

## 6. 国際化（i18n）

- デフォルト言語：中国語（`zh-CN`）
- `--lang en` で英語出力に切り替え可能（または `GITEE_LANG=en`）
- 言語パックは `internal/i18n/` ディレクトリにある（`zh_CN.go`、`en_US.go`）
- エラーメッセージ、ヘルptest、コマンドの説明はすべて i18n をサポート

---

## 7. 非機能要件

| 要件 | 値 |
|------|---|
| 起動時間 | < 100ms（`gitee --version` コールドスタート） |
| 初回コンパイル時間 | < 15s（Go 1.23 增量コンパイル） |
| ネットワークタイムアウト | デフォルト 30s（`--timeout` でオーバーライド可能） |
| リトライ戦略 | 5xx は自動リトライ 3 回（指数バックオフ、最大 8s） |
| Token キャッシュ | ユーザー情報は 5 分間キャッシュ（`GET /user` の繰り返しを回避） |
| プロキシ | `HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY` 環境変数を尊重 |
| クロスプラットフォーム | Linux amd64/arm64、macOS amd64/arm64（M1）、Windows amd64 |
| バイナリサイズ | < 20MB（`-ldflags="-s -w"` で圧縮有効） |

---

## 8. Gitee API v5 エンドポイントカバレッジマトリックス

| 機能モジュール | API エンドポイント数 | gitee-cli カバレッジ |
|-------------|---------------------|-------------------|
| リポジトリ（Repos） | 75+ | 8 コアエンドポイント |
| Issue | 23 | 8 エンドポイント |
| Pull Request | 27 | 9 エンドポイント |
| Gist | 17 | 7 エンドポイント |
| 組織（Org） | 14 | 5 エンドポイント |
| ユーザー（User） | 18 | auth/search 経由 |
| 検索（Search） | 3 | 3 エンドポイント（全カバレッジ） |
| Webhook | 6 | まだ未カバレッジ（v2.0 プラン） |
| エンタープライズ（Enterprise） | 20+ | まだ未カバレッジ（v2.0 プラン） |