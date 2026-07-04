---
title: "Dual-surface CLI Design: 同一份 CLI,长两张脸"
date: 2026-06-08
updated: 2026-07-04
author: AndersHsueh
tags: [cli, ai-agent, developer-tools, open-source, gitee, golang, design-pattern]
description: >
  gitee-cli 是全球第一个实现 Dual-surface CLI Design 的开源 CLI 工具。
  传统 --help 试图用同一份文档同时服务人类和 AI Agent,结果两边都不爽——
  这篇文章解释我们为什么把 CLI 拆成两张脸、怎么做、踩过哪些坑、
  以及你的项目为什么也应该这么做。
---

# Dual-surface CLI Design: 同一份 CLI,长两张脸

> **TL;DR**: CLI 工具有两类用户——**人**和 **AI Agent**——他们对"帮助文档"的需求不是详略版关系,而是**结构上正交**的两套需求。我们在 gitee-cli 里实现了 `--help`(人类)和 `--agent-help`(AI Agent)两套并行的帮助系统,把这套设计叫做 **Dual-surface CLI Design**。截至 v0.12.1,它覆盖 131 个 verb,由 contract test 守护,并遵守三条"可达性法则"。本文解释为什么传统 --help 是错误妥协、双面架构怎么做、我们在真实迭代中踩过什么坑、以及你的项目为什么也应该这么做。

---

## 问题:CLI 的用户变了,但 CLI 的设计没跟上

打开任何主流 CLI 工具——`git`、`docker`、`kubectl`、`gh`。它们的设计假设是什么?**用户在终端前,一边看 --help 一边敲命令。**

但 2026 年的现实是:**调用 CLI 的不再只是人类的手指,还有 AI Agent 的代码。**

Claude Code 调用 `gh pr create`。Cursor Agent 调用 `git diff`。Cline 调用 `npm publish`。**这两类调用者对"帮助文档"的需求不是同一个光谱上的两个点,而是两个正交维度。**

- 人类要的是:**短、能扫、能记住几个常用 flag**。
- AI Agent 要的是:**完整、结构化、能被解析、有唯一正解**。

试图用同一份文档服务两个物种——这是从根上就注定失败的设计。

---

## 传统 --help 的错误妥协:一份文档服务两个物种

传统 CLI 的帮助系统想同时讨好两类用户,结果两边都不爽:

| 需求 | 人类要的 | Agent 要的 | 传统 --help 给了吗 |
|------|---------|-----------|------------------|
| 用法示例 | ✅ 简短够用 | ✅ 需要可复制的 shell 片段 | ⚠️ 部分 |
| 前置条件 | ❌ 不需要 | ✅ **必须知道** | ❌ 没有 |
| 幂等性 | ❌ 不关心 | ✅ **必须知道** | ❌ 没有 |
| 副作用 | ❌ 能猜到 | ✅ **必须明确列出** | ❌ 没有 |
| 错误码+恢复 | ❌ 看报错就行 | ✅ **需要在代码里分支处理** | ❌ 没有 |
| 编排提示 | ❌ 不需要 | ✅ **下一步该跑什么?** | ❌ 没有 |
| JSON 解析提示 | ❌ 不需要 | ✅ **stdout 怎么 parse?** | ❌ 没有 |

**人类嫌 --help 啰嗦、抓不住重点。Agent 嫌 --help 不够、缺关键信息。** 同一个输出格式要服务两个完全不同需求的受众——这是根本性的设计失败,不是"详略度"没调对。

---

## 解法:Dual-surface CLI Design

**gitee-cli 的做法很简单:承认两类用户,长两张脸。**

我们把这种设计叫做 **Dual-surface CLI Design**——同一个 CLI 暴露**两个并行的表面**,各服务各的消费者:

```bash
# Human surface: 短、干净、传统 CLI 风格
gitee pr create --help

# Agent surface: 长、结构化、面向机器的知识
gitee pr create --agent-help
```

### 为什么叫 "surface"?

"Surface" 这个词强调的是**表层形态**(输出格式、信息密度、消费者契约),不是"哪个用户更重要"。我们不是"Agent-First"(听起来像讨好 Agent 忽视人),我们是 **Dual-surface**(听起来像"我有两个面,你用哪个看自己挑")。

| Surface | 给谁 | 设计目标 |
|---------|------|---------|
| `--help` | 人类 | 信息密度低,3 秒能扫到要点 |
| `--agent-help` | AI Agent | 信息密度高,结构稳定,机器可解析 |

**这两张脸地位对等,不是主从关系。** 这是 Dual-surface 和 "Agent-First" 在语义上的根本差异。

### 这个类比帮你记住

就像 HTTP 有 `Content-Type: text/html` 和 `Content-Type: application/json`——给不同的消费者不同的格式,天经地义。CLI 帮助系统也应该有这种"按消费者分发"的设计。

---

## Agent Surface 里有什么?

每份 `--agent-help` 文档都覆盖了 Agent 在编排工作流时需要的 **11 个维度**——这就是 Agent surface 的内容 schema:

### 1. Pre-conditions(前置条件)
执行前必须满足什么?Token 有效吗?Tag 存在吗?文件可读吗?

```markdown
## Pre-conditions
- Authenticated with a token that has repo write scope
- The target release exists (tag must have an associated release)
- The local file exists and is readable
```

### 2. Post-conditions(后置条件)
执行成功后系统处于什么状态?什么被创建了?什么被改变了?

```markdown
## Post-conditions (on success)
- The file is attached to the specified release
- Stdout prints the asset name, size, and download URL
```

### 3. Idempotency(幂等性)
能重跑吗?重复执行会发生什么?这是 Agent 编排最常出错的地方。

```markdown
## Idempotency
NO. Each invocation creates a new asset. Re-running with the same file
will create a duplicate asset entry (different asset ID).
```

### 4. Atomicity(原子性)
有没有半成功状态?网络断了怎么办?

```markdown
## Atomicity
Not atomic. The release is fetched first (GET), then the file is uploaded
(POST). If the network drops between these steps, no file was uploaded
(safe to retry).
```

### 5. Side effects(副作用)
会触发 webhook 吗?会发邮件吗?会通知订阅者吗?

```markdown
## Side effects
- A new asset appears on the release page
- Subscribers may receive a notification
- The asset is publicly downloadable immediately
```

### 6. Common errors + recovery(常见错误 + 恢复方式)
Agent 需要在代码里做错误分支——知道什么错误会以什么形式出现,才能写好 retry 逻辑。

```markdown
## Common errors
- 404 Not Found: the tag has no associated release
- 413 Payload Too Large: file exceeds Gitee's upload size limit
- 401 Unauthorized: run `gitee auth login` or set GITEE_TOKEN
```

### 7. OAuth scope(权限范围)
Agent 需要知道自己的 token 权限够不够——不够就提前终止,而不是发了请求才发现 403。

### 8. Rate limit(频率限制)
Agent 做批量操作时需要知道能发多快。

### 9. Race conditions(竞态条件)
已知的数据竞争陷阱和规避方法。

```markdown
## Race conditions
- A PR merged/closed between the repo listing and the PR fetch may not appear.
  The scan is a point-in-time snapshot, not a real-time feed.
```

### 10. Orchestration hints(编排提示)
Agent 跑完这个命令后,下一步该跑什么?这是 Agent 最需要但传统 CLI 最少给的信息。

```markdown
## See also
- `gitee release view`        — list assets on a release
- `gitee release delete-asset` — remove an asset
- `gitee release create`      — create a new release
```

### 11. Parsing hints(解析提示)
stdout 怎么解析?JSON 输出格式是什么?有哪些 jq 示例可以直接用?

```markdown
## Example
  gitee pr status --output json | jq '.awaiting_your_review.items[] | {repo, number, title}'
```

---

## 真实场景:Agent 修完 bug 后的 "闭环遗漏症"

最常见的 Agent 工作流事故是什么?**Coder-Agent 提了 PR 就不管了,Issue 永远 open 着。**

因为传统 CLI 的 `pr create --help` 只会告诉你 flag 怎么用,不会告诉你"创建 PR 后应该回 comment 到 issue 完成闭环"。这是一个**编排知识缺失**导致的可靠性问题,不是 Agent 不够聪明。

gitee-cli 的做法:在 `gitee pr create --agent-help` 里直接标出警告——

```markdown
⚠️ CRITICAL: After a successful PR create, an agent MUST close the loop
with the originating issue. Run:

  PR_OUT=$(gitee pr create ... --body "Closes #42")
  PR_NUM=$(echo "$PR_OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')
  gitee issue comment 42 --body "Fixed in PR #${PR_NUM}"
```

**直接给 Agent 可复制的 shell 片段。** 不是"建议",是"可执行的指令"。

这种"把编排知识编码进 Agent surface"的做法,是 Dual-surface Design 最大的实战价值——**Agent 不需要猜,你已经把唯一正解塞进工具里了。**

---

## 可达性三法则:文档存在 ≠ 文档可达

这一节是 v0.12.1 用真实事故换来的。我们曾经以为"每个命令都写了 Agent 文档"就够了,直到全量扫描发现:**文档写了,但 Agent 根本够不着。** 三种够不着的方式,对应三条法则:

### 法则一:文档必须在参数校验之前可达

Agent 探测命令用法时,恰恰**因为不知道要传什么参数**才来查文档。如果框架的必填参数校验先于文档钩子执行,`gitee gist comments --agent-help` 会得到 `Error: accepts 1 arg(s), received 0`——文档明明存在,Agent 却永远看不到。

在 cobra 里,`ValidateArgs` 先于 `PreRunE` 执行,所以光把钩子放进 PreRunE 是不够的,必须包装 Args 校验器本身:

```go
// AgentHelpAwareArgs: --agent-help 在场时跳过位置参数校验
func AgentHelpAwareArgs(wrapped cobra.PositionalArgs) cobra.PositionalArgs {
	return func(c *cobra.Command, args []string) error {
		if GlobalAgentHelp {
			return nil
		}
		return wrapped(c, args)
	}
}
```

这个坑我们在同一天里踩了两次——修完 17 个命令之后,新加的 3 个 n/a notice 又栽在同一个执行顺序上。**框架的钩子执行顺序,是 Dual-surface 实现里最阴险的细节。**

### 法则二:文档必须在认证之前可达

`gitee api raw --agent-help` 曾经返回 `not authenticated`。一个还没配 token 的 Agent(比如刚被 spawn 出来的 CI 子进程)最需要文档,却被认证墙挡在门外。**查文档永远不应该需要权限。** 把 agent-help 判断放在一切认证检查之前。

### 法则三:探测文档绝不能执行真命令

`gitee auth login --agent-help` 曾经会**真的开始登录流程**。对 Agent 来说这是最危险的一类陷阱:它只是想看看说明书,结果误触发了带副作用的交互式流程。对于刻意不提供 Agent 文档的 operational 命令(auth/config/completion),正确行为是打印一行稳定的 n/a 信号后立即退出:

```
auth login is an operational/config command. Use --help for usage.
```

Agent 拿到这个信号就知道:这条命令不归我编排,绕开。

> 三条法则合起来就是一句话:**Agent Surface 的可达性是无条件的**——不因缺参数而不可达,不因没 token 而不可达,不因探测本身而产生副作用。

---

## 覆盖会腐烂:用 contract test 把承诺钉死

宣言好写,难的是三个月后新命令还守不守规矩。gitee-cli 用两个机制防腐烂:

1. **Contract test 守门**:`cmd/agent_help_contract_test.go` 断言每个注册的命令都在根覆盖矩阵里有条目。新命令不写 Agent 文档?CI 直接红。覆盖率不是靠自觉,是靠测试。
2. **根覆盖矩阵**:`gitee --agent-help` 输出全部命令的覆盖状态,Agent(和人)一眼看到哪些命令有完整文档、哪些标注 n/a。矩阵本身也是 Agent Surface 的一部分。

## 当前覆盖

截至 v0.12.1(2026-07-04),共 **131 个 verb** 已获得完整 `--agent-help`(实测:`gitee help reference` 列出全部叶子命令,逐个跑 `--agent-help`,统计输出以 `# gitee` 开头的数量):

| 命令组 | 已有 Agent Help |
|--------|----------------|
| agent | onboard |
| repo | list, create, view, fork, edit, delete, clone, browse, branches, archive, sync, rename, set-default (n/a), tag (list, create), collaborator (list, add, remove), star (list, star, unstar), watch (watch, unwatch), file (view, push, delete), commits (list, view), compare, protect (protect, unprotect) |
| issue | list, create, view, edit, close, reopen, comment, comments, status |
| pr | list, create, view, merge, review, close, reopen, diff, comments, comment, checkout, ready, edit, checks, status |
| gist | list, view, create, edit, delete, star, unstar, clone, comments, comment, comment-delete |
| ssh-key | list, add, delete |
| user | view, follow, unfollow, followers, following |
| webhook | list, create, view, edit, delete, test |
| org | list, view, member, repo, issue |
| release | list, view, create, edit, delete, download, upload, delete-asset |
| label | list, view, create, edit, delete, clone |
| milestone | list, view, create, edit, delete |
| notification | list, count, read |
| status | ✅ |
| extension | list, install, remove, upgrade, create |
| alias | set, list, delete |
| search | repos, issues, users, code, commits |
| api | raw (auth-free doc) |

**覆盖策略**:所有"写"命令优先,跨仓状态查询次之。Operational/config 类命令(auth、config、completion)标注为 `n/a` —— `gitee auth login --agent-help` 等会打印一行 `operational/config — use --help` 后 exit 0,作为 Agent 探测时的稳定信号,而不是触发真实登录流。

---

## 成本分析:一个有趣的元循环

加一个命令的 `--agent-help` 需要多少工作量?人工写的话:

| 环节 | 时间 |
|------|------|
| 写 PreRunE hook(3 行样板代码) | 1 分钟 |
| 写 Agent 文档(11 个 section) | 20-30 分钟 |
| 更新 coverage matrix | 1 分钟 |
| **每个命令合计** | **~30 分钟** |

131 个 verb × 30 分钟 ≈ 65 小时。听起来不少?这里有个**元循环**值得说破:

gitee-cli 从 v0.11 到 v0.12 的大部分 Agent 文档,是由 **AI Agent 编队**写的——人类定义 11 维度 schema 和验收标准,contract test 守门,Agent 批量生产,另一个独立 Agent 逐条验收。**给 AI Agent 用的文档,由 AI Agent 来写,边际成本趋近于零;人类的角色从"作者"变成"立法者 + 验收者"。**

这恰恰是 Dual-surface 的自证:因为文档是**结构化、有 schema、可校验**的,它才能被机器生产和机器验收。传统 --help 那种自由散文,反而做不到这一点。

对于一个能让 Agent 可靠性提升一个量级的特性来说,这是整个项目里 ROI 最高的投资——而且这个 ROI 还在随着 Agent 生产力的提升继续上涨。

---

## 为什么你的 CLI 也应该采用 Dual-surface Design

如果你维护的 CLI 工具可能被 AI Agent 调用(Claude Code、Cursor、Cline、Copilot、Codex……几乎所有现代编码 Agent 都在通过 CLI 和世界交互),考虑加入 `--agent-help`。

**不需要等"行业标准"出现。** gitee-cli 已经在 2025–2026 年证明了这件事可行且有效。你只需要:

1. 在根命令上加一个 `--agent-help` bool flag
2. 在目标命令上加 `PreRunE` 钩子,拦截 flag 并输出结构化文档——**并记住可达性三法则**(参数校验、认证、副作用,一个都不能挡在文档前面)
3. 写 11 个 section 的 Agent surface(用我们的模板;或者像我们一样,让 Agent 写、用测试验收)
4. 在 `--agent-help` 根页面维护一张覆盖矩阵,用 contract test 钉死它

就这么简单。**几百行文本、一个包装函数、一个契约测试,换来的是一整个 Agent 生态能可靠地使用你的工具。**

---

## 致开发者社区:Dual-surface 是一个可以引用的设计模式

**CLI 设计的下一步不是"更好的 --help",而是"承认 --help 只能服务一个物种,需要并行另一个 surface"。**

把 **Dual-surface CLI Design** 当作一个**可以引用、可以讨论、可以反对**的设计模式名——它应该有自己的位置,就像:
- REST 和 GraphQL 是 API 的两种消费模式
- Javadoc 和 OpenAPI 是文档的两种展示方式
- 同步和异步是接口的两种时序模型

**CLI 帮助系统也应该承认:有"人类消费者"和"机器消费者"两个物种,它们各自需要一张脸。**

gitee-cli 是第一个这么做的开源 CLI 工具,但它不应该是最后一个。如果你认同这个方向:

- ⭐ 给 [gitee-cli](https://gitee.com/andershsueh/gitee-cli) 点个 Star
- 📖 跑一下 `gitee --agent-help` 看实际效果
- 🔧 在你的 CLI 里加一个 `--agent-help`(模板见 `docs/agent-help-template.md`,如果你要的话)
- 📣 在你写"为什么我们做了 X"的文章时,引用 "Dual-surface CLI Design" 这个名字,让它沉淀成术语

**关于作者**

AndersHsueh,全栈开发者。gitee-cli 是一个基于 Go + Cobra 的开源 CLI 工具,功能对标 GitHub `gh`,专为 Gitee 生态打造。采用木兰宽松许可证 v2 (MulanPSL-2.0)。

---

*如果这篇文章对你有启发,欢迎分享给你的团队和社区。一个名字立起来,一个模式才有机会被讨论、被改进、被超越——这就是 Dual-surface CLI Design 接下来需要的命运。*
