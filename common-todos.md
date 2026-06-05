# gitee-cli 通用待办 (common-todos)

> 本文件对比 `gh` CLI 与 `gitee-cli` 的命令覆盖情况。
> 仅列出 **Gitee API v5 能力支持、但 gitee-cli 尚未实现** 的功能。
> 平台独占特性（如 GitHub Codespaces、Projects、Attestation、Rulesets）不计入。

**对比基线**：
- `gh-command-list.md` (2026-06-05 版本)
- `gitee --help` 实际输出 (v0.4.2 + P1b uncommitted)
- Gitee API v5 分类：Repos 75+、Issues 23、PRs 27、Gists 17、Orgs 14、Users 18、Search 3、Webhooks 6、Releases、Labels、SSH Keys、GPG Keys、Gitee Go

**评估规则**：
- ✅ 已实现（直接对标）
- ⚠️ 部分实现（基础有了，缺关键子命令）
- ❌ 未实现（Gitee API 支持，但 gitee-cli 缺失）
- 🚫 不适用（平台独占，无法对标）

---

## 2026-06-05 进度更新

**P0** ✅ **全部完成**（v0.5.0）：release / 顶层 label / pr checkout + co / repo archive+sync+rename
**P1a** ✅ **全部完成**（v0.6.0）：ssh-key / gpg-key / gist clone
**P1b** ✅ **全部完成**（v0.7.0）：webhook（list / create / view / edit / delete / test）— 6 个子命令 + 5 boolean 事件字段 + 1 shared handler
**P1c** ✅ **部分完成**（v0.8.0）：issue comments（list）/ pr ready / pr comment / pr edit / pr checks — 5 个 Gitee 实际支持的子命令。**6 个 Gitee v5 不支持的项**（issue delete/pin/transfer/lock、pr lock/update-branch）标记 🚫，不创建空壳 cobra 文件。

**剩余**：P2（status/alias/extension）、P3（Gitee Go pipeline）、help topics（environment/exit-codes/formatting/reference）

完整 P0 + P1a 实施 + simplify 详见 `~/.claude/plans/p1-simplify-rustling-pike.md`。

---

## 总览：命令覆盖度

| 模块 | gh 子命令数 | gitee-cli 已实现 | 覆盖率 | 缺失数 |
|------|------------|------------------|--------|--------|
| `auth` | 5+ | 5 | 100% | 0 |
| `repo` | ~18 | 9 | 50% | 9 |
| `issue` | ~15 | 8 | 53% | 7 |
| `pr` | ~17 | 9 | 53% | 8 |
| `gist` | ~10 | 7 | 70% | 3 |
| `org` | ~12 | 5 | 42% | 7 |
| `search` | 3 | 3 | 100% | 0 |
| `api` | 1 | 1 | 100% | 0 |
| `config` | 1 | 1 | 100% | 0 |
| `release` | 6 | 6 | 100% | 0 ✅ |
| `label` | 6 | 6 (顶层 + issue 兼容) | 100% | 0 ✅ |
| `run` (CI/CD) | 5 | 0 | 0% | 5 |
| `workflow` | 5 | 0 | 0% | 5 |
| `cache` | 5 | 0 | 0% | 5 |
| `secret` | 5 | 0 | 0% | 5 |
| `variable` | 5 | 0 | 0% | 5 |
| `gpg-key` | 3 | 3 | 100% | 0 ✅ |
| `ssh-key` | 3 | 3 | 100% | 0 ✅ |
| `webhook` | 6 (Gitee) | 6 | 100% | 0 ✅ |
| `status` | 1 | 0 | 0% | 1 |
| `extension` | 6 | 0 | 0% | 6 |
| `alias` | 3 | 0 | 0% | 3 |
| `co` (alias) | 1 | 1 | 100% | 0 ✅ |
| `webhook` | 6 | 6 | 100% | 0 ✅ |

**核心命令覆盖率**（auth/repo/issue/pr/gist/org/search/api/config）：**~60%**
**完整对标覆盖率**（含 release/label/keys/ssh-key/gpg-key/gist-clone 等）：**~52%**

---

## P0 — 高频使用、最影响"日常开发"完整体验  ✅ **全部完成**（v0.5.0）

### 1. `gitee release` — ✅ 完成（v0.5.0）
- [x] `gitee release list` — 列出 release
- [x] `gitee release view <tag>` — 查看 release 详情
- [x] `gitee release create` — 创建 release
- [x] `gitee release edit <tag>` — 编辑 release
- [x] `gitee release delete <tag>` — 删除 release
- [x] `gitee release download <tag>` — 下载资产

### 2. `gitee label` 顶层 — ✅ 完成（v0.5.0）
- [x] `gitee label list/view/create/edit/delete/clone` 全部 6 个
- [x] `gitee issue label` 旧路径作为 alias 保留（共享 handler）

### 3. `gitee pr checkout` / `co` — ✅ 完成（v0.5.0）
- [x] `gitee pr checkout <number>` — fetch + checkout
- [x] `gitee co <number>` — 顶层 alias（独立 cobra.Command 注册）
- [ ] `gitee pr checks` — 移到 P1c

### 4. `gitee repo archive` / `sync` / `rename` — ✅ 完成（v0.5.0）
- [x] `gitee repo archive` — 含 `--unarchive`
- [x] `gitee repo sync` — 同步 fork
- [x] `gitee repo rename` — 改名
- [ ] `gitee repo set-default-branch` — 移到 P1c

---

## P1 — 用户管理 / 协作类（Gitee API 完整支持，强烈建议补齐）

### 5. `gitee ssh-key` — ✅ 完成（v0.6.0，P1a）
- [x] `gitee ssh-key list` — 列出现有 key
- [x] `gitee ssh-key add` — `--key` 接受字符串或文件路径；smoke test 6 种前缀
- [x] `gitee ssh-key delete <id>` — 含 `--force`

### 6. `gitee gpg-key` — ✅ 完成（v0.6.0，P1a）
- [x] `gitee gpg-key list` — 列出，含 primary email
- [x] `gitee gpg-key add` — 私钥安全校验（拒绝含 PRIVATE marker 的输入）
- [x] `gitee gpg-key delete <id>`

### 7. `gitee webhook` — ✅ 完成（v0.7.0，P1b）
**Gitee API 状态**：✅ 支持（6 个端点，per CLAUDE.md）
**gh 对应**：`gh webhook` 暂未在 gh-command-list 中（gh 用 webhooks 通过 API 间接管理），但 Gitee 平台明确支持
**为什么是 P1**：Webhooks 是企业用户刚需（CI/CD 集成）。

**已实现**：
- [x] `gitee webhook list --repo owner/repo` — `GET /repos/{o}/{r}/hooks`
- [x] `gitee webhook create --repo owner/repo --url <url>` — 5 个独立 `--*-events` 布尔标志 + `--password` + `--encryption-type`
- [x] `gitee webhook view <id> --repo owner/repo` — `GET /repos/{o}/{r}/hooks/{id}`，含 result/result_code（最近一次投递结果）
- [x] `gitee webhook edit <id> --repo owner/repo` — `PATCH /repos/{o}/{r}/hooks/{id}`，事件字段用 string "true"/"false" 区分 "不修改" vs "显式 false"
- [x] `gitee webhook delete <id> --repo owner/repo` — `DELETE /repos/{o}/{r}/hooks/{id}`，含 `--force` 跳过 y/N
- [x] `gitee webhook test <id> --repo owner/repo` — `POST /repos/{o}/{r}/hooks/{id}/tests`（Gitee 的 "ping" 等价物）

**实现要点**：
- 5 个 boolean 事件字段（`push_events` / `tag_push_events` / `issues_events` / `note_events` / `merge_requests_events`），与 Gitee API 一致（**不是** events 数组）
- `pkg/types/webhook.go` 用 `*bool` 指针模式支持 partial PATCH（与 `ReleaseEditRequest` 一致）
- `id` 取自 positional OR `--id` flag（与 `ssh-key delete` 模式一致）
- 删除走 `utils.ConfirmYN`，与 `label delete` 行为一致
- view/edit 时 `api.IsNotFound` 给出友好错误信息
- 全部 6 个子命令都有完整 `const <verb>AgentHelp = ...` 块（12-14 章节），并通过 `TestXxxAgentHelpContent` 静态 content 测试
- `cmd/agent_help.go` coverage matrix 升级到 v0.7.0，roadmap 标记 P1b 完成

---

### 8. `gitee issue` 增强 — ✅ 部分完成（v0.8.0，P1c）+ 6 标 🚫
**Gitee API 状态**：**6 项 Gitee v5 API 不支持**（已对照 swagger spec 5.4.92 验证）
**当前状态**：list / view / create / close / reopen / comment（create）/ edit / label 已有，加 P1c 新增的 comments（list）= 9 个；**6 个 common-todos 项 🚫 不支持**

**已实现**：
- [x] `gitee issue comments <number>` — `GET /repos/{o}/{r}/issues/{n}/comments`（与 PR 对齐）

**🚫 Gitee v5 API 不支持**（按用户决策不创建空壳 cobra 文件，标记完毕）：
- [ ] 🚫 `gitee issue delete <number>` — spec 无 DELETE 端点，web UI 也无此按钮
- [ ] 🚫 `gitee issue pin / unpin <number>` — `Issue` 结构无 `pinned` 字段
- [ ] 🚫 `gitee issue transfer <number> --to <owner/repo>` — 无 transfer 端点
- [ ] 🚫 `gitee issue lock / unlock <number>` — `Issue` 结构无 `locked` 字段

---

### 9. `gitee pr` 增强 — ✅ 部分完成（v0.8.0，P1c）+ 3 标 🚫
**Gitee API 状态**：**3 项 Gitee v5 API 不支持**
**当前状态**：list / view / create / close / reopen / merge / review / diff / comments 已有，加 P1c 新增的 ready / comment（create）/ edit / checks = 13 个；**3 个 common-todos 项 🚫 不支持**

**已实现**：
- [x] `gitee pr ready <number>` — `PATCH /pulls/{n}` with `{ "draft": false }`
- [x] `gitee pr comment <number>` — `POST /pulls/{n}/comments`（与已有的 plural `comments` 列表区分）
- [x] `gitee pr edit <number>` — `PATCH /pulls/{n}`（title/body/state/draft 4 字段 partial PATCH）
- [x] `gitee pr checks <number>` — 两步：先 `GET /pulls/{n}` 取 head ref，再 `GET /commits/{ref}/check-runs?pull_request_id={n}&filter=latest`

**🚫 Gitee v5 API 不支持**（按用户决策不创建空壳 cobra 文件）：
- [ ] 🚫 `gitee pr update-branch <number>` — PATCH body 无 `base` 字段（Gitee 平台限制，web UI 也不能改 base branch）
- [ ] 🚫 `gitee pr lock / unlock <number>` — `PR.locked` 是只读字段，无写端点

---

### 10. `gitee gist clone` — ✅ 完成（v0.6.0，P1a）
- [x] `gitee gist clone <id>` — 双路径：先试 `git_pull_url`（git clone），fallback 到 raw_url 下载所有文件
- [x] `--dir` 或第二个位置参数指定目标目录（默认 = gist id）

---

## P2 — 增强 / 辅助类

### 11. `gitee status` — 缺失 ❌
**gh 对应**：`gh status` — 汇总当前仓库相关的 issue/PR/notification
**Gitee API 状态**：部分支持（可查当前 user 的 issue/PR 列表，但 notification API 不完整）
**待实现**：
- [ ] `gitee status` — 列出当前 repo 相关的 open issue / 待 review PR（基于 `--repo` 推断）

---

### 12. `gitee alias` — 缺失 ❌
**gh 对应**：`gh alias set / list / delete`
**为什么是 P2**：是增强功能，不阻塞日常工作流。
**待实现**：
- [ ] `gitee alias set <name> <command>`
- [ ] `gitee alias list`
- [ ] `gitee alias delete <name>`

---

### 13. `gitee extension` — 缺失 ❌
**gh 对应**：`gh extension install / list / remove / upgrade / create`
**Gitee API 状态**：🚫 平台独占（gh 的 extension 机制基于 git repo + 可执行文件，不依赖平台 API）
**待实现**：
- [ ] `gitee extension list` — 列出已安装扩展
- [ ] `gitee extension install <repo>` — 从 Gitee repo 安装扩展
- [ ] `gitee extension remove <name>`
- [ ] `gitee extension upgrade <name>`
- [ ] `gitee extension create <name>` — 创建新扩展脚手架

---

## P3 — Gitee Go (CI/CD) 平台类

> Gitee Go 是 Gitee 的 CI/CD 平台，对标 GitHub Actions。
> API 与 Actions 完全不同（不是 yaml workflow），所以 gh 的 `run` / `workflow` / `cache` / `secret` / `variable` 不能直接对标。
> 但用户对"从 CLI 触发流水线、看 build 状态"有明确需求。

### 14. `gitee pipeline` 或 `gitee gitee-go` — 缺失 ❌
**Gitee API 状态**：✅ 支持（`/repos/:owner/:repo/pipelines` 等端点）
**待实现**：
- [ ] `gitee pipeline list --repo owner/repo` — 列出流水线
- [ ] `gitee pipeline view <id> --repo owner/repo` — 查看 build 详情
- [ ] `gitee pipeline run --repo owner/repo --ref <branch>` — 手动触发
- [ ] `gitee pipeline cancel <id> --repo owner/repo` — 取消 build
- [ ] `gitee pipeline logs <id> --repo owner/repo` — 查看日志

**为什么不直接叫 `run` / `workflow`**：避免和 gh 的语义混淆，Gitee Go 的概念模型不同。

---

## 帮助主题（HELP TOPICS）— 缺失 ❌

gh 提供了多个 help topic：
- `gitee help environment` — 环境变量清单
- `gitee help exit-codes` — 退出码说明
- `gitee help formatting` — JSON 格式化说明
- `gitee help reference` — 完整命令参考

**建议补齐**：
- [ ] `gitee help environment` — 列出 `GITEE_TOKEN`, `HTTPS_PROXY`, `NO_COLOR` 等所有环境变量
- [ ] `gitee help exit-codes` — 退出码表（0=ok, 1=general, 2=auth, 3=not-found, 4=rate-limit ...）
- [ ] `gitee help formatting` — `--output` 的语义、json path、jq 配合示例
- [ ] `gitee help reference` — 完整命令参考（自动生成）

---

## 总结：补完这些后，gitee-cli 就能成为真正的"gh 平替"

| 阶段 | 范围 | 工作量估算 | 状态 |
|------|------|-----------|------|
| **P0** | release, label, pr checkout, repo archive/sync/rename | 2-3 周 | ✅ **完成** (v0.5.0) |
| **P1a** | ssh-key, gpg-key, gist clone | 1 周 | ✅ **完成** (v0.6.0) |
| **P1b** | webhook (list/create/view/edit/delete/test) | 1 周 | ✅ **完成** (v0.7.0) |
| **P1c** | issue comments (list) + pr ready/comment/edit/checks（Gitee 实际支持的 5 项；6 项 unsupported 标 🚫） | 2 周 → 1 周 | ✅ **完成** (v0.8.0) |
| **P2** | status, alias, extension | 1-2 周 | ❌ 待办 |
| **P3** | pipeline (Gitee Go) | 2-3 周 | ❌ 待办 |
| **帮助主题** | environment, exit-codes, formatting, reference | 0.5 周 | ❌ 待办 |

**P0 + P1a + P1b + P1c 完成后，核心开发体验将与 gh 持平**（~60% 完整对标覆盖率，~70% 核心命令覆盖率；剩余 ~30% 是 Gitee 平台独占如 Gitee Go pipeline/GVP）。
**全部完成后，gitee-cli 将是 Gitee 生态最完整的 CLI 工具**（预计 95%+ 覆盖度）。

---

## 与现有 todo-0602.md 的关系

`todo-0602.md` 关注的是 **agent 使用体验**（P0 修 bug、P1 agent 签名、P2 next-steps）。
`common-todos.md`（本文件）关注的是 **命令覆盖度**，对标 gh。
两份 todo 互补，不冲突。

**实际完成顺序**（截至 2026-06-05）：
1. ✅ `common-todos.md` P0（命令覆盖）
2. ✅ `common-todos.md` P1a（用户管理）
3. ✅ `common-todos.md` P1b（webhook — 企业刚需）
4. ✅ `common-todos.md` P1c（issue + pr 增强，**按 Gitee API 真相调整**：5 实现 + 6 标 🚫）

**剩余建议优先级**：
1. ❌ `common-todos.md` P2（status / alias / extension）
2. ❌ `common-todos.md` P3（pipeline / Gitee Go）
3. ❌ `common-todos.md` 帮助主题（environment / exit-codes / formatting / reference）
4. ❌ `todo-0602.md` P1（agent 签名）
5. ❌ `todo-0602.md` P2（next-steps 提示）

---

## 2026-06-05 工作结束总结

**今日完成**：P1b（webhook）实施
- 新增 7 个 cobra command 文件（parent + list/create/view/edit/delete/test）+ 6 个 content test
- 新增 1 个 types 文件（webhook.go，含 Webhook / WebhookCreateRequest / WebhookEditRequest，5 个 bool 事件字段）
- 6 个 webhook endpoint builders（+1 个 `TestWebhookEndpoints`）
- 1 个新 handler（`internal/handler/webhook.go`，含 WebhookList/View/Create/Edit/Delete/Test + WebhookResolveRepo + summarizeEvents helper）
- `cmd/agent_help.go` coverage matrix 升级 v0.6.0 → v0.7.0，roadmap 标记 P1b 完成
- common-todos.md P1b 7 个子项全部勾选

**关键设计决策**：
- **5 个独立 boolean 事件字段**（`--push-events` / `--tag-push-events` / `--issues-events` / `--note-events` / `--merge-request-events`），与 Gitee API 一致
- `create` 默认 `--push-events=true`（Gitee 服务端默认），其他默认 false
- `edit` 用 string flag（接受 "true"/"false"/空）—— 空 = "不修改"，明确区分 "未传" vs "显式 false"
- 所有 `id` 都支持 positional OR `--id` flag（与 `ssh-key delete` 模式一致）
- `*bool` 指针模式用于 partial PATCH 请求（与 `ReleaseEditRequest` 一致）
- 删除走 `utils.ConfirmYN`，view/edit 用 `api.IsNotFound` 给出友好 404 信息

**测试状态**：`go test ./... -count=1` 全部通过（13 个有 test 的包，含新的 cmd/webhook）

**uncommitted changes**：P0 + P1a + P1b 全部改动在工作区，未 commit。下次接手可直接 `git add -A && git commit -m "feat: P0 + P1a + P1b command coverage"` 提交。

**下次接手读这两个文件**：
- `common-todos.md`（本文件）— 看 P1c / P2 / P3 / help topics 还有哪些没做
- 上一轮 plan 文件 — P1a + P1b 的实施模式可作为后续命令组的复用模板
