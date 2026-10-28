# gitee-cli 功能规格文档（Functional Specification Document）

> ⚠️ **状态提示（v0.13.2，2026-10 维护轮）**：本文档是项目早期的设计稿，部分内容**未实现或已过期**。
> 当前行为以代码、`gitee --help`、`gitee <cmd> --agent-help`、`gitee help reference`
> 和 `gitee help environment` 为准。与现状差异最大的段落已在文中直接标注/修正。


版本：1.0.0（设计稿；现状修订见页首横幅）| 语言：Go 1.27+ | 参考：Gitee API v5 + GitHub `gh` CLI

---

## 1. 项目概述

### 1.1 项目信息

| 字段 | 值 |
|------|-----|
| 项目名称 | `gitee-cli` |
| 类型 | 命令行工具（CLI） |
| 语言 | Go 1.27+ |
| 框架 | Cobra（配置为手写 JSON，未使用 Viper） |
| 目标 | 成为 Gitee 的官方 CLI 工具，功能对标 GitHub `gh` |
| API 基址 | `https://gitee.com/api/v5` |
| 参考项目 | [`gh`](https://github.com/cli/cli)、[`glab`](https://github.com/cli/go-gitlab) |

### 1.2 核心功能摘要

```
gitee auth login|logout|status|refresh|token        # PAT 认证（无 OAuth2 设备流）
gitee repo list|create|view|fork|edit|delete|clone|browse|branches  # 仓库管理
gitee repo tag|collaborator|star|watch|file|commits|compare|protect|archive|rename|sync  # 仓库扩展
gitee issue list|create|view|edit|close|reopen|comment  # Issue 管理
gitee pr list|create|view|merge|review|diff|checks|comments|checkout|close|reopen|edit|ready|status  # Pull Request
gitee gist list|create|view|edit|delete|star|clone|comments  # 代码片段
gitee org list|view|members|repos|issues             # 组织管理（子命令为复数）
gitee search issues|users                            # 搜索（repos 返回空；code/commits 该实例不支持）
gitee api <method> <path> [--field key=val]          # 原始 API 访问
gitee config set|get|list|unset                      # 配置管理
gitee completion bash|zsh|fish|powershell            # Shell 补全
gitee label|milestone|release|webhook|notification|ssh-key|user|alias|extension|agent|status  # 其余命令组
```

当前命令面以 `gitee help reference` 为准（每组命令的 help 与 `--agent-help` 是唯一权威）。

---

## 2. 全局选项（Global Flags）

| 标记 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `--repo string` | string | 目标仓库 owner/repo，缺省从 git remote 推断 | 自动 |
| `--hostname string` | string | Gitee 服务器地址（企业私有部署） | `gitee.com` |
| `--token string` | string | 访问令牌（PAT，优先于配置文件） | 读取配置 |
| `--api-url string` | string | API 根地址 | `https://{hostname}/api/v5` |
| `-o, --output string` | string | 输出格式 `table`\|`json`（**不支持 csv，非法值报错退出**） | `table` |
| `-P, --paginate` | bool | 自动分页遍历所有结果 | `false` |
| `--page int` | int | 指定页码 | `1` |
| `--per-page int` | int | 每页数量（最大 100） | `30` |
| `--limit int` | int | 结果总数上限（`--paginate` 时为跨页总数；`0` 表示不限） | `30` |
| `--timeout int` | int | 网络超时秒数 | `30` |
| `-v, --verbose` | bool | 输出调试信息（HTTP 请求/响应详情） | `false` |
| `--no-color` | bool | 关闭 ANSI 颜色输出 | `false` |
| `--no-extensions` | bool | 跳过外部扩展加载 | `false` |
| `--agent-help` | bool | 输出 Agent 向长文档（部分命令支持，`-o json` 可结构化） | `false` |

> 旧设计稿中的 `--config` / `--no-interaction` / `--insecure` / `--lang` **不存在**；
> 配置文件路径固定为 `~/.gitee-cli/settings.json`。

**环境变量（13 个，完整说明见 `gitee help environment`）：**

| 环境变量 | 对应/说明 |
|---------|----------|
| `GITEE_TOKEN` | 访问令牌，等价 `--token` |
| `GITEE_HOSTNAME` | 等价 `--hostname` |
| `GITEE_API_URL` | 等价 `--api-url` |
| `GITEE_OUTPUT` | 默认输出格式，等价 `--output`；非 `table`/`json` 报错 |
| `GITEE_VERBOSE` | `1`/`true`/`yes`/`on` 启用 HTTP 详情（stderr），等价 `--verbose` |
| `GITEE_TIMEOUT` | HTTP 超时秒数，等价 `--timeout`；非整数报错 |
| `GITEE_LIMIT` | 默认 `--limit` 值 |
| `GITEE_PAGINATE` | `1`/`true`/`yes`/`on` 默认开启 `--paginate` |
| `GITEE_NO_EXTENSIONS` | `1`/`true`/`yes`/`on` 跳过扩展加载，等价 `--no-extensions` |
| `NO_COLOR` | 非空关闭颜色，等价 `--no-color` |
| `GITEE_ALIASES_FILE` | 覆盖别名文件路径 |
| `GITEE_EXT_DIR` | 覆盖扩展目录 |
| `BROWSER` | `repo browse` 使用的浏览器 |

> 代理支持来自 Go `net/http` 的标准变量（`HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY`），CLI 不单独解析。

---

## 3. 命令详细规格

### 3.1 `gitee auth` — 认证管理

**实现方式：个人访问令牌（PAT）。** gitee-cli **没有** OAuth2 设备流、没有浏览器授权、
没有 `refresh_token` 交换。`auth login` 读取已有令牌并用 `GET /user` 校验。

**令牌解析优先级：** `--token` flag > `GITEE_TOKEN` 环境变量 > `~/.gitee-cli/settings.json`。

| 子命令 | 行为 |
|--------|------|
| `gitee auth login [--token <PAT>]` | 无参：读取已有令牌并校验 `/user`；带 `--token`：校验通过后写入 `~/.gitee-cli/settings.json`（目录 0700 / 文件 0600） |
| `gitee auth logout` | 删除本地存储的令牌 |
| `gitee auth status` | `GET /user`，显示登录状态与用户名 |
| `gitee auth refresh` | 占位实现：打印「Token refresh is not yet supported. Please re-authenticate with 'gitee auth login'.」后正常退出（退出码 0）；不刷新、不重新校验，**不是** OAuth `refresh_token` 流程 |
| `gitee auth token` | 打印当前访问令牌（供管道使用） |

**`gitee auth login` 真实流程：**
```
1. 按优先级解析候选令牌（--token > GITEE_TOKEN > settings.json）
2. GET /user，header `Authorization: token <PAT>` → 校验并取用户名
3. 仅在显式传 --token 时，把令牌写入 ~/.gitee-cli/settings.json（0600）
```
> ⚠️ `--token` 的值会出现在 `ps` 与 shell history；一次性脚本之外请用 `GITEE_TOKEN`。

**`gitee auth login` 参数：**

| 参数 | 说明 |
|------|------|
| `-t, --token string` | 指定令牌并保存（校验通过后写入 settings.json） |

> 旧设计稿里的 `--with-token`（从 stdin 读取）与 `--scopes` **不存在**。

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
| `--sort string` | `created`\|`updated`\|`pushed`\|`full_name`\|`stars` | `full_name` |
| `--direction string` | `asc`\|`desc` | `asc` |
| `-L, --limit int` | 返回条数上限 | `30` |

> 旧稿的 `--type` 不存在；该命令不支持位置参数（仓库名用 `--repo` 或从 git remote 推断）。

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
| `LABELS` | `labels[].name` | 标签（逗号连接；无标签显示 `-`） |
| `UPDATED` | `updated_at` | 更新日期（`YYYY-MM-DD`） |

> 表格只有这 5 列，没有 `ASSIGNEE` 列。

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
| `--state string` | `state` | `open`（`open`\|`closed`\|`merged`\|`all`） |
| `--head string` | `head` | — |
| `--base string` | `base` | — |

> 旧稿的 `--author` / `--sort` / `--direction` 在 `pr list` 上不存在。

**`gitee pr merge` 参数：**

| 参数 | 说明 |
|------|------|
| `--squash` | Squash 合并（body `squash=true`） |
| `--squash-message string` | Squash 合并的提交信息（body `squash_message`） |

> 旧设计稿里的 `--rebase` / `--merge-commit-message` / `--delete-branch` **不存在**：
> CLI 只实现 squash 与 squash-message 两个合并选项。

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
| `--file stringArray` | `files` | `filename=content` 或 `filename=@路径`，可重复 |
| `--filename string` | — | stdin 模式下的文件名（内容从 stdin 读） |
| `-i, --interactive` | — | 交互式编辑模式 |

> 没有 `--from-stdin`；从 stdin 输入时用 `--filename <名字>`（可配 `--file`）。

**`gitee gist list` 输出字段：**

| 字段 | API 字段 | 说明 |
|------|---------|------|
| `ID` | `id` | Gist ID |
| `DESCRIPTION` | `description` | 描述（截断 50 字符、空显示 `-`） |
| `PUBLIC` | `public` | 是否公开 |
| `FILES` | `files` 数量 | 包含文件数 |
| `UPDATED` | `updated_at` | 更新时间 |

---

### 3.6 `gitee org` — 组织管理

**对应 Gitee API：** `/orgs`, `/orgs/{org}`

| 子命令 | Gitee API | 说明 |
|--------|-----------|------|
| `gitee org list` | `GET /user/orgs` | 列出用户所属的组织 |
| `gitee org view <org>` | `GET /orgs/{org}` | 查看组织详情 |
| `gitee org members <org>` | `GET /orgs/{org}/members` | 列出组织成员 |
| `gitee org repos <org>` | `GET /orgs/{org}/repos` | 列出组织的仓库 |
| `gitee org issues <org>` | `GET /orgs/{org}/issues` | 列出组织内的 Issue |

---

### 3.7 `gitee search` — 搜索

**对应 Gitee API：** `/search/*`

| 子命令 | Gitee API | 说明 |
|--------|-----------|------|
| `gitee search repos <query>` | `GET /search/repositories` | 搜索仓库 |
| `gitee search issues <query>` | `GET /search/issues` | 搜索 Issue |
| `gitee search users <query>` | `GET /search/users` | 搜索用户 |
| `gitee search code <query>` | `GET /search/code` | 代码搜索（该 Gitee 实例返回 404，CLI 提示后非 0 退出） |
| `gitee search commits <query>` | `GET /search/commits` | 提交搜索（该 Gitee 实例返回 404，CLI 提示后非 0 退出） |

> ⚠️ 现状（v0.13.2 实测）：`search issues` / `search users` 正常；`search repos` 对任意查询返回空
> （上游接口行为变化，CLI 打印 `(no results)`）；`search code` / `search commits` 在该实例不可用。

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

### 4.3 CSV（**不支持**）

`-o/--output` 只接受 `table` 与 `json`；其他值（包括早期文档宣称的 `csv`）会报错退出，
不会静默回退成 table。需要表格数据时用 `-o json` + `jq`（示例见 `gitee help formatting`）。

### 4.4 分页输出

- `--paginate`：自动遍历所有分页，合并为单个结果集
- `--page N`：指定页码（`1` 起始）
- `--per-page N`：每页数量（最大 100）
- `--limit N`：限制总输出条数（优先于分页，`0` = 不限制）

---

## 5. 错误处理规格

| 场景 | Exit Code | stderr 输出（`Error: <operation>: <detail>` 形态） |
|------|-----------|------------|
| 未登录（本地检查，未发请求） | 1 | `not authenticated. Set GITEE_TOKEN or run 'gitee auth login'` |
| 网络错误 / 超时 | 1 | `<transport error>`（如 `context deadline exceeded`、`connection refused`） |
| API 返回 401 | 1 | `HTTP 401: <Gitee message>`；body 无 message 时回退 `HTTP 401: authentication failed: run 'gitee auth login' or set GITEE_TOKEN` |
| API 返回 403 | 1 | `HTTP 403: <Gitee message>`，无 message 时 `HTTP 403: permission denied` |
| API 返回 404 | 1 | `HTTP 404: <Gitee message>`，无 message 时 `HTTP 404: resource does not exist or you lack read access` |
| API 返回 422 | 1 | `HTTP 422: <Gitee message>`，无 message 时 `HTTP 422: validation failed` |
| API 返回 5xx | 1 | `HTTP 5xx: <Gitee message>`，无 message 时 `internal server error` / `service unavailable` |
| 参数错误 | 1 | `unknown flag: --x` / `unknown command "x" for "gitee ..."`（Cobra 统一输出，同样带 `Error: ` 前缀） |
| 操作被取消（交互确认拒绝） | 0 | 打印 `Cancelled`，不执行任何动作 |

---

## 6. 国际化（i18n）

- **未实现。** CLI 没有 `--lang`、没有语言包目录、没有运行时切换语言的机制；
  命令帮助与输出是固定文案。
- 多语言只存在于文档层：`README.md` / `README_en.md` / `README_ja.md` 与
  `docs/{zh,en,ja}/`。

---

## 7. 非功能性需求

| 需求 | 要求 |
|------|------|
| 启动时间 | < 100ms（`gitee --version` 冷启动） |
| 首次编译时间 | < 20s（Go 1.27 增量编译，视机器而定） |
| 网络超时 | 默认 30s（`--timeout` 可覆盖） |
| 重试策略 | **仅 GET**：5xx/网络错误重试 3 次（2s 起，最长 8s）；写操作不自动重试 |
| 用户名来源 | 优先读 `settings.json` 缓存的 username，缺失时回退 `GET /user`（无 TTL 缓存） |
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
| Webhook | 6 | 已实现（create / list / view / edit / delete / ping） |
| 企业（Enterprise） | 20+ | 通过 `--hostname` / `GITEE_HOSTNAME` 支持私有部署 |
