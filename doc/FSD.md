# gitee-cli 功能规格文档

## 1. 项目概述

### 1.1 项目信息

| 字段 | 值 |
|------|-----|
| 项目名称 | `gitee-cli` |
| 类型 | 命令行工具（CLI） |
| 语言 | Go 1.23+ |
| 框架 | Cobra + Viper |
| 目标 | 成为 Gitee 的官方 CLI 工具，功能对标 GitHub `gh` |
| 参考项目 | [`gh`](https://github.com/cli/cli)（GitHub CLI）、[`glab`](https://github.com/cli/go-gitlab)（GitLab CLI） |

### 1.2 核心功能摘要

```
gitee auth login|logout|status      # OAuth2 认证
gitee repo list|create|clone|fork|view  # 仓库管理
gitee issue list|create|view|edit|close  # Issue 管理
gitee pr list|create|view|merge|close    # Pull Request 管理
gitee gist list|create|view|edit         # 代码片段
gitee org list|member|teams              # 组织管理
gitee search repos|issues|users          # 搜索
gitee api <method> <path>                 # 原始 API 访问
gitee config set|get|list                 # 配置管理
```

---

## 2. 命令行接口（CLI）

### 2.1 全局选项（Global Flags）

| 标记 | 说明 | 默认值 |
|------|------|--------|
| `--hostname string` | Gitee 服务器地址 | `gitee.com` |
| `--token string` | OAuth Access Token | 读取配置 |
| `-o, --output string` | 输出格式 (`table`\|`json`\|`csv`) | `table` |
| `-P, --paginate` | 自动分页遍历所有结果 | `false` |
| `--page int` | 指定页码 | `1` |
| `--per-page int` | 每页数量 | `30` |
| `-c, --config string` | 配置文件路径 | `~/.gitee-cli/settings.json` |
| `--api-url string` | API 根地址 | `https://gitee.com/api/v5` |
| `-v, --verbose` | 输出调试信息 | `false` |

### 2.2 配色与交互

- 默认使用终端真彩色（True Color）
- 支持 `--no-color` 关闭 ANSI 颜色输出
- 支持 `--no-interaction` 静默模式（CI/CD 场景）

---

## 3. 命令详细规格

### 3.1 `gitee auth` — 认证管理

| 子命令 | 说明 |
|--------|------|
| `gitee auth login` | 启动 OAuth2 设备流程，获取 Access Token 并保存到 `~/.gitee-cli/settings.json` |
| `gitee auth logout` | 删除本地保存的 Token |
| `gitee auth status` | 显示当前登录状态（用户名、Token 有效期、企业） |
| `gitee auth refresh` | 刷新过期 Token |
| `gitee auth token` | 输出当前 Token（供其他工具使用） |

**`gitee auth login` 行为：**
1. 打印设备码和 URL：`https://gitee.com/device`
2. 打开默认浏览器（或输出手动操作说明）
3. 轮询 Token 直到授权完成
4. 将 Token 写入 `~/.gitee-cli/settings.json`（0600 权限）
5. 验证 Token 有效性，输出登录用户名

---

### 3.2 `gitee repo` — 仓库管理

| 子命令 | 说明 |
|--------|------|
| `gitee repo list` | 列出当前用户的仓库（支持 `--owner`, `--visibility public|private|internal`） |
| `gitee repo create` | 创建新仓库（支持 `--description`, `--private`, `--gitignore`, `--license`, `--auto-init`） |
| `gitee repo clone <repo>` | Clone 仓库到本地（调用 `git clone`） |
| `gitee repo fork <repo>` | Fork 仓库到当前用户 |
| `gitee repo view <repo>` | 查看仓库详情（描述、语言、Stars、Forks、链接） |
| `gitee repo edit <repo>` | 编辑仓库设置（描述、私有状态等） |
| `gitee repo delete <repo>` | 删除仓库（需确认 `--yes`） |
| `gitee repo browse <repo>` | 在默认浏览器打开仓库页面 |

**`gitee repo list` 输出字段：**

| 字段 | 说明 |
|------|------|
| `NAME` | 仓库名（owner/name 格式） |
| `VISIBILITY` | public/private/internal |
| `LANGUAGE` | 主要语言 |
| `STARS` | 收藏数 |
| `FORKS` | Fork 数 |
| `UPDATED` | 最后更新时间 |

**`gitee repo create` 参数：**

| 参数 | 必填 | 说明 |
|------|------|------|
| `--name` | 是 | 仓库名 |
| `--description` | 否 | 仓库描述 |
| `--private` | 否 | 私有仓库（默认 false） |
| `--auto-init` | 否 | 初始化 README（默认 true） |
| `--gitignore` | 否 | .gitignore 模板名 |
| `--license` | 否 | 许可证模板名 |
| `--org` | 否 | 在指定组织下创建 |

---

### 3.3 `gitee issue` — Issue 管理

| 子命令 | 说明 |
|--------|------|
| `gitee issue list` | 列出 Issue（支持 `--state open|closed|all`, `--labels`, `--assignee`, `--milestone`） |
| `gitee issue create` | 创建 Issue（支持 `--title`, `--body`, `--labels`, `--assignees`, `--milestone`） |
| `gitee issue view <number>` | 查看 Issue 详情（Markdown body 渲染输出） |
| `gitee issue edit <number>` | 编辑 Issue（所有创建参数均支持） |
| `gitee issue close <number>` | 关闭 Issue |
| `gitee issue reopen <number>` | 重开 Issue |
| `gitee issue comment <number>` | 对 Issue 发表评论 |
| `gitee issue label list` | 列出仓库所有标签 |

**`gitee issue list` 输出字段：**

| 字段 | 说明 |
|------|------|
| `#` | Issue 编号 |
| `TITLE` | 标题 |
| `STATE` | open/closed |
| `LABELS` | 标签（逗号分隔，最多显示3个） |
| `ASSIGNEE` | 指派人 |
| `UPDATED` | 更新日期 |

---

### 3.4 `gitee pr` — Pull Request 管理

| 子命令 | 说明 |
|--------|------|
| `gitee pr list` | 列出 PR（支持 `--state open|closed|merged|all`） |
| `gitee pr create` | 创建 PR（`--title`, `--body`, `--source`, `--target`, `--draft`） |
| `gitee pr view <number>` | 查看 PR 详情（含 diff 统计） |
| `gitee pr merge <number>` | 合并 PR（`--squash`, `--delete-branch`） |
| `gitee pr close <number>` | 关闭 PR |
| `gitee pr reopen <number>` | 重开 PR |
| `gitee pr review <number>` | 审查 PR（`--approve`, `--request-changes`, `--comment`） |
| `gitee pr diff <number>` | 输出 PR 的 diff 内容 |
| `gitee pr checks <number>` | 查看 PR 的 CI 检查状态 |

**`gitee pr list` 输出字段：**

| 字段 | 说明 |
|------|------|
| `#` | PR 编号 |
| `TITLE` | 标题 |
| `STATE` | open/closed/merged |
| `SOURCE` → `TARGET` | 源分支 → 目标分支 |
| `AUTHOR` | 作者 |
| `UPDATED` | 更新日期 |

---

### 3.5 `gitee gist` — 代码片段

| 子命令 | 说明 |
|--------|------|
| `gitee gist list` | 列出用户的 Gist |
| `gitee gist create [files...]` | 创建 Gist（文件参数，支持 `--public`, `--description`） |
| `gitee gist view <id>` | 查看 Gist 内容 |
| `gitee gist edit <id>` | 编辑 Gist |
| `gitee gist delete <id>` | 删除 Gist |
| `gitee gist star <id>` | Star Gist |
| `gitee gist clone <id>` | 将 Gist clone 为本地 Git 仓库 |

---

### 3.6 `gitee org` — 组织管理

| 子命令 | 说明 |
|--------|------|
| `gitee org list` | 列出用户所属的组织 |
| `gitee org view <org>` | 查看组织详情 |
| `gitee org member list <org>` | 列出组织成员 |
| `gitee org repo list <org>` | 列出组织的仓库 |
| `gitee org issue list <org>` | 列出组织内的 Issue |

---

### 3.7 `gitee search` — 搜索

| 子命令 | 说明 |
|--------|------|
| `gitee search repos <query>` | 搜索仓库（`--language`, `--sort stars`, `--order asc|desc`） |
| `gitee search issues <query>` | 搜索 Issue（`--repo`, `--state`, `--labels`） |
| `gitee search users <query>` | 搜索用户 |

---

### 3.8 `gitee api` — 原始 API 访问

| 用法 | 说明 |
|------|------|
| `gitee api GET /repos/{owner}/{repo}` | 执行 GET 请求并输出 JSON |
| `gitee api POST /repos /repos --field name=foo` | 执行 POST 请求，支持 `--field` 传参 |
| `gitee api --preview <method> <path>` | 以 Preview API 方式调用 |

示例：
```bash
gitee api GET /user
gitee api GET /user/repos?type=owner
gitee api POST /user/repos --field name=my-repo --field private=true
```

---

### 3.9 `gitee config` — 配置管理

| 子命令 | 说明 |
|--------|------|
| `gitee config list` | 列出当前所有配置 |
| `gitee config set <key> <value>` | 设置配置项 |
| `gitee config get <key>` | 读取指定配置项 |
| `gitee config unset <key>` | 删除配置项 |

配置项：
- `protocol` — git 协议 (`ssh`|`https`)，默认 `https`
- `editor` — 提交消息编辑器
- `browser` — 浏览器打开命令
- `prompt` — 交互式提示开关

---

### 3.10 `gitee completion` — Shell 补全

| 子命令 | 说明 |
|--------|------|
| `gitee completion bash` | 输出 Bash 补全脚本 |
| `gitee completion zsh` | 输出 Zsh 补全脚本 |
| `gitee completion fish` | 输出 Fish 补全脚本 |
| `gitee completion powershell` | 输出 PowerShell 补全脚本 |

---

## 4. 输出格式规格

### 4.1 表格格式（默认）

- 使用等宽列对齐，UTF-8 编码
- 列宽自动适应，最宽 40 字符截断
- 表头加粗

### 4.2 JSON 格式

- 美化 JSON 输出（4 空格缩进）
- 错误输出到 stderr

### 4.3 分页输出

- `--paginate` 自动遍历所有分页并合并输出
- `--limit N` 限制输出条数

---

## 5. 错误处理

| 场景 | 输出 |
|------|------|
| 未登录 | `Error: not authenticated. Run 'gitee auth login' first.` |
| 网络错误 | `Error: failed to connect to gitee.com: <detail>` |
| API 错误 | `Error: <API error message> (HTTP <code>)` |
| Token 过期 | 提示自动刷新或重新登录 |
| 参数错误 | 显示用法帮助和错误说明 |

---

## 6. 国际化（i18n）

- 默认语言：中文（`zh-CN`）
- 支持 `--lang en` 切换英文输出
- 语言包位于 `i18n/` 目录

---

## 7. 非功能性需求

| 需求 | 要求 |
|------|------|
| 启动时间 | < 100ms（不含子命令初始化） |
| 冷编译时间 | < 10s（Go 增量编译） |
| 网络超时 | 默认 30s，可通过 `--timeout` 配置 |
| 缓存 | Token 和用户信息缓存 5 分钟 |
| 代理 | 支持 `HTTPS_PROXY` 环境变量 |