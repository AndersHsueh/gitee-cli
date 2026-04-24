# gitee-cli 功能规格文档（Functional Specification Document）

版本：1.0.0 | 语言：Go 1.23+ | 参考：Gitee API v5 + GitHub `gh` CLI

---

## 1. 项目概述

### 1.1 项目信息

| 字段 | 值 |
|------|-----|
| 项目名称 | `gitee-cli` |
| 类型 | 命令行工具（CLI） |
| 语言 | Go 1.23+ |
| 框架 | Cobra + Viper |
| 目标 | 成为 Gitee 的官方 CLI 工具，功能对标 GitHub `gh` |
| API 基址 | `https://gitee.com/api/v5` |
| 参考项目 | [`gh`](https://github.com/cli/cli)、[`glab`](https://github.com/cli/go-gitlab) |

### 1.2 核心功能摘要

```
gitee auth login|logout|status|refresh|token   # OAuth2 认证
gitee repo list|create|clone|fork|view|edit|delete|browse  # 仓库管理
gitee issue list|create|view|edit|close|reopen|comment     # Issue 管理
gitee pr list|create|view|merge|close|reopen|review|diff|checks  # Pull Request
gitee gist list|create|view|edit|delete|star|clone         # 代码片段
gitee org list|view|member|repo|issue                      # 组织管理
gitee search repos|issues|users                            # 搜索
gitee api <method> <path> [--field key=val]                # 原始 API 访问
gitee config set|get|list|unset                            # 配置管理
gitee completion bash|zsh|fish|powershell                  # Shell 补全
```

---

## 2. 全局选项（Global Flags）

| 标记 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `--hostname string` | string | Gitee 服务器地址（企业私有部署） | `gitee.com` |
| `--token string` | string | OAuth Access Token（优先于配置文件） | 读取配置 |
| `-o, --output string` | string | 输出格式 (`table`\|`json`\|`csv`) | `table` |
| `-P, --paginate` | bool | 自动分页遍历所有结果 | `false` |
| `--page int` | int | 指定页码 | `1` |
| `--per-page int` | int | 每页数量（最大 100） | `30` |
| `--limit int` | int | 结果总数上限（0 = 不限制） | `0` |
| `-c, --config string` | string | 配置文件路径 | `~/.gitee-cli/settings.json` |
| `--api-url string` | string | API 根地址 | `https://gitee.com/api/v5` |
| `--timeout int` | int | 网络超时秒数 | `30` |
| `-v, --verbose` | bool | 输出调试信息（HTTP 请求/响应详情） | `false` |
| `--no-color` | bool | 关闭 ANSI 颜色输出 | `false` |
| `--no-interaction` | bool | 静默模式，禁止交互提示（CI/CD 场景） | `false` |
| `--insecure` | bool | 允许不安全的 HTTPS 连接 | `false` |
| `--lang string` | string | 界面语言 (`zh`\|`en`) | `zh` |

**环境变量映射（优先级低于 flag，高于配置文件）：**

| 环境变量 | 对应 Flag |
|---------|----------|
| `GITEE_TOKEN` | `--token` |
| `GITEE_HOSTNAME` | `--hostname` |
| `GITEE_OUTPUT` | `--output` |
| `GITEE_API_URL` | `--api-url` |
| `HTTPS_PROXY` / `HTTP_PROXY` | 代理设置 |
| `NO_COLOR` | `--no-color` |

---

## 3. 命令详细规格

### 3.1 `gitee auth` — 认证管理

**对应 Gitee API：** OAuth2 Device Flow（`/oauth/authorizations/device/*`）

| 子命令 | Gitee API 端点 | 说明 |
|--------|--------------|------|
| `gitee auth login` | `POST /oauth/authorizations/device/code` + 轮询 | 启动 OAuth2 设备流程，获取 Access Token |
| `gitee auth logout` | 本地操作 | 删除 `~/.gitee-cli/settings.json` 中的 Token |
| `gitee auth status` | `GET /user` | 显示当前登录状态（用户名、Token 有效期） |
| `gitee auth refresh` | `POST /oauth/token?grant_type=refresh_token` | 刷新过期 Token |
| `gitee auth token` | 本地操作 | 输出当前 Access Token（供管道使用） |

**`gitee auth login` 完整流程：**
```
1. POST /oauth/authorizations/device/code
   Body: { client_id, scope }
   → device_code, user_code, verification_uri, expires_in, interval

2. 显示提示：
   ! First copy your one-time code: XXXX-XXXX
   - Press Enter to open gitee.com/device in your browser

3. 打开浏览器 / 或在 --no-interaction 模式下打印 URL

4. 轮询 POST /oauth/authorizations/device/token
   Body: { client_id, device_code, grant_type="urn:ietf:params:oauth:grant-type:device_code" }
   → access_token, token_type, scope, refresh_token

5. 将 access_token 写入 ~/.gitee-cli/settings.json（0600 权限）

6. GET /user → 验证并显示 "✓ Logged in as <username>"
```

**`gitee auth login` 参数：**

| 参数 | 说明 |
|------|------|
| `--scopes string` | 请求的 OAuth 权限范围（默认：`user_info,projects,pull_requests,issues,notes,gists`） |
| `--with-token` | 从 stdin 读取已有 Token 并保存（跳过 OAuth 流程） |

---

### 3.2 `gitee repo` — 仓库管理

**对应 Gitee API：** `/repos`, `/user/repos`, `/orgs/{org}/repos`

| 子命令 | Gitee API | 说明 |
|--------|-----------|------|
| `gitee repo list` | `GET /user/repos` | 列出当前用户的仓库 |
| `gitee repo create` | `POST /user/repos` | 创建新仓库 |
| `gitee repo clone <repo>` | 调用 `git clone` | Clone 仓库到本地 |
| `gitee repo fork <repo>` | `POST /repos/{owner}/{repo}/forks` | Fork 仓库 |
| `gitee repo view <repo>` | `GET /repos/{owner}/{repo}` | 查看仓库详情 |
| `gitee repo edit <repo>` | `PATCH /repos/{owner}/{repo}` | 编辑仓库设置 |
| `gitee repo delete <repo>` | `DELETE /repos/{owner}/{repo}` | 删除仓库（需 `--yes` 确认） |
| `gitee repo browse <repo>` | 打开浏览器 | 在浏览器打开仓库页面 |

**`gitee repo list` 输出字段：**

| 字段 | API 字段 | 说明 |
|------|---------|------|
| `NAME` | `full_name` | owner/repo 格式 |
| `VISIBILITY` | `private` + `internal` | public/private/internal |
| `LANGUAGE` | `language` | 主要编程语言 |
| `STARS` | `stargazers_count` | 收藏数 |
| `FORKS` | `forks_count` | Fork 数 |
| `UPDATED` | `updated_at` | 最后更新时间（相对时间） |

**`gitee repo list` 参数：**

| 参数 | 说明 | 默认 |
|------|------|------|
| `--owner string` | 指定用户/组织名 | 当前登录用户 |
| `--visibility string` | `public`\|`private`\|`internal`\|`all` | `all` |
| `--sort string` | `created`\|`updated`\|`pushed`\|`full_name` | `updated` |
| `--direction string` | `asc`\|`desc` | `desc` |
| `--type string` | `owner`\|`member`\|`all` | `owner` |

**`gitee repo create` 参数：**

| 参数 | 必填 | API 字段 | 说明 |
|------|------|---------|------|
| `--name string` | ✅ | `name` | 仓库名 |
| `--description string` | ❌ | `description` | 仓库描述 |
| `--private` | ❌ | `private=true` | 设为私有 |
| `--auto-init` | ❌ | `auto_init=true` | 初始化 README（默认 true） |
| `--gitignore string` | ❌ | `gitignore_template` | .gitignore 模板名 |
| `--license string` | ❌ | `license_template` | 许可证模板名 |
| `--org string` | ❌ | — | 在组织下创建（使用 `POST /orgs/{org}/repos`） |

---

### 3.3 `gitee issue` — Issue 管理

**对应 Gitee API：** `/repos/{owner}/{repo}/issues`

| 子命令 | Gitee API | 说明 |
|--------|-----------|------|
| `gitee issue list` | `GET /repos/{owner}/{repo}/issues` | 列出 Issue |
| `gitee issue create` | `POST /repos/{owner}/{repo}/issues` | 创建 Issue |
| `gitee issue view <number>` | `GET /repos/{owner}/{repo}/issues/{number}` | 查看详情 |
| `gitee issue edit <number>` | `PATCH /repos/{owner}/{repo}/issues/{number}` | 编辑 Issue |
| `gitee issue close <number>` | `PATCH /repos/{owner}/{repo}/issues/{number}` body `state=closed` | 关闭 Issue |
| `gitee issue reopen <number>` | `PATCH /repos/{owner}/{repo}/issues/{number}` body `state=open` | 重开 Issue |
| `gitee issue comment <number>` | `POST /repos/{owner}/{repo}/issues/{number}/comments` | 发表评论 |
| `gitee issue label list` | `GET /repos/{owner}/{repo}/labels` | 列出仓库标签 |

**`gitee issue list` 参数：**

| 参数 | API 字段 | 默认 |
|------|---------|------|
| `--state string` | `state` | `open` |
| `--labels string` | `labels`（逗号分隔） | — |
| `--assignee string` | `assignee` | — |
| `--milestone string` | `milestone` | — |
| `--repo string` | `{owner}/{repo}` | 从 git remote 检测 |

**`gitee issue list` 输出字段：**

| 字段 | API 字段 | 说明 |
|------|---------|------|
| `#` | `number` | Issue 编号 |
| `TITLE` | `title` | 标题（截断至 60 字符） |
| `STATE` | `state` | open/closed |
| `LABELS` | `labels[].name` | 标签（最多显示 3 个） |
| `ASSIGNEE` | `assignee.login` | 指派人 |
| `UPDATED` | `updated_at` | 更新日期（相对时间） |

**`gitee issue create` 参数：**

| 参数 | API 字段 | 必填 |
|------|---------|------|
| `--title string` | `title` | ✅ |
| `--body string` | `body` | ❌ |
| `--labels string` | `labels`（逗号分隔） | ❌ |
| `--assignees string` | `assignees`（逗号分隔） | ❌ |
| `--milestone int` | `milestone_number` | ❌ |
| `--repo string` | `{owner}/{repo}` | 从 git remote 检测 |

---

### 3.4 `gitee pr` — Pull Request 管理

**对应 Gitee API：** `/repos/{owner}/{repo}/pulls`

| 子命令 | Gitee API | 说明 |
|--------|-----------|------|
| `gitee pr list` | `GET /repos/{owner}/{repo}/pulls` | 列出 PR |
| `gitee pr create` | `POST /repos/{owner}/{repo}/pulls` | 创建 PR |
| `gitee pr view <number>` | `GET /repos/{owner}/{repo}/pulls/{number}` | 查看 PR 详情 |
| `gitee pr merge <number>` | `PUT /repos/{owner}/{repo}/pulls/{number}/merge` | 合并 PR |
| `gitee pr close <number>` | `PATCH /repos/{owner}/{repo}/pulls/{number}` `state=closed` | 关闭 PR |
| `gitee pr reopen <number>` | `PATCH /repos/{owner}/{repo}/pulls/{number}` `state=open` | 重开 PR |
| `gitee pr review <number>` | `POST /repos/{owner}/{repo}/pulls/{number}/reviews` | 审查 PR |
| `gitee pr diff <number>` | `GET /repos/{owner}/{repo}/pulls/{number}` Accept: diff | 输出 diff 内容 |
| `gitee pr checks <number>` | `GET /repos/{owner}/{repo}/statuses/{sha}` | CI 检查状态 |

**`gitee pr list` 参数：**

| 参数 | API 字段 | 默认 |
|------|---------|------|
| `--state string` | `state` | `open` |
| `--author string` | `author` | — |
| `--head string` | `head` | — |
| `--base string` | `base` | — |
| `--sort string` | `sort` | `created` |
| `--direction string` | `direction` | `desc` |

**`gitee pr merge` 参数：**

| 参数 | API 字段 | 说明 |
|------|---------|------|
| `--squash` | `merge_method=squash` | Squash 合并 |
| `--rebase` | `merge_method=rebase` | Rebase 合并 |
| `--merge-commit-message string` | `merge_commit_message` | 合并提交信息 |
| `--delete-branch` | — | 合并后删除源分支 |

---

### 3.5 `gitee gist` — 代码片段

**对应 Gitee API：** `/gists`

| 子命令 | Gitee API | 说明 |
|--------|-----------|------|
| `gitee gist list` | `GET /gists` | 列出用户的 Gist |
| `gitee gist create [files...]` | `POST /gists` | 创建 Gist |
| `gitee gist view <id>` | `GET /gists/{id}` | 查看 Gist 内容 |
| `gitee gist edit <id>` | `PATCH /gists/{id}` | 编辑 Gist |
| `gitee gist delete <id>` | `DELETE /gists/{id}` | 删除 Gist |
| `gitee gist star <id>` | `PUT /gists/{id}/star` | Star Gist |
| `gitee gist clone <id>` | 调用 `git clone` | Clone Gist 为本地 Git 仓库 |

**`gitee gist create` 参数：**

| 参数 | API 字段 | 说明 |
|------|---------|------|
| `--description string` | `description` | Gist 描述 |
| `--public` | `public=true` | 设为公开（默认 private） |
| `--from-stdin` | — | 从 stdin 读取内容（配合文件名 `--filename`） |
| `--filename string` | — | stdin 模式下的文件名 |

**`gitee gist list` 输出字段：**

| 字段 | API 字段 | 说明 |
|------|---------|------|
| `ID` | `id` | Gist ID |
| `DESCRIPTION` | `description` | 描述（截断 50 字符） |
| `FILES` | `files` 数量 | 包含文件数 |
| `PUBLIC` | `public` | 是否公开 |
| `UPDATED` | `updated_at` | 更新时间 |

---

### 3.6 `gitee org` — 组织管理

**对应 Gitee API：** `/orgs`, `/orgs/{org}`

| 子命令 | Gitee API | 说明 |
|--------|-----------|------|
| `gitee org list` | `GET /user/orgs` | 列出用户所属的组织 |
| `gitee org view <org>` | `GET /orgs/{org}` | 查看组织详情 |
| `gitee org member list <org>` | `GET /orgs/{org}/members` | 列出组织成员 |
| `gitee org repo list <org>` | `GET /orgs/{org}/repos` | 列出组织的仓库 |
| `gitee org issue list <org>` | `GET /orgs/{org}/issues` | 列出组织内的 Issue |

---

### 3.7 `gitee search` — 搜索

**对应 Gitee API：** `/search/*`

| 子命令 | Gitee API | 说明 |
|--------|-----------|------|
| `gitee search repos <query>` | `GET /search/repositories` | 搜索仓库 |
| `gitee search issues <query>` | `GET /search/issues` | 搜索 Issue |
| `gitee search users <query>` | `GET /search/users` | 搜索用户 |

**`gitee search repos` 参数：**

| 参数 | API 字段 | 说明 |
|------|---------|------|
| `--language string` | `language` | 按编程语言过滤 |
| `--sort string` | `sort` | `stars`\|`forks`\|`updated` |
| `--order string` | `order` | `asc`\|`desc`（默认 `desc`） |
| `--limit int` | `per_page` | 返回条数上限 |

---

### 3.8 `gitee api` — 原始 API 访问

直接执行 Gitee API v5 请求，方便调试和自动化脚本：

```bash
gitee api GET /user
gitee api GET /user/repos?type=owner
gitee api POST /user/repos --field name=my-repo --field private=true
gitee api DELETE /repos/owner/repo
gitee api PATCH /repos/owner/repo --field description="new desc"
```

**参数：**

| 参数 | 说明 |
|------|------|
| `<method>` | HTTP 方法：GET / POST / PUT / PATCH / DELETE |
| `<path>` | API 路径（如 `/user/repos`），自动拼接 `--api-url` 前缀 |
| `--field key=value` | 请求体参数（JSON），可多次使用 |
| `--header key=value` | 自定义请求头，可多次使用 |
| `--raw-field key=value` | 字符串类型参数（不做 JSON 类型推断） |
| `--input file` | 从文件读取请求体（JSON） |
| `--include` | 输出响应头 |
| `--paginate` | 自动分页并合并 JSON 数组 |

---

### 3.9 `gitee config` — 配置管理

| 子命令 | 说明 |
|--------|------|
| `gitee config list` | 列出所有配置（含来源：file/env/default） |
| `gitee config set <key> <value>` | 设置配置项（写入 `~/.gitee-cli/settings.json`） |
| `gitee config get <key>` | 读取指定配置项 |
| `gitee config unset <key>` | 删除配置项 |

**可配置项：**

| 配置键 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `protocol` | string | `https` | git 协议 (`ssh`\|`https`) |
| `editor` | string | `$EDITOR` | 提交消息/正文编辑器 |
| `browser` | string | 系统默认 | 浏览器打开命令 |
| `prompt` | string | `enabled` | 交互提示开关 (`enabled`\|`disabled`) |
| `output` | string | `table` | 默认输出格式 |
| `pager` | string | `$PAGER` | 分页查看命令 |

---

### 3.10 `gitee completion` — Shell 补全

| 子命令 | 说明 |
|--------|------|
| `gitee completion bash` | 输出 Bash 补全脚本 |
| `gitee completion zsh` | 输出 Zsh 补全脚本 |
| `gitee completion fish` | 输出 Fish 补全脚本 |
| `gitee completion powershell` | 输出 PowerShell 补全脚本 |

**安装示例：**
```bash
# Bash
gitee completion bash >> ~/.bashrc

# Zsh
gitee completion zsh > "${fpath[1]}/_gitee"

# Fish
gitee completion fish > ~/.config/fish/completions/gitee.fish
```

---

## 4. 输出格式规格

### 4.1 表格格式（默认 `--output table`）

- 使用等宽列对齐，UTF-8 编码
- 列宽自动适应终端宽度，最宽列截断为 60 字符（末尾 `...`）
- 表头大写加粗（ANSI `\033[1m`）
- 时间字段显示相对时间（如 `2 days ago`）

### 4.2 JSON 格式（`--output json`）

- 美化 JSON 输出（2 空格缩进）
- `list` 命令返回 JSON 数组
- `view` 命令返回单个 JSON 对象
- 错误信息输出到 stderr（不影响管道）

### 4.3 CSV 格式（`--output csv`）

- 第一行为表头（字段名小写）
- 值用双引号包裹，内部双引号用 `""` 转义
- 适合导入 Excel / 数据库

### 4.4 分页输出

- `--paginate`：自动遍历所有分页，合并为单个结果集
- `--page N`：指定页码（`1` 起始）
- `--per-page N`：每页数量（最大 100）
- `--limit N`：限制总输出条数（优先于分页，`0` = 不限制）

---

## 5. 错误处理规格

| 场景 | Exit Code | stderr 输出 |
|------|-----------|------------|
| 未登录 | 1 | `Error: not authenticated. Run 'gitee auth login' first.` |
| 网络超时 | 1 | `Error: failed to connect to gitee.com: <detail>` |
| API 返回 401 | 1 | `Error: authentication failed. Token may be expired. Run 'gitee auth refresh'.` |
| API 返回 403 | 1 | `Error: insufficient permissions: <message>` |
| API 返回 404 | 1 | `Error: not found: <resource>` |
| API 返回 422 | 1 | `Error: validation failed: <field errors>` |
| API 返回 5xx | 1 | `Error: Gitee server error (HTTP <code>): <message>` |
| 参数错误 | 2 | `Error: <description>` + 用法帮助 |
| Token 过期 | 1 | 提示 `gitee auth refresh` 或 `gitee auth login` |
| 操作被取消 | 1 | `Aborted.` |

---

## 6. 国际化（i18n）

- 默认语言：中文（`zh-CN`）
- 支持 `--lang en` 切换英文输出（或 `GITEE_LANG=en`）
- 语言包位于 `internal/i18n/` 目录（`zh_CN.go`, `en_US.go`）
- 错误消息、帮助文本、命令描述均支持国际化

---

## 7. 非功能性需求

| 需求 | 要求 |
|------|------|
| 启动时间 | < 100ms（`gitee --version` 冷启动） |
| 首次编译时间 | < 15s（Go 1.23 增量编译） |
| 网络超时 | 默认 30s（`--timeout` 可覆盖） |
| 重试策略 | 5xx 自动重试 3 次（指数退避，最大 8s） |
| Token 缓存 | 用户信息缓存 5 分钟（避免重复 `GET /user`） |
| 代理 | 遵循 `HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY` 环境变量 |
| 跨平台 | Linux amd64/arm64、macOS amd64/arm64（M1）、Windows amd64 |
| 二进制大小 | < 20MB（启用 `-ldflags="-s -w"` 压缩） |

---

## 8. Gitee API v5 端点覆盖矩阵

| 功能模块 | API 端点数 | gitee-cli 覆盖 |
|---------|-----------|--------------|
| 仓库（Repos） | 75+ | 8 核心端点 |
| Issue | 23 | 8 端点 |
| Pull Request | 27 | 9 端点 |
| Gist | 17 | 7 端点 |
| 组织（Org） | 14 | 5 端点 |
| 用户（User） | 18 | 通过 auth/search |
| 搜索（Search） | 3 | 3 端点（全覆盖） |
| Webhook | 6 | 暂不覆盖（v2.0 规划） |
| 企业（Enterprise） | 20+ | 暂不覆盖（v2.0 规划） |
