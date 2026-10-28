# gitee-cli

[中文](README.md) | [English](README_en.md) | **日本語**

> Gitee 公式 CLI ツール — ターミナルで Gitee の全操作が可能、GitHub の `gh` に対抗

[![Go Version](https://img.shields.io/badge/go-1.27+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.13.2-orange.svg)](https://gitee.com/andershsueh/gitee-cli/releases/latest)

---

## 機能一覧

```
gitee auth        # 認証（login / logout / status / token）                         ✅ 実装済
gitee issue       # Issue 管理（list / create / view / edit / close / reopen / comment） ✅ 実装済
gitee config      # 設定管理（set / get / list / unset）                           ✅ 実装済
gitee search      # Issue・ユーザー検索（repos は上流都合で空、code/commits は非対応） ⚠️ 一部制限
gitee completion  # シェル補完（bash / zsh / fish / powershell）                    ✅ 実装済
gitee version     # バージョン情報                                                 ✅ 実装済
gitee repo        # リポジトリ管理（list / create / view / fork / edit / delete / clone / browse / branches / star / watch / protect / file / commits / compare） ✅ 実装済
gitee pr          # Pull Request（list / create / view / merge / review / comments / diff） ✅ 実装済
gitee gist        # Gist（list / create / view / edit / delete / star / unstar）    ✅ 実装済
gitee org         # 組織管理（list / view / members / repos / issues）             ✅ 実装済
gitee api         # Gitee API v5 直接呼び出し（GET / POST / PUT / PATCH / DELETE）   ✅ 実装済
gitee --agent-help # 構造化された Agent 向けヘルプ（コマンドごとに 11 次元）          ✅ 実装済
```

> `search` の実状（v0.13.2 実測）：`gitee search issues` / `gitee search users` は動作します。
> `gitee search repos` は現状どのクエリでも 0 件（上流 API の挙動変化）で `(no results)` を表示。
> `gitee search code` / `gitee search commits` はこの Gitee インスタンスで 404 になり、
> CLI はその旨を説明して非 0 で終了します（成功したふりをしません）。

> 📖 `--help` / `--agent-help` の二面設計の全体像は中国語 [README.md](README.md) と
> [docs/philosophy.md](docs/philosophy.md) を参照してください。

---

## クイックスタート

### ソースからビルド

```bash
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go install ./cmd/gitee    # $(go env GOPATH)/bin にインストール
```

### 認証

`~/.gitee-cli/settings.json` に Token を設定（`settings.json.sample` を参照）：

```json
{
  "hosts": {
    "gitee.com": {
      "token": "your_access_token",
      "username": "your_username"
    }
  }
}
```

または環境変数を使用：

```bash
export GITEE_TOKEN=your_access_token
```

---

## インストール

**バイナリダウンロード**

[Releases](https://gitee.com/andershsueh/gitee-cli/releases/latest) からプラットフォームに合うアーカイブをダウンロード：

| プラットフォーム | ファイル名 |
|----------------|------------|
| Linux amd64 | `gitee-linux-amd64.tar.gz` |
| Linux arm64 | `gitee-linux-arm64.tar.gz` |
| macOS amd64 | `gitee-darwin-amd64.tar.gz` |
| macOS arm64（M1/M2） | `gitee-darwin-arm64.tar.gz` |
| Windows amd64 | `gitee-windows-amd64.exe.zip` |

各アーカイブにはバイナリと `LICENSE` が含まれ、`SHA256SUMS.txt` も同時に公開されます
（Linux/macOS は `sha256sum -c SHA256SUMS.txt`、Windows は `certutil -hashfile <file> SHA256`）。

**ソースからビルド**

```bash
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go install ./cmd/gitee    # $(go env GOPATH)/bin にインストール
```

### 認証

`gitee auth login` は OAuth デバイスフローやブラウザ認証を**行いません**。既存の
パーソナルアクセストークン（`GITEE_TOKEN` または `~/.gitee-cli/settings.json`）を読み、
`/user` で検証します：

```bash
export GITEE_TOKEN=your_access_token   # 推奨：ディスクに残らない
gitee auth status                      # Token を検証

# または ~/.gitee-cli/settings.json（0600）に保存
gitee auth login --token your_access_token
```

`gitee auth refresh` は現在スタブです。トークン更新は未対応である旨と、`gitee auth login` で
再認証するよう表示するだけで、実際の更新は行いません（OAuth の `refresh_token` 交換でもありません）。

> ⚠️ `--token` の値は `ps` で他ユーザーから見え、シェル履歴にも残ります。使い捨ての
> スクリプト以外では `GITEE_TOKEN` を推奨します。

---

## 使用例

### リポジトリ管理

```bash
# 自分のリポジトリを一覧表示
gitee repo list

# ユーザーの公開リポジトリを一覧表示
gitee repo list --owner torvalds --visibility public

# リポジトリ作成
gitee repo create --name my-project --description "私の新規プロジェクト" --private

# リポジトリをクローン
gitee repo clone owner/repo-name

# リポジトリをフォーク
gitee repo fork owner/repo-name

# ブラウザでリポジトリを開く
gitee repo browse owner/repo-name
```

### Issue 管理

```bash
# 現在のリポジトリの open Issue を一覧表示
gitee issue list

# 全てのステータスを表示、ラベルでフィルター
gitee issue list --state all --labels bug,enhancement

# Issue 作成
gitee issue create --title "バグを発見しました" --body "再現手順：..." --labels bug

# Issue 詳細を表示
gitee issue view 42

# Issue をクローズ
gitee issue close 42

# コメントを追加
gitee issue comment 42 --body "報告ありがとうございます、修正しました"
```

### Pull Request

```bash
# 全ての open PR を一覧表示
gitee pr list

# PR 作成
gitee pr create --title "feat: 検索機能を追加" --source feature/search --target main

# PR の diff を表示
gitee pr diff 18

# PR をマージ（squash）
gitee pr merge 18 --squash

# PR をレビュー
gitee pr review 18 --approve
gitee pr review 18 --request-changes --body "ユニットテストを追加してください"
```

### 検索

```bash
# Issue を検索（利用可）
gitee search issues "メモリリーク" --state open

# ユーザーを検索（利用可）
gitee search users "john"

# ⚠️ リポジトリ検索は上流都合で現在 0 件
gitee search repos "web framework" --language go --sort stars   # ⚠️ 現在 0 件（非対応ではない）

# ⚠️ この Gitee インスタンスでは code / commits 検索は非対応
gitee search code "func main" --repo owner/repo   # ⚠️ このインスタンスでは 404・非対応
```

### API 直接呼び出し

```bash
# GET リクエスト
gitee api GET /user
gitee api GET /user/repos?type=owner&per_page=10

# POST リクエスト（--field でパラメータ指定）
gitee api POST /user/repos --field name=test --field private=true

# 全ての結果をページネーション
gitee api GET /user/repos --paginate

# HTTP のやり取りを表示
gitee api GET /user --verbose
```

### 設定管理

```bash
gitee config set editor vim         # エディタを設定
gitee config set protocol ssh       # クローンに SSH を使用
gitee config get protocol           # 設定を読み取る
gitee config list                   # 全ての設定を表示
gitee config unset editor           # 設定項目を削除
```

---

## グローバルオプション

| オプション | 説明 | デフォルト値 |
|-----------|------|-------------|
| `-R, --repo string` | 対象リポジトリ（owner/repo）。省略時は git remote から推定 | 自動 |
| `--token string` | Access Token（設定をオーバーライド）。`GITEE_TOKEN` 推奨 | 設定から読み込み |
| `--hostname string` | Gitee インスタンスアドレス（企業用） | `gitee.com` |
| `--api-url string` | API ベース URL（非標準のプライベート配置用） | `https://{hostname}/api/v5` |
| `-o, --output string` | 出力形式：`table` \| `json`（**CSV 非対応**、不正値はエラー） | `table` |
| `-P, --paginate` | 自動ページネーション、全結果を取得 | `false` |
| `--limit int` | 結果の最大数（`--paginate` 時はページ横断の総上限、`0` は無制限） | `30` |
| `--page int` | ページ番号（`--paginate` なしのとき） | `1` |
| `--per-page int` | 1 ページあたりの件数（最大 100） | `30` |
| `--agent-help` | Agent 向けの長い構造化ヘルプ（一部コマンド） | `false` |
| `--no-extensions` | 外部拡張の読み込みをスキップ | `false` |
| `--verbose` | HTTP リクエスト/レスポンスの詳細を表示 | `false` |
| `--no-color` | ANSI カラーを無効化 | `false` |
| `--timeout int` | HTTP タイムアウト秒数 | `30` |

**環境変数**（全 13 個。完全な一覧は `gitee help environment`）：

| 変数 | 説明 |
|------|------|
| `GITEE_TOKEN` | Access Token |
| `GITEE_HOSTNAME` | Gitee インスタンスアドレス |
| `GITEE_API_URL` | API ベース URL |
| `GITEE_OUTPUT` | デフォルト出力形式 `table` \| `json` |
| `GITEE_VERBOSE` | `1`/`true`/`yes`/`on` で HTTP 詳細を stderr に表示 |
| `GITEE_TIMEOUT` | HTTP タイムアウト秒数 |
| `GITEE_LIMIT` | `--limit` のデフォルト値 |
| `GITEE_PAGINATE` | `1`/`true`/`yes`/`on` で `--paginate` を既定化 |
| `GITEE_NO_EXTENSIONS` | `1`/`true`/`yes`/`on` で拡張読み込みをスキップ |
| `NO_COLOR` | 非空でカラー無効（標準変数） |
| `GITEE_ALIASES_FILE` | 別名ファイルのパス上書き（既定 `~/.gitee-cli/aliases.json`） |
| `GITEE_EXT_DIR` | 拡張ディレクトリ上書き（既定 `~/.gitee-cli/extensions`） |
| `BROWSER` | `repo browse` が使うブラウザ |

> プロキシは Go `net/http` 標準の変数（`HTTPS_PROXY` / `NO_PROXY`）に従います。
> CLI 自身はこれらを解析しません。

---

## シェル補完

```bash
# Bash
echo 'eval "$(gitee completion bash)"' >> ~/.bashrc

# Zsh
gitee completion zsh > "${fpath[1]}/_gitee"

# Fish
gitee completion fish > ~/.config/fish/completions/gitee.fish

# PowerShell
gitee completion powershell >> $PROFILE
```

---

## 企業用 / プライベートデプロイ

```bash
# 企業インスタンスにログイン
gitee auth login --hostname your-company.gitee.com

# または環境変数で
export GITEE_HOSTNAME=your-company.gitee.com
gitee repo list
```

---

## 開発

[`docs/ja/developer-guide.md`](docs/ja/developer-guide.md) を参照

```bash
# クローン
git clone https://gitee.com/andershsueh/gitee-cli.git && cd gitee-cli

# ビルド
go build -o bin/gitee ./cmd/gitee

# テスト
go test ./... -race -count=1

# リント
golangci-lint run ./...

# 機能テスト
./test/run.sh
```

### プロジェクトドキュメント

| ドキュメント | 説明 |
|------------|------|
| [`docs/ja/FSD.md`](docs/ja/FSD.md) | 機能仕様書 |
| [`docs/ja/SDD.md`](docs/ja/SDD.md) | システム設計書 |
| [`docs/ja/developer-guide.md`](docs/ja/developer-guide.md) | 開発者ガイド |

---

## 参考資料

- [Gitee API v5 ドキュメント](https://gitee.com/api/v5/swagger)
- [Gitee パーソナルアクセストークン](https://gitee.com/profile/personal_access_tokens)
- [GitHub gh CLI](https://github.com/cli/cli)（デザイン参照）
- [GitLab glab CLI](https://github.com/gitlab-org/cli)（デザイン参照）

---

## License

[Mulan PSL v2](LICENSE)

---

## Buy Me Coffee

このプロジェクトが役に立ったなら、コーヒーを請いたいです：

![Buy Me Coffee](./etc/BuyMeCoffe.jpg)