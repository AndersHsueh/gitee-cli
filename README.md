# gitee-cli

> 在终端里完整操作 Gitee 的现代命令行工具，功能对标 GitHub `gh` ——
> 并且是**全球第一个**同时为人类和 AI Agent 设计帮助系统的开源 CLI。

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.12.2-orange.svg)](https://gitee.com/andershsueh/gitee-cli/releases/latest)
[![Agent Help](https://img.shields.io/badge/--agent--help-131%20verbs-8A2BE2.svg)](docs/philosophy.md)

```bash
gitee repo clone owner/repo && cd repo
gitee issue create --title "发现一个 Bug" --labels bug
gitee pr create --title "fix: ..." --source fix/x --target main
gitee pr merge 18 --squash --delete-branch
```

如果你用过 `gh`，你已经会用 `gitee`。如果你的代码是 AI Agent 写的——**这个工具是为它准备的**。

---

## 为什么选 gitee-cli

**1. 覆盖面完整。** 22 个命令组、150+ 子命令，从仓库、Issue、PR、Release 到
里程碑、通知、协作者、分支保护，Gitee 日常工作流全部终端化：

| 领域 | 命令组 |
|------|--------|
| 代码协作 | `repo` `pr` `issue` `milestone` `label` `release` `gist` |
| 仓库运维 | `webhook` `org` `user` `ssh-key` `notification` |
| 检索 | `search`（repos / issues / users / code / commits）`status`（跨仓看板） |
| 工具链 | `api`（裸 API）`alias` `extension` `config` `completion` `agent` |

其中 `repo` 一组就有 tag / collaborator / star / watch / file（直接读写仓库文件）/
commits / compare / protect（分支保护）等 20+ 个动词——**部分能力连 `gh` 都没有内建**，
比如 `gitee notification`（终端收通知）和 `gitee repo star`（给朋友的项目点星，就在命令行）。

**2. 为 AI Agent 而生。** 这是 gitee-cli 与所有其他 CLI 的分水岭，往下看。

---

## 🤖 Dual-surface CLI Design：同一份 CLI，长两张脸

> 📖 完整设计宣言：[docs/philosophy.md](docs/philosophy.md)

2026 年调用 CLI 的不再只是人类的手指，还有 Claude Code、Cursor、Cline 们的代码。
这两类用户对"帮助文档"的需求**结构上正交**：人要短、能扫；Agent 要全、可解析、有唯一正解。
传统 `--help` 想一份文档讨好两个物种，结果两边都不爽。

gitee-cli 的答案：**承认两类用户，长两张脸。**

| Surface | Flag | 给谁 | 设计目标 |
|---------|------|------|---------|
| Human | `--help` | 人 | 信息密度低，3 秒扫到要点 |
| Agent | `--agent-help` | AI Agent | 信息密度高，结构稳定，机器可解析（`-o json`） |

```bash
gitee pr create --agent-help            # Agent 手册：幂等性/副作用/错误恢复/编排提示
gitee pr create --agent-help -o json    # 机器可读版，直接 parse
gitee --agent-help                      # 全部命令的覆盖矩阵
gitee agent onboard                     # 一键为你的仓库生成 Agent 编排模板
```

每份 Agent 文档覆盖 **11 个维度**：Pre/Post-conditions、Idempotency、Atomicity、
Side effects、错误码与恢复、OAuth scope、Rate limit、竞态条件、
**Orchestration hints**（跑完这条命令下一步跑什么）、**Parsing hints**（stdout 怎么 parse）。

**当前覆盖：131 个 verb（v0.12.1 实测）**，并由 contract test 在 CI 层守护——
新命令不写 Agent 文档，测试直接红。此外我们保证三条"可达性法则"：
缺少必填参数时 `--agent-help` 照样能出文档、查文档不需要 token、
探测文档永远不会误执行真命令（`auth login --agent-help` 只会告诉你它是 operational 命令，
而不是突然开始登录流程）。

一个真实场景——Agent 修完 bug 提了 PR 就跑，Issue 永远 open 着？
`gitee pr create --agent-help` 里直接写着 ⚠️ 警告和可复制的闭环脚本：

```bash
PR_OUT=$(gitee pr create --repo owner/repo --title "fix: ..." \
  --source fix/x --target main --body "Closes #42")
PR_NUM=$(echo "$PR_OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')
gitee issue comment 42 --body "Fixed in PR #${PR_NUM}"   # 闭环！
```

**不是"建议"，是可执行的唯一正解。** Agent 不需要猜。

---

## 📦 安装

**方式一：下载二进制（不需要 Go 环境）**

前往 [Releases](https://gitee.com/andershsueh/gitee-cli/releases/latest) 下载对应平台：

| 平台 | 文件 |
|------|------|
| Linux x86_64 | `gitee-linux-amd64.tar.gz` |
| Linux ARM64（树莓派 / ARM 服务器） | `gitee-linux-arm64.tar.gz` |
| macOS Intel | `gitee-darwin-amd64.tar.gz` |
| macOS Apple Silicon（M 系列） | `gitee-darwin-arm64.tar.gz` |
| Windows 64-bit | `gitee-windows-amd64.exe.zip` |

```bash
sha256sum -c SHA256SUMS.txt   # 校验完整性（Windows: certutil -hashfile <file> SHA256）
```

**方式二：从源码构建**

```bash
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli && go install ./cmd/gitee
```

## 🔑 认证

```bash
gitee auth login          # 按提示完成授权，Token 保存到 ~/.gitee-cli/settings.json
```

也支持 `echo "your_token" | gitee auth login --with-token`、
环境变量 `export GITEE_TOKEN=...`，或手动编辑 `~/.gitee-cli/settings.json`
（参考 `settings.json.sample`）。

---

## 使用示例

### 仓库

```bash
gitee repo list --owner torvalds --visibility public
gitee repo create --name my-project --private
gitee repo fork owner/repo && gitee repo clone owner/repo
gitee repo star owner/repo                  # 点星支持
gitee repo watch owner/repo --releases-only # 只订阅发版
gitee repo file push ./notes.md docs/notes.md -m "docs: add notes"  # 不 clone 直接改文件
gitee repo commits --author someone --since 2026-06-01
gitee repo compare master...feature/x
gitee repo protect master                   # 分支保护
```

### Issue / 里程碑 / 通知

```bash
gitee issue list --state all --labels bug,enhancement
gitee issue create --title "发现一个 Bug" --body "复现步骤：..." --labels bug
gitee issue edit 42 --title "更准确的标题" --add-assignee someone
gitee issue close 42 && gitee issue comment 42 --body "已修复"

gitee milestone create --title "v1.0" --due-on 2026-09-30
gitee notification list --unread            # 终端里收 Gitee 通知
gitee notification read --all
```

### Pull Request

```bash
gitee pr create --title "feat: 新增搜索" --source feature/search --target main
gitee pr diff 18 && gitee pr checks 18
gitee pr review 18 --approve
gitee pr review 18 --request-changes --body "请补充单元测试"
gitee pr merge 18 --squash --delete-branch
gitee pr status                             # 我名下所有仓库的 PR 看板
```

### 搜索 / 裸 API

```bash
gitee search repos "web framework" --language go --sort stars
gitee search code "func main" --repo owner/repo
gitee api GET /user/repos?type=owner --paginate
gitee api POST /user/repos --field name=test --field private=true
```

---

## 全局选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `--repo string` | 目标仓库（owner/repo），缺省从 git remote 推断 | 自动 |
| `-o, --output string` | 输出格式：`table` \| `json` \| `csv` | `table` |
| `-P, --paginate` | 自动分页取全部结果 | `false` |
| `--limit int` | 结果总数上限 | `0`（不限） |
| `--token string` | Access Token（覆盖配置文件） | 读配置 |
| `--hostname string` | Gitee 实例地址（企业私有部署） | `gitee.com` |
| `--verbose` | 打印 HTTP 请求/响应详情 | `false` |
| `--no-color` | 关闭颜色输出 | `false` |

环境变量：`GITEE_TOKEN` / `GITEE_HOSTNAME` / `GITEE_OUTPUT` / `GITEE_API_URL` / `NO_COLOR`。
更多请看 `gitee help environment`、`gitee help formatting`、`gitee help exit-codes`。

## Shell 补全 / 企业部署

```bash
echo 'eval "$(gitee completion bash)"' >> ~/.bashrc     # bash（zsh/fish/powershell 同理）
gitee auth login --hostname your-company.gitee.com       # 企业私有实例
```

---

## 🛠 开发

```bash
go install ./cmd/gitee            # 构建并安装
go test ./... -race -count=1      # 测试：29 个包 · 97 个测试文件 · 363 个测试函数
golangci-lint run ./...           # Lint
```

| 文档 | 说明 |
|------|------|
| [docs/philosophy.md](docs/philosophy.md) | Dual-surface CLI Design 设计宣言 |
| [docs/zh/FSD.md](docs/zh/FSD.md) | 功能规格文档 |
| [docs/zh/SDD.md](docs/zh/SDD.md) | 系统设计文档 |
| [docs/zh/developer-guide.md](docs/zh/developer-guide.md) | 开发指导文档 |

参考项目：[gh](https://github.com/cli/cli) · [glab](https://gitlab.com/gitlab-org/cli) ·
[Gitee API v5](https://gitee.com/api/v5/swagger)

---

## 更新日志

### v0.12.2 (2026-07-11) — 正确性与简化批次

> 对全项目做了一轮四维度 code review（76 条原始发现 → 67 条确认），本批次修复全部 4 个 high bug 与 26 个 medium，37 个 low 留后续。

**🔧 正确性修复（high）**

- `api` 客户端 retry 只对 GET 生效——写操作（POST/PUT/PATCH/DELETE）收到 5xx 或网络错误时不再重试，避免服务端已生效但响应失败时**创建重复 issue/PR/release/gist**
- `auth status` / `auth token` / `auth logout` 改用 `ResolveHostname()`——`gitee auth switch` 后这些子命令终于查对 host（之前 switch 对它们完全失效）
- `gitee api POST /user/repos --field name=x` 现在可用——`--field`/`--raw` 此前只注册在 `api raw` 子命令，canonical 入口 `gitee api` 报 unknown flag
- `gitee ssh-key list --limit --page` 现在真正下发 `per_page`/`page` 查询参数——此前 flag 注册了但被忽略

**🔧 正确性修复（medium）**

- `pr checkout`/`pr view` 增加 `Head`/`Base` nil 守卫，API 返回缺字段时不再 panic
- `pr edit` 补注册 `--id/-i` flag（此前文档化了但未注册，传 `--id` 报 unknown flag）
- `pr status` 补注册 `--limit/-L` 并接入分页
- `gist comments --page` 下发查询参数；`gist view` 不再静默吞掉 raw_url 拉取错误
- `release download` 检查 `Close` 错误并在 `io.Copy` 失败时清理半截文件
- `repo sync` 用 typed `IsUnprocessable` 替代 `strings.Contains(err,"422")` 字符串匹配
- `repo delete` 用 `utils.ConfirmYN` 替代裸 `fmt.Scanln`（补齐非-TTY/EOF 处理）
- `notification read` 确认提示文案随 `--unread` 方向切换

**♻️ 简化与一致性**

- `issue`/`pr`/`repo commits` list 统一改用 `api.FetchListOrAll`，删除 3 份重复的 `addPageParams` + 内联 perPage clamp，并修复 `issue list` 单页路径不处理 404 的行为不一致
- `issue create/list/view` 改用 `api` 端点 builder，不再在 cmd 层手拼 URL
- `org list/repo/member` 查询串改用 `url.Values` + perPage clamp
- 29 处表格输出统一改用 `output.NewTableWriter()`
- `extension` 复用 `utils.ValidateName` 与 `atomicfile.Write`（删除逐字节重复的本地副本）
- 抽取 `utils.ParseGistFileFlag`，消除 `gist create`/`edit` 重复的 `--file` 解析
- 提取共享 `cmd.AgentHelpNA` helper，消除 auth 6 个子命令 + config 的内联 n/a 闭包
- `alias --agent-help --output json` 恢复产出结构化 JSON（此前绕过了 `PrintAgentHelp`）
- 删除死 flag `auth login --scopes`/`--web`、死函数 `colorAttr`/`urlEncodeRef`、死分支若干
- 修正 agent-help 中不存在的 `--reason completed` 文档、过时版本号（v0.10.0/v0.11.0/v0.9.x → v0.12.1）

**📖 文档**

- 本仓库的开发指引文档修正：项目结构与技术栈对齐实际（移除不存在的 `internal/auth`/`printer`/`i18n`、未引入的 Viper/oauth2，凭证路径更正为 `~/.gitee-cli/settings.json`，端点 builder 与 `FetchListOrAll` 等开发约定补入）

### v0.12.1 (2026-07-04) — Agent Surface 可达性

- **🔧 修复**: 17 个带必填位置参数的命令，`--agent-help` 不带参数也能出文档（新增 `AgentHelpAwareArgs`）
- **🔧 修复**: `gitee api raw --agent-help` 不再要求认证——查文档不需要 token
- **🔧 修复**: `auth` / `config` 全部子命令 `--agent-help` 打印 n/a 说明并退出，不再误执行真命令
- **📖 文档**: agent-help 覆盖 42 → **131 verbs**，philosophy.md 覆盖矩阵同步刷新

### v0.12.0 (2026-07-04) — 11 项新功能

- **🆕 新命令组**: `milestone`（里程碑全 CRUD）· `notification`（终端收通知）· `user`（资料/关注）
- **🆕 issue**: `edit` / `close` / `reopen`
- **🆕 repo**: `tag` · `collaborator` · `star`/`unstar` · `watch`/`unwatch` ·
  `file view/push/delete`（不 clone 直接读写仓库文件）· `commits` · `compare` · `protect`/`unprotect`
- **🆕 gist**: `comments` / `comment` / `comment-delete`
- 每个新 verb 都带完整 11 维度 `--agent-help`，contract test 守护覆盖率

### 更早版本

| 版本 | 要点 |
|------|------|
| v0.11.0 | agent-help 覆盖 38 → 78 命令 |
| v0.10.x | `agent onboard`、`--agent-help -o json`、philosophy.md 发布 |
| v0.4.x | **`--agent-help` 双面设计首发**；真分页修复；5 平台 release 流水线 |
| v0.3.x | PR review 修复、错误处理与输出统一、测试体系起步 |

完整历史见 [Releases](https://gitee.com/andershsueh/gitee-cli/releases) 与 git log。

---

## License

[木兰宽松许可证 v2](LICENSE)

## ☕ Buy Me Coffee

如果这个项目帮到了你——或者帮到了你的 Agent——可以请我喝杯咖啡：

![Buy Me Coffee](./etc/BuyMeCoffe.jpg)
