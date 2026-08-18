# gitee-cli

> Gitee 公式 CLI ツール — ターミナルで Gitee の操作を完結、GitHub の `gh` に相当

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.13.0-orange.svg)](https://gitee.com/andershsueh/gitee-cli/releases/latest)

---

## 機能

```
gitee auth        # 認証 (login / logout / status / token)               ✅ 実装済
gitee repo        # リポジトリ操作 (list / view / create / fork ...)      ✅ 実装済
gitee issue       # Issue 操作 (list / view / create / close / comment)   ✅ 実装済
gitee pr          # PR 操作 (list / view / create / merge / checkout)     ✅ 実装済
gitee gist        # コードスニペット操作                                  ✅ 実装済
gitee api         # Raw API 呼び出し                                     ✅ 実装済
gitee search      # リポジトリ・コード・Issue・ユーザー検索                ✅ 実装済
gitee config      # 設定管理                                              ✅ 実装済
gitee completion  # シェル補完生成                                        ✅ 実装済
gitee version     # バージョン表示                                        ✅ 実装済
```

---

## インストール

```bash
# macOS (Homebrew)
brew install gitee-cli

# バイナリを直接ダウンロード
curl -L https://gitee.com/andershsueh/gitee-cli/releases/latest/download/gitee-darwin-amd64 -o /usr/local/bin/gitee
chmod +x /usr/local/bin/gitee
```

---

## クイックスタート

```bash
gitee auth login                                          # 認証
gitee repo list andershsueh                               # リポジトリ一覧
gitee issue list andershsueh/gitee-cli --state open        # Issue 一覧
gitee pr list andershsueh/gitee-cli --state open           # PR 一覧
gitee search repos "web framework" --language go           # 検索
```

---

## 更新履歴

### v0.13.0 (2026-08-19) — Runtime コンテキストリファクタ + アーキテクチャ収束

**🧱 アーキテクチャリファクタ**
- `cmd.Global*` グローバル変数を排除し、`cmd.Runtime` 明示的コンテキストを注入
- 110 のコマンドファイルを移行: `NewAPIClient()` → `NewAPIClientFrom(rt)` など
- `internal/handler` 層の倒置を解消、ビジネスロジックを `internal/domain` に移動
- リストコマンド実行スケルトンを抽象化、全 API list コマンドを移行
- リポジトリ解析エントリを統合、冗長な `ResolveRepoFromFlag` を削除

**🐛 修正**
- `issue comment` がロックされた Issue で 401 認証エラーを誤検出する問題を修正

**📖 ドキュメント**
- 11 次元スキーマで 129 件の agent-help ドキュメントを補完 + 契約テスト
- コマンドツリー全体を網羅する completion/version 信号 + 契約テスト

---

## 開発

```bash
git clone https://gitee.com/andershsueh/gitee-cli.git && cd gitee-cli
go build -o bin/gitee ./cmd/gitee
go test ./... -race -count=1
```

---

## ライセンス

[Mulan Permissive Software License v2](LICENSE)