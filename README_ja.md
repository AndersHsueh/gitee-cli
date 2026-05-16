# gitee-cli

> Gitee 公式 CLI ツール — ターミナルで Gitee の全操作が可能、GitHub の `gh` に対抗

[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.3.0-orange.svg)]()

---

## 機能一覧

```
gitee auth        # 認証（login / logout / status / token）                         ✅ 実装済
gitee issue       # Issue 管理（list / create / view / edit / close / reopen / comment） ✅ 実装済
gitee config      # 設定管理（set / get / list / unset）                           ✅ 実装済
gitee search      # リポジトリ / Issue / ユーザー検索                              ✅ 実装済
gitee completion  # シェル補完（bash / zsh / fish / powershell）                    ✅ 実装済
gitee version     # バージョン情報                                                 ✅ 実装済
gitee repo        # リポジトリ管理（list / create / view / fork / edit / delete / clone / browse / branches） ✅ 実装済
gitee pr          # Pull Request（list / create / view / merge / review / comments / diff） ✅ 実装済
gitee gist        # Gist（list / create / view / edit / delete / star / unstar）    ✅ 実装済
gitee org         # 組織管理（list / view / members / repos / issues）             ✅ 実装済
gitee api         # Gitee API v5 直接呼び出し（GET / POST / PUT / PATCH / DELETE）   ✅ 実装済
```

---

## クイックスタート

### ソースからビルド

```bash
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go build -o gitee ./cmd/gitee
sudo mv gitee /usr/local/bin/
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

**Homebrew（macOS / Linux）**

```bash
brew install gitee/tap/gitee
```

**バイナリダウンロード**

[Releases](https://gitee.com/gitee/gitee-cli/releases) からプラットフォーム 맞는 バイナリをダウンロード：

| プラットフォーム | ファイル名 |
|----------------|------------|
| Linux amd64 | `gitee-linux-amd64` |
| Linux arm64 | `gitee-linux-arm64` |
| macOS amd64 | `gitee-darwin-amd64` |
| macOS arm64（M1/M2） | `gitee-darwin-arm64` |
| Windows amd64 | `gitee-windows-amd64.exe` |

**ソースからビルド**

```bash
git clone https://gitee.com/gitee/gitee-cli.git
cd gitee-cli
go build -o gitee ./cmd/gitee
sudo mv gitee /usr/local/bin/
```

### 認証

```bash
gitee auth login
# ブラウザで認証を完了し、Token は自動的に ~/.gitee-cli/settings.json に保存
```

既存の Token を使用：

```bash
echo "your_access_token" | gitee auth login --with-token
```

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
gitee pr merge 18 --squash --delete-branch

# PR をレビュー
gitee pr review 18 --approve
gitee pr review 18 --request-changes --body "ユニットテストを追加してください"
```

### 検索

```bash
# Go 言語のリポジトリを検索、Stars 順でソート
gitee search repos "web framework" --language go --sort stars

# ユーザーを検索
gitee search users "john"

# Issue を検索
gitee search issues "メモリリーク" --state open
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

# レスポンスヘッダを表示
gitee api GET /user --include
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
| `--token string` | Access Token（設定をオーバーライド） | 設定から読み込み |
| `--hostname string` | Gitee インスタンスアドレス（企業用） | `gitee.com` |
| `-o, --output string` | 出力形式：`table` \| `json` \| `csv` | `table` |
| `-P, --paginate` | 自動ページネーション、全結果を取得 | `false` |
| `--limit int` | 結果の最大数 | `0`（制限なし） |
| `--page int` | ページ番号 | `1` |
| `--per-page int` | 1 ページあたりの件数（最大 100） | `30` |
| `--verbose` | HTTP リクエスト/レスポンスの詳細を表示 | `false` |
| `--no-color` | ANSI カラーを無効化 | `false` |
| `--no-interaction` | サイレントモード（CI/CD 用） | `false` |
| `--lang string` | UI 言語：`zh` \| `en` | `zh` |
| `--timeout int` | ネットワークタイムアウト秒数 | `30` |

**環境変数：**

| 変数 | 説明 |
|------|------|
| `GITEE_TOKEN` | Access Token |
| `GITEE_HOSTNAME` | Gitee インスタンスアドレス |
| `GITEE_OUTPUT` | デフォルト出力形式 |
| `GITEE_API_URL` | API ルート URL |
| `HTTPS_PROXY` | HTTPS プロキシ |
| `NO_COLOR` | カラーを無効化（標準） |

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

[`documents/developer-guide.md`](documents/developer-guide.md) を参照

```bash
# クローン
git clone https://gitee.com/gitee/gitee-cli.git && cd gitee-cli

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
| [`documents/FSD.md`](documents/FSD.md) | 機能仕様書 |
| [`documents/SDD.md`](documents/SDD.md) | システム設計書 |
| [`documents/developer-guide.md`](documents/developer-guide.md) | 開発者ガイド |

---

## 参考資料

- [Gitee API v5 ドキュメント](https://gitee.com/api/v5/swagger)
- [Gitee OAuth2 ドキュメント](https://gitee.com/api/v5/oauth_doc)
- [GitHub gh CLI](https://github.com/cli/cli)（デザイン参照）
- [GitLab glab CLI](https://github.com/gitlab-org/cli)（デザイン参照）

---

## License

[Apache 2.0](LICENSE)

---

## Buy Me Coffee

このプロジェクトが役に立ったなら、コーヒーを請いたいです：

![Buy Me Coffee](./etc/BuyMeCoffe.jpg)