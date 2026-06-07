---
title: "Agent-First CLI: 为什么你的命令行工具需要一个 --agent-help 标志"
date: 2026-06-08
author: AndersHsueh
tags: [cli, ai-agent, developer-tools, open-source, gitee, golang]
description: >
  gitee-cli 是全球第一个实现双轨帮助系统的开源 CLI 工具。传统 --help 
  在人类和 AI Agent 之间做了错误妥协——这篇文章解释了我们为什么拆分它、
  怎么做、以及你的项目为什么也应该这么做。
---

# Agent-First CLI: 为什么你的命令行工具需要一个 --agent-help 标志

> **TL;DR**: CLI 工具的真实用户越来越是 AI Agent，而不是人。我们在 gitee-cli 里拆出了两套并行帮助系统（`--help` 给人类，`--agent-help` 给 Agent），实现了全球首个 "Agent-First" CLI 设计。这篇文章解释为什么传统 CLI 的 --help 是错误妥协、双轨系统怎么做、以及你的项目为什么也应该这么做。

---

## 问题：CLI 的用户变了，但 CLI 的设计没变

打开任何一个主流 CLI 工具——`git`、`docker`、`kubectl`、`gh`。它们的设计假设是什么？**用户在终端前，一边看 --help 一边敲命令。**

但 2026 年的现实是：**越来越多的 CLI 调用不是来自人类的手指，而是来自 AI Agent 的代码。**

Claude Code 调用 `gh pr create`。Cursor Agent 调用 `git diff`。Cline 调用 `npm publish`。这些 Agent 不是在"读"帮助文本——它们是在**解析**知识：这个命令的前置条件是什么？重复执行安全吗？会有什么副作用？执行完下一步该做什么？

传统 `--help` 对这些问题的回答是：**零。**

---

## 传统 --help 的错误妥协

传统 CLI 的帮助系统想同时讨好两类用户，结果两边都不爽：

| 需求 | 人类要的 | Agent 要的 | 传统 --help 给了吗 |
|------|---------|-----------|------------------|
| 用法示例 | ✅ 简短够用 | ✅ 需要可复制的 shell 片段 | ⚠️ 部分 |
| 前置条件 | ❌ 不需要 | ✅ **必须知道** | ❌ 没有 |
| 幂等性 | ❌ 不关心 | ✅ **必须知道** | ❌ 没有 |
| 副作用 | ❌ 能猜到 | ✅ **必须明确列出** | ❌ 没有 |
| 错误码+恢复 | ❌ 看报错就行 | ✅ **需要在代码里分支处理** | ❌ 没有 |
| 编排提示 | ❌ 不需要 | ✅ **下一步该跑什么？** | ❌ 没有 |
| JSON 解析提示 | ❌ 不需要 | ✅ **stdout 怎么 parse？** | ❌ 没有 |

**人类嫌 --help 啰嗦、抓不住重点。Agent 嫌 --help 不够、缺关键信息。** 同一个输出格式要服务两个完全不同需求的受众——这是根本性的设计失败。

---

## 解法：不要妥协，拆成两套

gitee-cli 的做法很简单：**承认两类用户，用两套文档分别服务。**

```bash
# 人类版：短、干净、传统 CLI 风格
gitee pr create --help

# Agent 版：长、结构化、面向机器的知识
gitee pr create --agent-help
```

这不是一个 hack，而是**基础设施层的设计决策**。就像 HTTP 有 `Content-Type: text/html` 和 `Content-Type: application/json`——给不同的消费者不同的格式，天经地义。

---

## --agent-help 里有什么？

每一份 `--agent-help` 文档都覆盖了 Agent 在编排工作流时需要的 11 个维度：

### 1. Pre-conditions（前置条件）
执行前必须满足什么？Token 有效吗？Tag 存在吗？文件可读吗？

```markdown
## Pre-conditions
- Authenticated with a token that has repo write scope
- The target release exists (tag must have an associated release)
- The local file exists and is readable
```

### 2. Post-conditions（后置条件）
执行成功后系统处于什么状态？什么被创建了？什么被改变了？

```markdown
## Post-conditions (on success)
- The file is attached to the specified release
- Stdout prints the asset name, size, and download URL
```

### 3. Idempotency（幂等性）
能重跑吗？重复执行会发生什么？这是 Agent 编排最常出错的地方。

```markdown
## Idempotency
NO. Each invocation creates a new asset. Re-running with the same file
will create a duplicate asset entry (different asset ID).
```

### 4. Atomicity（原子性）
有没有半成功状态？网络断了怎么办？

```markdown
## Atomicity
Not atomic. The release is fetched first (GET), then the file is uploaded
(POST). If the network drops between these steps, no file was uploaded
(safe to retry).
```

### 5. Side effects（副作用）
会触发 webhook 吗？会发邮件吗？会通知订阅者吗？

```markdown
## Side effects
- A new asset appears on the release page
- Subscribers may receive a notification
- The asset is publicly downloadable immediately
```

### 6. Common errors + recovery（常见错误 + 恢复方式）
Agent 需要在代码里做错误分支——知道什么错误会以什么形式出现，才能写好 retry 逻辑。

```markdown
## Common errors
- 404 Not Found: the tag has no associated release
- 413 Payload Too Large: file exceeds Gitee's upload size limit
- 401 Unauthorized: run `gitee auth login` or set GITEE_TOKEN
```

### 7. OAuth scope（权限范围）
Agent 需要知道自己的 token 权限够不够——不够就提前终止，而不是发了请求才发现 403。

### 8. Rate limit（频率限制）
Agent 做批量操作时需要知道能发多快。

### 9. Race conditions（竞态条件）
已知的数据竞争陷阱和规避方法。

```markdown
## Race conditions
- A PR merged/closed between the repo listing and the PR fetch may not appear.
  The scan is a point-in-time snapshot, not a real-time feed.
```

### 10. Orchestration hints（编排提示）
Agent 跑完这个命令后，下一步该跑什么？这是 Agent 最需要但传统 CLI 最少给的信息。

```markdown
## See also
- `gitee release view`        — list assets on a release
- `gitee release delete-asset` — remove an asset
- `gitee release create`      — create a new release
```

### 11. Parsing hints（解析提示）
stdout 怎么解析？JSON 输出格式是什么？有哪些 jq 示例可以直接用？

```markdown
## Example
  gitee pr status --output json | jq '.awaiting_your_review.items[] | {repo, number, title}'
```

---

## 真实场景：Agent 修完 bug 后的 "闭环遗漏症"

最常见的 Agent 工作流事故是什么？**Coder-Agent 提了 PR 就不管了，Issue 永远 open 着。**

因为传统 CLI 的 `pr create --help` 只会告诉你 flag 怎么用，不会告诉你"创建 PR 后应该回 comment 到 issue 完成闭环"。这是一个**编排知识缺失**导致的可靠性问题，不是 Agent 不够聪明。

gitee-cli 的做法：在 `gitee pr create --agent-help` 里直接标出警告：

```markdown
⚠️ CRITICAL: After a successful PR create, an agent MUST close the loop
with the originating issue. Run:

  PR_OUT=$(gitee pr create ... --body "Closes #42")
  PR_NUM=$(echo "$PR_OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')
  gitee issue comment 42 --body "Fixed in PR #${PR_NUM}"
```

**直接给 Agent 可复制的 shell 片段。** 不是"建议"，是"可执行的指令"。

---

## 当前覆盖

截至 v0.10.0（2026-06-08），共 **42 个 verb** 已获得完整 `--agent-help`：

| 命令组 | 已有 Agent Help |
|--------|----------------|
| repo | archive, sync, rename |
| issue | create, comments, status |
| pr | create, comment, checkout, ready, edit, checks, status |
| gist | clone |
| ssh-key | list, add, delete |
| webhook | list, create, view, edit, delete, test |
| release | list, view, create, edit, delete, download, upload, delete-asset |
| label | list, view, create, edit, delete, clone |
| status | ✅ |
| extension | list, install, remove, upgrade, create |
| alias | set, list, delete |
| search | code, commits |

**覆盖策略**: 所有"写"命令优先，跨仓状态查询次之。Operational/config 类命令（auth、config、completion）标注为 `n/a`。

---

## 成本分析

加一个命令的 `--agent-help` 需要多少工作量？

| 环节 | 时间 |
|------|------|
| 写 PreRunE hook（3 行样板代码） | 1 分钟 |
| 写 Agent 文档（Pre/post-conditions 等 11 个 section） | 20-30 分钟 |
| 更新 coverage matrix | 1 分钟 |
| **每个命令合计** | **~30 分钟** |

gitee-cli 当前有 42 个带 agent-help 的 verb。如果每个平均 30 分钟，**总投入约 21 小时**——不到三个工作日。

对于一个能让 Agent 可靠性提升一个量级的特性来说，这是整个项目里 ROI 最高的投资之一。

---

## 为什么你的 CLI 也需要 --agent-help

如果你维护的 CLI 工具可能被 AI Agent 调用（Claude Code、Cursor、Cline、Copilot、Codex……几乎所有现代编码 Agent 都在通过 CLI 和世界交互），考虑加入 `--agent-help`。

**不需要等"行业标准"出现。** gitee-cli 已经在 2025 年证明了这件事可行且有效。你只需要：

1. 在根命令上加一个 `--agent-help` bool flag
2. 在目标命令上加 `PreRunE` 钩子，拦截 flag 并输出结构化文档
3. 写 11 个 section 的文档（用我们的模板，20 分钟一个命令）
4. 在 `--agent-help` 根页面维护一张覆盖矩阵

就这么简单。**几百行文本，几个小时的投入，换来的是一整个 Agent 生态能可靠地使用你的工具。**

---

## 致开发者社区

**CLI 设计的未来不是"更好的 --help"，而是"两套帮助系统并行"。**

就像 API 设计有 REST 和 GraphQL 两种消费模式、文档有 Javadoc 和 OpenAPI 两种展示方式——CLI 帮助系统也应该区分"人类消费者"和"机器消费者"。

gitee-cli 是第一个这么做的开源 CLI 工具，但它不应该是最后一个。

如果你认同这个方向：
- ⭐ 给 [gitee-cli](https://gitee.com/andershsueh/gitee-cli) 点个 Star
- 📖 看看 `gitee --agent-help` 的实际效果
- 🔧 在你的 CLI 里试试 `--agent-help`
- 📣 向你的团队和社区推广 "Agent-First CLI" 概念

**关于作者**

AndersHsueh，全栈开发者。gitee-cli 是一个基于 Go + Cobra 的开源 CLI 工具，功能对标 GitHub `gh`，专为 Gitee 生态打造。采用木兰宽松许可证 v2 (MulanPSL-2.0)。

---

*如果这篇文章对你有启发，欢迎分享给你的团队和社区。Agent-First CLI 不需要等标准——需要的是共识。每多一个 CLI 支持 --agent-help，整个 AI Agent 生态就可靠一分。*
