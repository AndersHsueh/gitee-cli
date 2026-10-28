# gitee-cli 機能仕様書（Functional Specification Document）

> ⚠️ **ステータス通知（v0.13.2、2026-10 メンテナンス）**：本文書は初期の設計ドラフトであり、
> **未実装または古くなった**記述を含みます。現在の挙動はコード、`gitee --help`、
> `gitee <cmd> --agent-help`、`gitee help reference`、`gitee help environment` が正となります。
> 現状と大きく乖離する段落は本文中で直接修正・注記しています。


バージョン：1.0.0（設計ドラフト。現状の修正は冒頭の通知を参照）| 言語：Go 1.27+ | 参照：Gitee API v5 + GitHub `gh` CLI

---

## 1. プロジェクト概要

### 1.1 プロジェクト情報

| フィールド | 値 |
|-----------|-----|
| プロジェクト名 | `gitee-cli` |
| タイプ | コマンドラインツール（CLI） |
| 言語 | Go 1.27+ |
| フレームワーク | Cobra（設定は手書き JSON、Viper 未使用） |
| 目標 | Gitee の公式 CLI ツール成為、GitHub `gh` に対抗 |
| API ベースURL | `https://gitee.com/api/v5` |
| 参照プロジェクト | [`gh`](https://github.com/cli/cli)、[`glab`](https://github.com/cli/go-gitlab) |

### 1.2 コア機能サマリー

```
gitee auth login|logout|status|refresh|token        # PAT 認証（OAuth2 デバイスフローなし）
gitee repo list|create|view|fork|edit|delete|clone|browse|branches  # リポジトリ管理
gitee repo tag|collaborator|star|watch|file|commits|compare|protect|archive|rename|sync  # リポジトリ拡張
gitee issue list|create|view|edit|close|reopen|comment  # Issue 管理
gitee pr list|create|view|merge|review|diff|checks|comments|checkout|close|reopen|edit|ready|status  # Pull Request
gitee gist list|create|view|edit|delete|star|clone|comments  # Gist
gitee org list|view|members|repos|issues                   # 組織（サブコマンドは複数形）
gitee search issues|users                            # 検索（repos は空、code/commits はこのインスタンスで非対応）
gitee api <method> <path> [--field key=val]          # 生 API アクセス
gitee config set|get|list|unset                      # 設定
gitee completion bash|zsh|fish|powershell            # シェル補完
gitee label|milestone|release|webhook|notification|ssh-key|user|alias|extension|agent|status  # その他のグループ
```

コマンド面の正は `gitee help reference`（各グループの `--help` と `--agent-help`）です。

---

## 2. グローバルフラグ

| フラグ | タイプ | 説明 | デフォルト値 |
|--------|--------|------|-------------|
| `--repo string` | string | 対象リポジトリ owner/repo（省略時は git remote から推定） | 自動 |
| `--hostname string` | string | Gitee サーバーアドレス（エンタープライズ/プライベートデプロイ） | `gitee.com` |
| `--token string` | string | パーソナルアクセストークン（PAT、設定より優先） | 設定から読み込み |
| `--api-url string` | string | API ルート URL | `https://{hostname}/api/v5` |
| `-o, --output string` | string | 出力形式 `table`\|`json`（**csv 非対応、不正値はエラー終了**） | `table` |
| `-P, --paginate` | bool | 自動ページネーション、全結果を取得 | `false` |
| `--page int` | int | ページ番号 | `1` |
| `--per-page int` | int | 1 ページあたりの件数（最大 100） | `30` |
| `--limit int` | int | 結果の最大数（`--paginate` 時はページ横断の総上限、`0` は無制限） | `30` |
| `--timeout int` | int | HTTP タイムアウト秒数 | `30` |
| `-v, --verbose` | bool | HTTP リクエスト/レスポンス詳細を出力 | `false` |
| `--no-color` | bool | ANSI カラー出力を無効化 | `false` |
| `--no-extensions` | bool | 外部拡張の読み込みをスキップ | `false` |
| `--agent-help` | bool | Agent 向けの長い構造化ヘルプ（一部コマンド、`-o json` 可） | `false` |

> 旧設計の `--config` / `--no-interaction` / `--insecure` / `--lang` は**存在しません**。
> 設定ファイルのパスは `~/.gitee-cli/settings.json` に固定です。

**環境変数（全 13 個、詳細は `gitee help environment`）：**

| 環境変数 | 対応/意味 |
|---------|----------|
| `GITEE_TOKEN` | アクセストークン、`--token` と同等 |
| `GITEE_HOSTNAME` | `--hostname` と同等 |
| `GITEE_API_URL` | `--api-url` と同等 |
| `GITEE_OUTPUT` | デフォルト出力形式、`--output` と同等。`table`/`json` 以外はエラー |
| `GITEE_VERBOSE` | `1`/`true`/`yes`/`on` で HTTP 詳細（stderr）を有効化、`--verbose` と同等 |
| `GITEE_TIMEOUT` | HTTP タイムアウト秒数、`--timeout` と同等。整数以外はエラー |
| `GITEE_LIMIT` | `--limit` のデフォルト値 |
| `GITEE_PAGINATE` | `1`/`true`/`yes`/`on` で `--paginate` を既定化 |
| `GITEE_NO_EXTENSIONS` | `1`/`true`/`yes`/`on` で拡張読み込みをスキップ |
| `NO_COLOR` | 非空でカラー無効、`--no-color` と同等 |
| `GITEE_ALIASES_FILE` | 別名ファイルのパス上書き |
| `GITEE_EXT_DIR` | 拡張ディレクトリの上書き |
| `BROWSER` | `repo browse` が使うブラウザ |

> プロキシは Go `net/http` 標準の変数（`HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY`）に従い、
> CLI 自身は解析しません。

---

## 3. コマンド詳細仕様

### 3.1 `gitee auth` — 認証管理

**方式：パーソナルアクセストークン（PAT）。** gitee-cli には OAuth2 デバイスフロー、
ブラウザ認証、`refresh_token` 交換は**ありません**。`auth login` は既存トークンを読み、
`GET /user` で検証します。

**トークン解決の優先順位：** `--token` フラグ > `GITEE_TOKEN` 環境変数 > `~/.gitee-cli/settings.json`。

| サブコマンド | 動作 |
|--------------|------|
| `gitee auth login [--token <PAT>]` | 引数なし：既存トークンを読み `/user` で検証。`--token` 指定時：検証後に `~/.gitee-cli/settings.json` へ保存（ディレクトリ 0700 / ファイル 0600） |
| `gitee auth logout` | 保存済みトークンを削除 |
| `gitee auth status` | `GET /user` でログイン状態とユーザー名を表示 |
| `gitee auth refresh` | スタブ実装：「Token refresh is not yet supported. Please re-authenticate with 'gitee auth login'.」と表示して終了コード 0。更新も再検証もせず、OAuth の `refresh_token` 交換でもない |
| `gitee auth token` | 現在のアクセストークンを出力（パイプ用） |

**`gitee auth login` の実際の流れ：**
```
1. 優先順位に従って候補トークンを解決（--token > GITEE_TOKEN > settings.json）
2. `Authorization: token <PAT>` で GET /user → 検証しユーザー名を取得
3. --token を明示した場合のみ ~/.gitee-cli/settings.json に保存（0600）
```
> ⚠️ `--token` の値は `ps` やシェル履歴に残ります。使い捨て以外は `GITEE_TOKEN` を推奨します。

**`gitee auth login` のフラグ：**

| フラグ | 説明 |
|--------|------|
| `-t, --token string` | トークンを指定（検証成功後に保存） |

> 旧設計の `--with-token`（stdin から読み込み）と `--scopes` は**存在しません**。

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
| `--sort string` | `created`\|`updated`\|`pushed`\|`full_name`\|`stars` | `full_name` |
| `--direction string` | `asc`\|`desc` | `asc` |
| `-L, --limit int` | 返すリポジトリ数の上限 | `30` |

> 旧稿の `--type` は存在しません。位置引数も受け付けません（`--repo` 指定か git remote から推定）。

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
| `LABELS` | `labels[].name` | ラベル（カンマ連結。無ければ `-`） |
| `UPDATED` | `updated_at` | 更新日（`YYYY-MM-DD`） |

> 表はこの 5 列のみで、`ASSIGNEE` 列はありません。

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
| `--state string` | `state` | `open`（`open`\|`closed`\|`merged`\|`all`） |
| `--head string` | `head` | — |
| `--base string` | `base` | — |

> 旧稿の `--author` / `--sort` / `--direction` は `pr list` に存在しません。

**`gitee pr merge` パラメータ：**

| パラメータ | 説明 |
|-----------|------|
| `--squash` | Squash マージ（body `squash=true`） |
| `--squash-message string` | Squash マージのコミットメッセージ（body `squash_message`） |

> 旧設計の `--rebase` / `--merge-commit-message` / `--delete-branch` は**存在しません**。
> CLI が実装しているのは squash と squash-message の 2 つだけです。

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
| `--file stringArray` | `files` | `filename=content` または `filename=@path`、複数指定可 |
| `--filename string` | — | stdin モードのファイル名（内容は stdin から） |
| `-i, --interactive` | — | 対話編集モード |

> `--from-stdin` はありません。stdin 入力は `--filename <name>`（任意で `--file` 併用）で行います。

**`gitee gist list` 出力フィールド：**

| フィールド | API フィールド | 説明 |
|-----------|---------------|------|
| `ID` | `id` | Gist ID |
| `DESCRIPTION` | `description` | 説明（50 文字で切り捨て、空なら `-`） |
| `PUBLIC` | `public` | 公開かどうか |
| `FILES` | `files` 数 | 含まれるファイル数 |
| `UPDATED` | `updated_at` | 更新時刻 |

---

### 3.6 `gitee org` — 組織管理

**対応 Gitee API：** `/orgs`, `/orgs/{org}`

| サブコマンド | Gitee API | 説明 |
|-------------|-----------|------|
| `gitee org list` | `GET /user/orgs` | ユーザーが属する組織を一覧表示 |
| `gitee org view <org>` | `GET /orgs/{org}` | 組織詳細を表示 |
| `gitee org members <org>` | `GET /orgs/{org}/members` | 組織メンバーを一覧表示 |
| `gitee org repos <org>` | `GET /orgs/{org}/repos` | 組織のリポジトリを一覧表示 |
| `gitee org issues <org>` | `GET /orgs/{org}/issues` | 組織内の Issue を一覧表示 |

---

### 3.7 `gitee search` — 検索

**対応 Gitee API：** `/search/*`

| サブコマンド | Gitee API | 説明 |
|-------------|-----------|------|
| `gitee search repos <query>` | `GET /search/repositories` | リポジトリを検索 |
| `gitee search issues <query>` | `GET /search/issues` | Issue を検索 |
| `gitee search users <query>` | `GET /search/users` | ユーザーを検索 |
| `gitee search code <query>` | `GET /search/code` | コード検索（この Gitee インスタンスでは 404、CLI が説明して非 0 終了） |
| `gitee search commits <query>` | `GET /search/commits` | コミット検索（この Gitee インスタンスでは 404、CLI が説明して非 0 終了） |

> ⚠️ 現状（v0.13.2 実測）：`search issues` / `search users` は動作。`search repos` はどのクエリでも
> 空（上流 API の挙動変化、CLI は `(no results)` を表示）。`search code` / `search commits` は
> このインスタンスでは利用できません。

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

### 4.3 CSV（**非対応**）

`-o/--output` は `table` と `json` のみ受け付けます。それ以外の値（初期ドキュメントが
謳っていた `csv` を含む）はエラー終了し、table へ黙ってフォールバックしません。
表形式のデータが必要な場合は `-o json` + `jq` を使ってください（`gitee help formatting` 参照）。

### 4.4 ページネーション出力

- `--paginate`：全ページを自動トラバース、単一結果セットにマージ
- `--page N`：ページ番号を指定（`1` 起始）
- `--per-page N`：1 ページあたりの件数（最大 100）
- `--limit N`：出力アイテムの合計を制限（ページネーションより優先、`0` = 無制限）

---

## 5. エラー処理仕様

| シナリオ | 終了コード | stderr 出力（`Error: <operation>: <detail>` 形式） |
|---------|-----------|------------|
| 未ログイン（ローカル判定・リクエストなし） | 1 | `not authenticated. Set GITEE_TOKEN or run 'gitee auth login'` |
| ネットワークエラー / タイムアウト | 1 | `<transport error>`（例：`context deadline exceeded`、`connection refused`） |
| API が 401 を返す | 1 | `HTTP 401: <Gitee message>`。body に message が無ければ `HTTP 401: authentication failed: run 'gitee auth login' or set GITEE_TOKEN` |
| API が 403 を返す | 1 | `HTTP 403: <Gitee message>`、無ければ `HTTP 403: permission denied` |
| API が 404 を返す | 1 | `HTTP 404: <Gitee message>`、無ければ `HTTP 404: resource does not exist or you lack read access` |
| API が 422 を返す | 1 | `HTTP 422: <Gitee message>`、無ければ `HTTP 422: validation failed` |
| API が 5xx を返す | 1 | `HTTP 5xx: <Gitee message>`、無ければ `internal server error` / `service unavailable` |
| パラメータエラー | 1 | `unknown flag: --x` / `unknown command "x" for "gitee ..."`（Cobra の出力。`Error: ` 接頭辞付き） |
| 操作キャンセル（確認で拒否） | 0 | `Cancelled` と表示し、何も実行しない |

---

## 6. 国際化（i18n）

- **未実装。** `--lang` も言語パックのディレクトリも実行時の言語切替もありません。
  コマンドヘルプと出力は固定文言です。
- 多言語対応はドキュメント層のみ：`README.md` / `README_en.md` / `README_ja.md` と
  `docs/{zh,en,ja}/`。

---

## 7. 非機能要件

| 要件 | 値 |
|------|---|
| 起動時間 | < 100ms（`gitee --version` コールドスタート） |
| 初回コンパイル時間 | < 20s（Go 1.27 増分コンパイル、マシン依存） |
| ネットワークタイムアウト | デフォルト 30s（`--timeout` でオーバーライド可能） |
| リトライ戦略 | **GET のみ**：5xx/ネットワークエラーを 3 回リトライ（2s 起点、最大 8s）。書き込みは自動リトライしない |
| ユーザー名の取得元 | `settings.json` の username を優先し、無ければ `GET /user` にフォールバック（TTL キャッシュなし） |
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
| Webhook | 6 | 実装済み（create / list / view / edit / delete / ping） |
| エンタープライズ（Enterprise） | 20+ | `--hostname` / `GITEE_HOSTNAME` でプライベートデプロイに対応 |