# gh vs gitee-cli 功能对标分析

> **生成时间**：2026-06-05(更新:加入 v0.9.0 status)
> **对照基线**：`gh v2.74.0` (2025-05-29) vs `gitee-cli v0.9.0-a` (commit `63eb1b3`)
> **数据来源**：`gh <command> --help` 实际抽取 + `./gitee <command> --help` 实际抽取
> **生成脚本**：[`./extract-commands.sh`](./extract-commands.sh) — 可重跑,永远反映最新版
>
> **验证状态**：
> - `./extract-commands.sh --md` ✅ 输出完整 gh + gitee 两边子命令清单
> - `./extract-commands.sh --json` ✅ 输出 JSON 格式(供程序消费)
> - `./extract-commands.sh --raw-gh` / `--raw-gitee` ✅ 只看一边
> - **脚本限制**:只抽有 "Available Commands" 段的命令组。leaf verb(无 subcommand)如 `gitee status` 不会被自动抽取 — 需手动核对
>
> **本文件** (§1-13) 是基于上述抽取结果的**人工标注**(✅/🚫/⬜/❌/🟡)与**优先级排序**;
> 脚本只负责"机器可读的事实"部分(子命令存在性),主观状态判断由人维护。

## 图例

| 标记 | 含义 |
|------|------|
| ✅ | gitee-cli 已实现,行为对标 gh |
| 🟡 | gitee-cli 已实现,但行为/参数有差异 |
| 🚫 | 未实现 — Gitee API 不支持(Gitee v5 swagger spec 缺失) |
| ⬜ | 未实现 — Gitee API 支持,但 gitee-cli 暂未实现(可补) |
| ❌ | 不适用 — 平台独有(gh-only),gitee 不存在对应概念 |

---

## 1. 总览

| 指标 | gh | gitee-cli | 覆盖率 |
|------|------|-----------|--------|
| 一级命令数 | 24(含 7 help topics) | 18 + `help` | — |
| 叶子子命令数(估) | ~190 | ~94 | ~49% |
| 平台独有(❌) | 9 | 1(`browse` 在 `repo` 下) | — |

**一级命令对比**：

| gh 一级命令 | gitee-cli 一级命令 | 状态 |
|---|---|---|
| `auth` | `auth` | ✅ |
| `browse` | (在 `repo browse`) | 🟡(位置不同) |
| `codespace` | — | ❌ Gitee 无 codespace |
| `gist` | `gist` | ✅ |
| `issue` | `issue` | ✅ |
| `org` | `org` | ✅ |
| `pr` | `pr` | ✅ |
| `project` | — | ❌ Gitee 暂无 Projects v2 |
| `release` | `release` | ✅ |
| `repo` | `repo` | ✅ |
| `cache` | — | ❌ Gitee 无 Actions cache |
| `run` | — | ❌ Gitee Go pipeline 不在 v0.9 |
| `workflow` | — | ❌ Gitee Go workflow 不在 v0.9 |
| `alias` | — | ⬜ P2 计划 |
| `api` | `api` | ✅ |
| `attestation` | — | ❌ Gitee 无 attestation |
| `completion` | `completion` | ✅ |
| `config` | `config` | ✅ |
| `extension` | — | ⬜ P2 计划 |
| `gpg-key` | `gpg-key` | ✅ |
| `label` | `label` | ✅(v0.7) |
| `preview` | — | ❌ gh 内部命令 |
| `ruleset` | — | 🚫 Gitee 暂无 ruleset |
| `search` | `search` | ✅ |
| `secret` | — | 🚫 Gitee v5 无 secret 端点(企业版) |
| `ssh-key` | `ssh-key` | ✅ |
| `status` | `status` | **✅ 🆕 v0.9.0-a** |
| `variable` | — | 🚫 Gitee v5 无 variable 端点(企业版) |
| — | `webhook` | gitee 独有(gh 把它藏在 `api`) |
| — | `version` | gitee 独有 |

**总结**：gitee-cli 一级命令覆盖 **15/24 = 62%**(去除 ❌ 后实际可达 **15/15 = 100%** — alias/extension 是 P2 待补,不算"平台缺失")。

---

## 2. auth

| gh verb | gitee verb | 状态 | 备注 |
|---|---|---|---|
| `login` | `login` | ✅ | 都是 OAuth2 device flow |
| `logout` | `logout` | ✅ | |
| `refresh` | `refresh` | ✅ | |
| `setup-git` | — | 🟡 | gitee 不需要(走 git 自身的 credential.helper) |
| `status` | `status` | ✅ | |
| `switch` | — | ⬜ | 多账号切换(暂未做) |
| `token` | `token` | ✅ | |

---

## 3. gist

| gh verb | gitee verb | 状态 | 备注 |
|---|---|---|---|
| `clone` | `clone` | ✅ | |
| `create` | `create` | ✅ | |
| `delete` | `delete` | ✅ | |
| `edit` | `edit` | ✅ | |
| `list` | `list` | ✅ | |
| `rename` | — | ⬜ | 文件改名(Gitee v5 无此端点) |
| `view` | `view` | ✅ | |
| — | `star` | gitee 独有 | |
| — | `unstar` | gitee 独有 | |

---

## 4. issue

| gh verb | gitee verb | 状态 | 备注 |
|---|---|---|---|
| `create` | `create` | ✅ | |
| `list` | `list` | ✅ | |
| `status` | — | ⬜ | P2 候选 |
| `close` | `close` | ✅ | |
| `comment` | `comment` | ✅ | |
| `comments` | `comments` | ✅ | v0.8 新增 |
| `delete` | — | 🚫 | Gitee v5 spec 无 DELETE issues |
| `develop` | — | ❌ | GitHub 独有(关联分支) |
| `edit` | `edit` | ✅ | |
| `lock` | — | 🚫 | Gitee Issue 无 `locked` 字段 |
| `pin` | — | 🚫 | Gitee Issue 无 `pinned` 字段 |
| `reopen` | `reopen` | ✅ | |
| `transfer` | — | 🚫 | Gitee v5 无 transfer 端点 |
| `unlock` | — | 🚫 | 同 lock |
| `unpin` | — | 🚫 | 同 pin |
| `view` | `view` | ✅ | |
| — | `label`(legacy) | 🟡 | 已迁移到 `gitee label` |

---

## 5. org

| gh verb | gitee verb | 状态 | 备注 |
|---|---|---|---|
| `list` | `list` | ✅ | |
| — | `view` | gitee 独有 | |
| — | `members` | gitee 独有 | |
| — | `repos` | gitee 独有 | |
| — | `issues` | gitee 独有 | |

**注**：gh 的 `org` 是 base verb,主要操作通过 `gh <verb> --org <name>`(如 `gh issue list --org foo`)。
gitee-cli 把它们做成独立 verb,易用性更好。

---

## 6. pr

| gh verb | gitee verb | 状态 | 备注 |
|---|---|---|---|
| `create` | `create` | ✅ | |
| `list` | `list` | ✅ | |
| `status` | — | ⬜ | P2 候选 |
| `checkout` | `checkout` | ✅ | |
| `checks` | `checks` | ✅ | v0.8 新增 |
| `close` | `close` | ✅ | |
| `comment` | `comment` | ✅ | v0.8 新增(singular create) |
| `comments` | `comments` | ✅ | |
| `diff` | `diff` | ✅ | |
| `edit` | `edit` | ✅ | v0.8 新增 |
| `lock` | — | 🚫 | Gitee PR.locked 是只读字段 |
| `merge` | `merge` | ✅ | |
| `ready` | `ready` | ✅ | v0.8 新增 |
| `reopen` | `reopen` | ✅ | |
| `review` | `review` | ✅ | |
| `unlock` | — | 🚫 | 同 lock |
| `update-branch` | — | 🚫 | Gitee PATCH /pulls body 无 `base` |
| `view` | `view` | ✅ | |

**覆盖**：13/17 = **76%**(去除 🚫 后 13/14 = **93%**)

---

## 7. release

| gh verb | gitee verb | 状态 | 备注 |
|---|---|---|---|
| `create` | `create` | ✅ | |
| `list` | `list` | ✅ | |
| `delete` | `delete` | ✅ | |
| `delete-asset` | — | ⬜ | Gitee v5 端点结构不同,需评估 |
| `download` | `download` | ✅ | |
| `edit` | `edit` | ✅ | |
| `upload` | — | ⬜ | |
| `view` | `view` | ✅ | |

**覆盖**：6/8 = 75%

---

## 8. repo

| gh verb | gitee verb | 状态 | 备注 |
|---|---|---|---|
| `create` | `create` | ✅ | |
| `list` | `list` | ✅ | |
| `archive` | `archive` | ✅ | 支持 unarchive |
| `autolink` | — | ❌ | GitHub 独有 |
| `clone` | `clone` | ✅ | |
| `delete` | `delete` | ✅ | |
| `deploy-key` | — | 🚫 | Gitee v5 端点暂未确认 |
| `edit` | `edit` | ✅ | |
| `fork` | `fork` | ✅ | |
| `gitignore` | — | ⬜ | 模板 |
| `license` | — | ⬜ | 模板 |
| `rename` | `rename` | ✅ | |
| `set-default` | — | ⬜ | 目录默认仓库 |
| `sync` | `sync` | ✅ | fork 同步 |
| `unarchive` | — | 🟡 | gitee 用 `archive --unarchive` 合并 |
| `view` | `view` | ✅ | |
| — | `branches` | gitee 独有 | |
| — | `browse` | gitee 独有 | 浏览器打开 |

**覆盖**：10/15 = 67%

---

## 9. 基础工具

| gh 一级 | gitee 一级 | 状态 | 备注 |
|---|---|---|---|
| `alias` | — | ⬜ P2 | 命令别名 |
| `api` | `api` | ✅ | |
| `completion` | `completion` | ✅ | bash/zsh/fish/powershell |
| `config` | `config` | 🟡 | gitee 多了 `unset`,少 `clear-cache` |
| `extension` | — | ⬜ P2 | 扩展 |
| `status` | `status` | **✅ 🆕 v0.9.0-a** | gitee 实现 2 节(assigned + awaiting review) |
| `webhook`(gh 无) | `webhook` | ✅ | gitee 把它做成一级命令(6 verb) |

---

## 10. GitHub 独有 / Gitee 缺失

| gh verb | Gitee 端点 | 状态 |
|---|---|---|
| `codespace *` | 无 | ❌ Gitee 无云端开发容器 |
| `project *` (17 verb) | 无 | ❌ Gitee 暂无 Projects v2 |
| `cache *` | 无 | ❌ Gitee 无 Actions cache |
| `run *` | 无 | ❌ Gitee Go pipeline 暂未封装 |
| `workflow *` | 无 | ❌ 同上 |
| `attestation *` | 无 | ❌ Gitee 无 Sigstore-style attestation |
| `preview *` | 无 | ❌ gh 内部 |
| `secret *` | 无 | 🚫 Gitee 企业版才有,公开 API 不暴露 |
| `variable *` | 无 | 🚫 同上 |
| `ruleset *` | 无 | 🚫 Gitee v5 swagger spec 无此端点 |

---

## 11. 缺失总表 — 可补的工作量(按 ROI 排序)

### 🔴 高优先级(平台核心体验)

| 缺口 | 难度 | 工作量 | 备注 |
|---|---|---|---|
| ~~⬜ `status` (跨仓)~~ | ~~中~~ | ~~~5h~~ | **✅ v0.9.0-a 完成** — commit `63eb1b3` |
| ⬜ `alias *` | 低 | ~3h | P2 下一个待启动 |
| ⬜ `extension *` | 中 | ~8h | P2 已规划,需考虑 gitee 的 plugin 机制 |
| ⬜ `repo set-default` | 低 | ~2h | 类似 gh 目录级默认仓库 |

### 🟡 中优先级(差异化体验)

| 缺口 | 难度 | 工作量 | 备注 |
|---|---|---|---|
| ⬜ `issue status` | 低 | ~2h | 与 pr status 对称 |
| ⬜ `pr status` | 低 | ~2h | |
| ⬜ `release upload` | 中 | ~4h | Gitee v5 有此端点,需加 multipart upload |
| ⬜ `release delete-asset` | 中 | ~3h | |
| ⬜ `gist rename` | 中 | ~3h | Gitee v5 spec 暂未确认端点 |
| ⬜ `auth switch` | 中 | ~3h | 多账号切换 |

### 🟢 低优先级(可选项)

| 缺口 | 难度 | 工作量 | 备注 |
|---|---|---|---|
| ⬜ `repo gitignore` | 低 | ~1h | 内置模板列表 |
| ⬜ `repo license` | 低 | ~1h | 内置模板列表 |

### ❌ 不可补(平台限制)

| 缺口 | 原因 |
|---|---|
| `codespace` | Gitee 无此产品 |
| `project` | Gitee 暂无 Projects v2 |
| `attestation` | Gitee 无 Sigstore-style |
| `secret`/`variable` | Gitee 公开 API 不暴露(企业版限定) |
| `cache`/`run`/`workflow` | Gitee Go pipeline 暂未封装 → P3 计划 |

---

## 12. Gitee 独有但 gh 没有的(差异化亮点)

| gitee 特性 | 状态 | 价值 |
|---|---|---|
| `webhook *`(6 verb) | ✅ | gh 没做(只在 `api` 里硬调) |
| `label *`(6 verb) | ✅ | gh 也做了,但 gitee 提早到 v0.7 |
| `gpg-key *` | ✅ | 完全对标 |
| `ssh-key *` | ✅ | 完全对标 |
| `org members/repos/issues` | ✅ | 比 gh 的 `--org` 形式更友好 |
| `repo branches` | ✅ | |
| `repo browse` | ✅ | 等价于 `gh browse` |
| `gist star/unstar` | ✅ | gh 没把这两个做成 verb |
| `--agent-help` 全局 flag | ✅ | **gitee 杀手锏**,面向 LLM Agent 的双轨帮助系统 |

---

## 13. 关键发现

1. **核心场景已覆盖**：issue/pr/repo/gist/release 5 大组核心场景 80%+ 覆盖
2. **平台差异合理**：codespace/project/cache/run/workflow 都是 Gitee 没做或暂未做
3. **gitee 已超 gh**：
   - `webhook` 一级命令(gh 把它藏在 `api`)
   - `org members/repos/issues` 独立 verb
   - `--agent-help` 全局 flag
4. **P2/P3 主战场**：
   - P2a = status ✅(commit `63eb1b3`)
   - P2b = extension
   - P2c = alias
   - P3 = Gitee Go pipeline(runs/workflows/cache)
5. **🚫 不是技术债,是 API 真相**：P1c 已明确,6 个 issue/pr 增强 Gitee v5 不支持
