# gitee-cli

> 在终端里完整操作 Gitee 的现代命令行工具，功能对标 GitHub `gh` ——
> 并且是**全球第一个**同时为人类和 AI Agent 设计帮助系统的开源 CLI。

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.12.6-orange.svg)](https://gitee.com/andershsueh/gitee-cli/releases/latest)
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

**当前覆盖：137 个 verb（v0.12.6 实测）**，并由 contract test 在 CI 层守护——
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

### v0.12.6 (2026-08-15) — `gitee issue close/reopen/edit` 根因修复（v0.12.5 workaround 升级）

> 1 个 commit。在 v0.12.5 的 `formatStateTransitionError` 优雅降级基础上，找出真正的根因——Gitee v5 写路径的形态与读路径不同，且只接受 formData。close / reopen / edit 三个 verb 现在直接走正确契约，HTTP 200 完成；v0.12.5 的 workaround 仍然保留作为边缘场景的兜底（enterprise/project 容器路由不到时仍按 v0.12.5 方式提示 Web UI URL）。

**🔧 正确性根因修复**

- `gitee issue close` / `reopen` / `edit` 此前对所有个人/普通组织仓返回 `HTTP 400` / `HTTP 404`，根因：
  - **路径**：`PATCH /repos/{owner}/{repo}/issues/{number}`（用了 GET 读路径形状） → 正确：`PATCH /repos/{owner}/issues/{number}`（写路径**去掉** `{repo}`，`repo` 改用 formData 字段提交）。
  - **Body 编码**：`Content-Type: application/json` `{"state":"closed"}` → 正确：`application/x-www-form-urlencoded` `repo=<repo>&state=closed`。
  - 旧实现同时踩了两个坑，所以 v0.12.5 的 workaround 永远会被触发，因为它试图包装一个 404，而实际原因不是 Gitee v5 拒绝 enterprise，而是请求形态根本不对。
- `internal/api/endpoints.go`：`IssuesEdit` 路径削掉 `{repo}`，附详细 doc 注释引用 swagger 与实测链接。
- `internal/api/client.go`：新增 `Client.PatchForm(ctx, path, url.Values, result)`（走 resty `SetFormData`，Content-Type 自动设为 `application/x-www-form-urlencoded`）。`Patch`（JSON）保留给真吃 JSON 的端点（如 `repo edit`）。
- `cmd/issue/helpers.go`：`runClose` / `runReopen` / `runEdit` 改用 `url.Values` + `PatchForm`，显式提交 `repo` + `state` 字段；`edit` 把 `assignee` / `milestone` 字段名对齐 Gitee v5 真实契约（不再发 `assignees` / `milestone_number`）。
- v0.12.5 的 `formatStateTransitionError` + `IsProjectOrEnterpriseNotFound` 兜底**保留**：当修复后仍偶发 404（如 enterprise/project 容器路由不可达），错误信息仍带 Web UI URL 引导。

**📖 文档同步**

- `issue edit` / `close` / `reopen` 三段 `--agent-help` 都新增 `## Implementation notes (Gitee v5 quirk)` 章节，写明真实 wire shape + 排查指向（`internal/api/client.go` PatchForm + `internal/api/endpoints.go` IssuesEdit）。
- `cmd/issue/{close,edit,reopen}_content_test.go` 契约测试同步加入新章节作为 required list 元素，防止后续重构被默默删掉。

**🧪 测试**

- `internal/api/endpoints_test.go` 加 `TestIssuesEditEndpoint`：锁定 `/repos/{owner}/issues/{number}` 不带 `{repo}` 段。
- `internal/api/client_test.go` 加 3 个回归测试（路径 + Content-Type + form 字段全维度）：`TestClient_PatchForm` / `TestClient_PatchForm_EmptyForm` / `TestClient_PatchForm_400Error`。
- `cmd/issue/state_transition_test.go` 既有 18 个测试更新断言到新契约（路径去掉 `{repo}` + formData 含 `repo`/`state`）。

**🔁 验证**

- 真实端到端：用本仓库 `IK5FZX` 跑 `close` → `state=closed` → `reopen` → `state=open` → `edit --title …`，全部 HTTP 200，状态正确切换。
- `go test ./...` 32 个包全绿，`golangci-lint run ./...` 0 issue。

### v0.12.5 (2026-07-31) — `gitee issue close/reopen` 优雅降级 Gitee v5 路由 404

> 1 个 commit：1 处 API 兼容性降级（#IK5FZX），gitee-cli 不再因 Gitee v5 上游 API 变更而给用户死端。**无新功能，无 breaking change**，行为对外保持一致；`golangci-lint run ./...` 与 `go test ./...` 持续全绿。

**🔧 上游兼容**

- **`gitee issue close/reopen` 优雅降级 Gitee v5 路由 404（#IK5FZX）** — 调研发现 Gitee v5 路由层不再接受对归属 enterprise/project 容器的 issue 做 PATCH state 修改（`/repos/{owner}/{repo}/issues/{number}` 返 404 `project or enterprise`）。多个第三方 SDK、中文社区博客与官方 swagger 一致指向该端点，无备选端点可 fallback，gitee-cli 这侧能做的是把已知失败模式显式识别出来包装出可操作错误。修复：新增 `api.IsProjectOrEnterpriseNotFound(err)` predicate（用 `errors.As(*APIError)` + `StatusCode==404` + `apiErr.Message` 子串匹配，避免外层包装误判）；`runClose` / `runReopen` 改走 `formatStateTransitionError`，404 模式时输出含 `cmd.ResolveHostname()` 生成的 Web UI URL 与人工操作指引的多行错误；其他错误保持原 `closing issue: %w` 包装。

**🧪 测试**

- 合并原 `close_test.go` + `reopen_test.go` 为 `state_transition_test.go`（189+189 → 226 行带 t.Run 子测试），共享 server setup / globals restore 模板；`io.ReadAll` 替代 `r.Body.Read(len=ContentLength)` 短读模式；`t.Cleanup` 替代 `defer`。
- 新增 `internal/api/errors_test.go::TestIsProjectOrEnterpriseNotFound` 覆盖 nil、随机 error、含/不含 signature 的 404、非 404、wrapped chain、外层文本含 signature 但内层 APIError 不含（验证 predicate 严格基于 `*APIError.Message`）等 case。
- `cmd/issue` 现有 18 个测试全部保留 + 通过。

### v0.12.4 (2026-07-31) — `repo edit` PATCH 修复 + `--public` 反向开关

> 1 个 commit：2 处正确性修复（#IK5FYW + #IK5FYX），新增 7 个 wire-format 测试。无新功能，无 breaking change，golangci-lint 与 `go test ./...` 持续全绿。

**🔧 正确性修复**

- **`repo edit` / `archive` PATCH 修复（#IK5FYW）** — Gitee API v5 强制 `PATCH /repos/{owner}/{repo}` 的 body 必须含 `name` 字段。之前 `cmd/repo/edit.go` 在用户没传 `--name` 时把 `req.Name` 留空，`cmd/repo/archive.go` 干脆没设 `Name`，加上 `pkg/types/repo.go` 的 `name` 字段是 `omitempty`，导致任何非 rename 路径（`--description`、`--private`、`--default-branch`、archive、unarchive）都返回 HTTP 400 `name is missing`。修复：3 处调用方都把当前 `repoName` 作为 baseline 写入 `req.Name`，`--name` 显式传入时再覆盖为新名；`types.RepositoryEditRequest.Name` 同步去掉 `omitempty` 并加契约注释。
- **`repo edit --public` 反向开关（#IK5FYX）** — `--private` 是 cobra `BoolVar` 注册的"开关"，无反向标志；`--no-private` 也不会触发 `Changed("private")`，导致把 private 仓改回 public 没有任何 CLI 出口。修复：增加 `--public` 标志并用 `MarkFlagsMutuallyExclusive("private", "public")` 与 `--private` 互斥；`--public` → `private: false`。`--agent-help` 文档同步更新示例与互斥说明。

**🧪 测试**

- 新增 `cmd/repo/edit_test.go`（7 个 case）：name baseline、--public 路径、--private 路径、互斥短路（确认 HTTP 不被调用）、--name 覆盖、空 body 形态、agent-help 文档契约。
- `registerEditFlags` 从 `init()` 抽出，使测试可 `ResetFlags() + registerEditFlags()` 重建 FlagSet 清除 pflag `Changed` map 的状态泄漏。
- 测试模式（`httptest.NewServer` + 5 个 cmd global 保存/恢复）抽取为 `captureEditRequest` / `expectNoEditRequest` 两个 helper，避免 wire-format 测试之间的 setup 重复。

### v0.12.3 (2026-07-14) — 累积修复与内部清理

> 19 个 commit：6 处正确性修复 + 11 处内部重构 + 2 项 lint 工程化。**无新功能，无 breaking change**。行为对外保持完全一致，代码侧提升可维护性、`golangci-lint run ./...` 持续 0 issue。

**🔧 正确性修复**

- `issue view` 增加本地 `--owner`/`--repo` 标志——脱离 git 仓库时也能直接查询任意仓库的 issue
- `ssh-key delete` 在 agent-help 中提示非交互式场景下 `--force` 缺省会自动取消，避免误删
- `release download` 检查 `resp.Body.Close` 错误（之前静默吞掉），写盘失败时清理半截文件
- `types`: 修正拼写错误的 `Comitter` 字段名为 `Committer`（影响 1 处 webhook payload）
- `utils.GitConfigGet`: 区分 `git` exit-code-1（key 不存在）与真实错误（命令失败），避免把"未配置"误报为故障
- `api raw` 输出缩进后 JSON 时检查 `Indent`/`Write` 错误而非静默吞掉

**♻️ 重构（行为不变）**

- `repo list`: 统一走 `api.FetchListOrAll`，删除重复 `addPageParams`；`repo delete` 折入 `do()`，去重 raw JSON 输出
- `api`: 抽取 `do()` helper 收敛 GET/POST/PATCH/PUT/DELETE 公共逻辑；删除死代码 `Unwrap`/`Errors` 与未用的 Link-header 分页
- `repo`/`org`/`extension`: 查询串改用 `url.Values` + perPage clamp，删除裸 `fmt.Sprintf` 拼接
- `auth`: `token source` 标签统一为 `flag`/`env`/`config`，拆分 `logout.go`，集中 `switch` 初始化
- `root`: 用 `Flags().Changed` 替代自维护的 `--hostname` 集合；提取 `ResolveAPIBaseURL`，合并 reservedNames 列表
- `gist`: 导出 `GistEditFile` 类型，删除不可达的 `create` 分支
- `utils`: 移除未使用的 `contains`/`indexOf` test helper；`url.Values` 校验 list flags
- `label`/`extension`/`search`: 简化十六进制颜色检查（De Morgan）、删除死 `colorAttr`、去除冗余 type-assert

**🛠️ 工程化**

- `golangci-lint v2.12.2` 全面生效：`.golangci.yml` 改写为 v2 schema，删除冲突的 `exclude-use-default`，删除仓库内遗留的 v1.64.8 二进制
- CI 端 `go test ./...` 32 包全绿，`golangci-lint run ./...` 0 issue

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
