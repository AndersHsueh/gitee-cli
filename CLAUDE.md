# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in this repository.

## 项目概述

**`gitee-cli`** — Gitee 官方 CLI 工具，功能对标 GitHub `gh`。用于在终端操作 Gitee 仓库、Issue、PR、代码片段等。

技术选型：**Go 1.23+**，参考项目为 [`gh`](https://github.com/cli/cli) 和 [`glab`](https://github.com/cli/go-gitlab)。

详细设计文档位于 `doc/` 目录：
- `doc/FSD.md` — 功能规格文档
- `doc/SDD.md` — 系统设计文档
- `doc/developer-guide.md` — 开发指导文档

## 技术栈

| 组件 | 选型 | 说明 |
|------|------|------|
| 语言 | Go 1.23+ | 交叉编译简单，社区成熟 |
| CLI 框架 | Cobra + Viper | gh/glab 均使用此组合 |
| HTTP 客户端 | `net/http` + `resty` | Gitee API v5 客户端 |
| 认证 | `golang.org/x/oauth2` | OAuth2 设备码流程 |

## 常用命令

```bash
# 构建
go build -o gitee ./cmd/gitee

# 测试
go test ./... -v -count=1

# Lint
golangci-lint run ./...

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o dist/gitee-linux-amd64 ./cmd/gitee

# 安装依赖
go mod tidy
```

## 项目结构

```
cmd/           # Cobra 命令（root + auth/repo/issue/pr/gist/org/search/config）
internal/      # 内部实现
  api/         # Gitee API v5 客户端（client.go, endpoints.go, errors.go, pagination.go）
  auth/        # OAuth2 认证（token.go, oauth.go, config.go）
  config/      # Viper 配置封装
  printer/     # 输出格式化（table/json/csv）
  terminal/    # 终端交互（spinner, prompt, browser）
  i18n/        # 国际化（zh_CN, en_US）
pkg/types/     # Gitee API 数据模型（repo/issue/pr/user 等）
doc/           # 设计文档（FSD, SDD, developer-guide）
```

## 开发约定

- 所有命令子模块在 `cmd/` 下按功能组织（`cmd/repo/list.go` 等）
- `internal/api` 对外暴露 `Client` 接口，支持 `Do/Get/Post/Delete` 方法
- `pkg/types/` 定义所有 API 响应结构体，对标 Gitee API v5 JSON 字段
- Token 凭证存储在 `~/.config/gitee/credentials.yml`，文件权限 `0600`
- Commit 规范使用 Conventional Commits（`feat/`, `fix/`, `docs/`, `test/`）

## Git 远程与认证

- **Git push/pull** 使用 **SSH 公钥**认证（`~/.ssh/id_ed25519`），远程地址：
  ```
  origin  git@gitee.com:andershsueh/gitee-cli.git
  ```
- **Gitee OpenAPI**（程序内部 API 调用）使用私人令牌，见 `~/ai-info/gitee-auth-info.md`
- 不要用 HTTPS + token 方式推送 —— token 会返回 403，SSH 模式已经验证通过

## Gitee API v5 参考

- Swagger 文档：https://gitee.com/api/v5/swagger
- OAuth 文档：https://gitee.com/api/v5/oauth_doc
- 主要端点分类：Repos（75+）、Issues（23）、PRs（27）、Gists（17）、Orgs（14）、Users（18）、Search（3）、Webhooks（6）