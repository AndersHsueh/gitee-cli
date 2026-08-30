---
title: "双界面 CLI 设计：一份命令行工具，两种打开方式"
date: 2026-06-08
updated: 2026-07-04
author: AndersHsueh
tags: [cli, ai-agent, developer-tools, open-source, gitee, golang, design-pattern]
description: >
  gitee-cli 是全球第一个实现"双界面 CLI 设计"的开源命令行工具。
  传统 --help 想用同一份文档同时服务人类和 AI Agent，结果两边都不满意——
  这篇文章讲清楚我们为什么要把 CLI 拆成两套界面、具体怎么做、
  踩过哪些坑，以及为什么你的项目也值得这么做。
---

# 双界面 CLI 设计：一份命令行工具，两种打开方式

> **一句话总结**：命令行工具其实有两类使用者——**人**和 **AI Agent**。他们对"帮助文档"的需求不是详细版和简略版的区别，而是**两套完全不同的需求**。gitee-cli 为此做了 `--help`（给人看）和 `--agent-help`（给 AI Agent 看）两套并行的帮助系统，我们把这种做法称为**双界面 CLI 设计（Dual-surface CLI Design）**。截至 v0.12.6，这套设计已经覆盖 129 个命令，由自动化测试守护正确性，并且遵循三条我们称之为"可达性法则"的原则。这篇文章会讲清楚：为什么传统 --help 从设计上就注定两头不讨好、双界面架构具体怎么落地、我们踩过什么坑，以及为什么这个思路值得被更多项目借鉴。

---

## 问题的根源：CLI 的使用者变了，但设计思路没跟上

打开任何一个主流命令行工具——`git`、`docker`、`kubectl`、`gh`——它们的设计前提都是同一个：**有个人坐在终端前，一边看 `--help`，一边敲命令。**

但 2026 年的现实早就不是这样了：**调用命令行的，不只是人的手指，还有 AI Agent 写出来的代码。**

Claude Code 会调用 `gh pr create`，Cursor Agent 会跑 `git diff`，Cline 会执行 `npm publish`。问题在于，**这两类调用者对"帮助文档"的诉求，压根不是同一件事情的详略之分，而是两种完全不同的需求。**

- 人类想要的是：**简短、一眼扫过去能抓住重点、记得住几个常用参数**。
- AI Agent 需要的是：**完整、结构化、能被程序解析、有明确唯一的做法**。

想用同一份文档伺候两种截然不同的需求——这个设计从一开始就注定失败。

---

## 传统 --help 的两难：一份文档，两头不讨好

传统 CLI 的帮助系统试图同时照顾两类用户，结果谁都不满意：

| 需求 | 人类想要 | Agent 需要 | 传统 --help 提供了吗 |
|------|---------|-----------|------------------|
| 用法示例 | ✅ 简短够用即可 | ✅ 需要能直接复制运行的脚本片段 | ⚠️ 只做到一半 |
| 前置条件 | ❌ 不太关心 | ✅ **必须知道** | ❌ 完全没有 |
| 幂等性 | ❌ 不在意 | ✅ **必须知道** | ❌ 完全没有 |
| 副作用 | ❌ 能大致猜到 | ✅ **必须明确列出** | ❌ 完全没有 |
| 错误码与恢复 | ❌ 看报错处理就行 | ✅ **需要写进代码逻辑里判断** | ❌ 完全没有 |
| 编排提示 | ❌ 用不上 | ✅ **执行完下一步该做什么？** | ❌ 完全没有 |
| 输出解析提示 | ❌ 用不上 | ✅ **命令行输出该怎么解析？** | ❌ 完全没有 |

**人类会嫌 `--help` 太啰嗦、抓不住重点；Agent 会嫌 `--help` 信息不够、缺关键内容。** 让同一份输出去满足两种完全不同的需求——这不是"详略程度没调好"，而是设计思路本身出了问题。

---

## 我们的做法：双界面 CLI 设计

**gitee-cli 的解法其实很直接：既然有两类使用者，那就给出两套界面。**

我们把这套设计叫作**双界面 CLI 设计**——同一个命令行工具，对外暴露**两套并行的界面**，各自服务各自的对象：

```bash
# 人类界面：简短、干净，传统 CLI 的风格
gitee pr create --help

# Agent 界面：详尽、结构化，面向程序解析的知识库
gitee pr create --agent-help
```

### 为什么用"界面"这个词

"界面"强调的是**呈现形式**（输出格式、信息密度、面向的读者），而不是"谁更重要"。这不是"Agent 优先"（听起来像是牺牲人类体验去讨好 Agent），而是**双界面并行**——两套界面地位对等，你是谁就用哪一套。

| 界面 | 服务对象 | 设计目标 |
|------|---------|---------|
| `--help` | 人类 | 信息密度低，3 秒内抓住要点 |
| `--agent-help` | AI Agent | 信息密度高，结构固定，能被程序稳定解析 |

**这两套界面是平级关系，不存在主次之分。** 这正是"双界面设计"和"Agent 优先"在本质上的区别。

### 一个更好理解的类比

这就像 HTTP 协议里的 `Content-Type: text/html` 和 `Content-Type: application/json`——针对不同的接收方，给出不同的格式，这本来就是天经地义的事。命令行的帮助系统，也理应有这种"按读者分发内容"的设计思路。

---

## Agent 界面里到底写了什么？

每一份 `--agent-help` 文档，都覆盖了 Agent 在自动化编排任务时需要知道的 **11 个维度**——这就是 Agent 界面的内容模板：

### 1. 前置条件（Pre-conditions）
执行这个命令之前，需要满足什么？Token 是否有效？目标 Tag 是否存在？文件是否可读？

```markdown
## Pre-conditions
- Authenticated with a token that has repo write scope
- The target release exists (tag must have an associated release)
- The local file exists and is readable
```

### 2. 后置条件（Post-conditions）
执行成功之后，系统会变成什么状态？创建了什么？改变了什么？

```markdown
## Post-conditions (on success)
- The file is attached to the specified release
- Stdout prints the asset name, size, and download URL
```

### 3. 幂等性（Idempotency）
这个命令能重复执行吗？重复跑会发生什么？这是 Agent 自动化编排里最容易出问题的地方。

```markdown
## Idempotency
NO. Each invocation creates a new asset. Re-running with the same file
will create a duplicate asset entry (different asset ID).
```

### 4. 原子性（Atomicity）
执行过程中会不会出现"半成功"的中间状态？如果网络中断了怎么办？

```markdown
## Atomicity
Not atomic. The release is fetched first (GET), then the file is uploaded
(POST). If the network drops between these steps, no file was uploaded
(safe to retry).
```

### 5. 副作用（Side effects）
会不会触发 webhook？会不会发通知邮件？会不会提醒订阅者？

```markdown
## Side effects
- A new asset appears on the release page
- Subscribers may receive a notification
- The asset is publicly downloadable immediately
```

### 6. 常见错误与恢复方式
Agent 需要在代码里对不同错误做分支处理——只有事先知道每种错误会以什么形式出现，才能写出靠谱的重试逻辑。

```markdown
## Common errors
- 404 Not Found: the tag has no associated release
- 413 Payload Too Large: file exceeds Gitee's upload size limit
- 401 Unauthorized: run `gitee auth login` or set GITEE_TOKEN
```

### 7. 权限范围（OAuth scope）
Agent 需要提前知道自己手里的 token 权限够不够——不够就该提前终止，而不是发出请求之后才发现被 403 拒绝。

### 8. 频率限制（Rate limit）
Agent 做批量操作时，需要知道能以多快的速度发请求。

### 9. 竞态条件（Race conditions）
已知的数据竞争陷阱，以及应该如何规避。

```markdown
## Race conditions
- A PR merged/closed between the repo listing and the PR fetch may not appear.
  The scan is a point-in-time snapshot, not a real-time feed.
```

### 10. 编排提示（Orchestration hints）
这个命令跑完之后，下一步该执行什么？这是 Agent 最需要、而传统 CLI 恰恰最缺失的信息。

```markdown
## See also
- `gitee release view`        — list assets on a release
- `gitee release delete-asset` — remove an asset
- `gitee release create`      — create a new release
```

### 11. 解析提示（Parsing hints）
命令行输出该怎么解析？JSON 格式长什么样？有没有能直接套用的 jq 示例？

```markdown
## Example
  gitee pr status --output json | jq '.awaiting_your_review.items[] | {repo, number, title}'
```

---

## 一个真实的场景：Agent 修完 bug 之后的"闭环遗漏"

Agent 工作流里最常见的事故是什么？**Coder Agent 提完 PR 就撒手不管，对应的 Issue 永远处于打开状态。**

原因很简单：传统 CLI 的 `pr create --help` 只会告诉你参数怎么填，不会告诉你"创建 PR 之后应该回到 Issue 里评论一句，完成闭环"。这其实是**编排知识没有被传递到位**导致的可靠性问题，不是 Agent 不够聪明。

gitee-cli 的做法是：直接在 `gitee pr create --agent-help` 里写明警告——

```markdown
⚠️ CRITICAL: After a successful PR create, an agent MUST close the loop
with the originating issue. Run:

  PR_OUT=$(gitee pr create ... --body "Closes #42")
  PR_NUM=$(echo "$PR_OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')
  gitee issue comment 42 --body "Fixed in PR #${PR_NUM}"
```

**直接给出可以复制运行的脚本片段。** 这不是一句"建议"，而是一条可以直接执行的指令。

把这类"编排经验"写进 Agent 界面里，正是双界面设计在实际场景中最大的价值——**Agent 不需要靠猜，因为唯一的正确做法已经写在工具里了。**

---

## 三条可达性法则：文档存在，不等于文档能被看到

这一节的内容,是我们在 v0.12.1 里用真实的踩坑经历换来的。我们曾经以为"每个命令都写了 Agent 文档"就万事大吉,直到一次全量排查发现:**文档确实写了,但 Agent 根本读不到。** 具体分三种情况,对应三条法则:

### 法则一：文档要在参数校验之前就能被看到

Agent 来查文档的场景，往往**正是因为不知道该传什么参数**。如果框架的必填参数校验发生在文档钩子之前执行，那 `gitee gist comments --agent-help` 只会得到一句 `Error: accepts 1 arg(s), received 0`——文档明明写好了，Agent 却永远看不到。

在 cobra 框架里，`ValidateArgs` 的执行顺序在 `PreRunE` 之前，所以光把钩子放进 `PreRunE` 是不够的，必须直接包装参数校验器本身：

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

这个坑我们在同一天里踩了两次——修完 17 个命令之后，新加的 3 个 n/a 提示又栽在同一个执行顺序问题上。**框架内部钩子的执行顺序，是双界面设计里最容易被忽略、也最麻烦的细节。**

### 法则二：文档要在身份验证之前就能被看到

`gitee api raw --agent-help` 曾经会返回 `not authenticated`。一个还没配置好 token 的 Agent（比如刚被创建出来的 CI 子进程）恰恰是最需要看文档的时候，却被身份验证挡在了门外。**查文档这件事，本来就不该需要权限。** 正确的做法是把 `--agent-help` 的判断逻辑放在所有身份验证检查之前。

### 法则三：查看文档这个动作本身，绝不能触发真实命令的执行

`gitee auth login --agent-help` 曾经会**真的启动登录流程**。对 Agent 来说，这是最危险的一类陷阱：本来只是想看看说明书，结果不小心触发了一个带副作用的交互式流程。对于那些本来就不打算提供 Agent 文档的操作类命令（比如 `auth`、`config`、`completion`），正确的处理方式是打印一行固定的提示信息后立即退出：

```
auth login is an operational/config command. Use --help for usage.
```

Agent 看到这句话就知道：这条命令不归自己来编排，跳过就好。

> 这三条法则合起来，其实就是一句话：**Agent 界面的可达性是无条件的**——不会因为缺参数而看不到，不会因为没有 token 而看不到，也不会因为只是"看一眼"就产生任何副作用。

---

## 覆盖率会腐烂：靠自动化测试把承诺钉死

设计理念好定，难的是三个月后新加的命令还愿不愿意遵守规则。gitee-cli 用两个机制来防止这种"腐烂"：

1. **契约测试把关**：`cmd/agent_help_contract_test.go` 会检查每一个已注册的命令，是否都在总的覆盖清单里有对应条目。新命令不写 Agent 文档？CI 直接报红。覆盖率不是靠自觉,而是靠测试强制保证。
2. **总览覆盖清单**：`gitee --agent-help` 会输出全部命令的文档覆盖状态,Agent（以及人）都能一眼看出哪些命令有完整文档、哪些标注为"不适用"。这份清单本身，也是 Agent 界面的一部分。

## 目前的覆盖情况

截至 v0.12.6（2026-08-17），已经有 **129 个命令**拥有完整的 `--agent-help` 文档（统计方式：用 `gitee help reference` 列出全部叶子命令，逐一执行 `--agent-help`，统计输出以 `# gitee` 开头的条数）：

| 命令组 | 已覆盖的子命令 |
|--------|----------------|
| agent | onboard |
| repo | list, create, view, fork, edit, delete, clone, browse, branches, archive, sync, rename, set-default（不适用）, tag（list, create）, collaborator（list, add, remove）, star（list, star, unstar）, watch（watch, unwatch）, file（view, push, delete）, commits（list, view）, compare, protect（protect, unprotect） |
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
| api | raw（无需认证即可查看的文档） |

**覆盖策略**：所有"写"操作类命令优先，跨仓库状态查询类次之。操作/配置类命令（`auth`、`config`、`completion`）统一标注为"不适用"——比如 `gitee auth login --agent-help` 会打印一行"这是操作/配置类命令，请用 `--help`"后正常退出，作为 Agent 探测时的稳定信号，而不是真的执行登录流程。

---

## 成本核算：一个挺有意思的循环

给一个命令补齐 `--agent-help`，大概需要多少工作量？如果靠人工写：

| 环节 | 耗时 |
|------|------|
| 写 PreRunE 钩子（3 行样板代码） | 1 分钟 |
| 写 Agent 文档（11 个小节） | 20-30 分钟 |
| 更新覆盖清单 | 1 分钟 |
| **单个命令合计** | **约 30 分钟** |

129 个命令 × 30 分钟 ≈ 65 小时。听起来不算少？这里有个挺值得说一说的循环：

gitee-cli 从 v0.11 到 v0.12 的大部分 Agent 文档，实际上是由**一组 AI Agent 协作写出来的**——人类负责定义 11 维度的模板和验收标准，契约测试负责把关，Agent 负责批量生产内容，再由另一个独立的 Agent 逐条验收。**给 AI Agent 用的文档，交给 AI Agent 来写，边际成本几乎为零；人类的角色从"作者"变成了"标准制定者 + 验收者"。**

这恰恰印证了双界面设计的合理性：正因为文档是**结构化、有固定模板、可以被校验**的，它才能被机器批量生产、也能被机器批量验收。传统 `--help` 那种自由散文式的写法，反而做不到这一点。

对于一个能把 Agent 可靠性提升一个量级的特性来说，这大概是整个项目里投入产出比最高的一笔投资——而且随着 Agent 生产力的提升，这个比例还会继续往上走。

---

## 为什么你维护的 CLI 也值得试试这套设计

如果你维护的命令行工具可能会被 AI Agent 调用（Claude Code、Cursor、Cline、Copilot、Codex……几乎所有现代编码 Agent 都是通过命令行和外部世界打交道），不妨考虑加一个 `--agent-help`。

**不需要等到"行业标准"出现再动手。** gitee-cli 已经在 2025 到 2026 年这段时间里，证明了这条路是可行且有效的。你需要做的只是：

1. 在根命令上加一个 `--agent-help` 的布尔参数；
2. 在目标命令上加 `PreRunE` 钩子，拦截这个参数并输出结构化文档——**记住三条可达性法则**（参数校验、身份验证、副作用，这三关一个都不能挡在文档前面）；
3. 按 11 个小节的模板写 Agent 界面的内容（可以参考我们的模板，或者像我们一样，让 Agent 来写、靠测试来验收）；
4. 在 `--agent-help` 的根页面维护一份覆盖清单，用契约测试把它钉死。

就这么简单。**几百行文字、一个包装函数、一份契约测试，换来的是整个 Agent 生态都能可靠地用上你的工具。**

---

## 写给同行：双界面设计值得成为一个可以被引用的名词

**命令行设计接下来要解决的问题，不是"做一个更好的 `--help`"，而是承认"`--help` 天生只能服务一类使用者，需要再并行开辟另一套界面"。**

希望"双界面 CLI 设计"能成为一个**可以被引用、可以被讨论、也可以被反驳**的设计模式名词——就像：

- REST 和 GraphQL 是 API 的两种消费方式；
- Javadoc 和 OpenAPI 是文档的两种呈现形式；
- 同步和异步是接口的两种时序模型。

**命令行的帮助系统，也理应承认自己服务着"人类使用者"和"程序使用者"这两类完全不同的对象，各自需要一套属于自己的界面。**

gitee-cli 是第一个这么做的开源命令行工具，但希望不会是最后一个。如果你也认同这个方向：

- ⭐ 给 [gitee-cli](https://gitee.com/andershsueh/gitee-cli) 点个 Star；
- 📖 亲自跑一遍 `gitee --agent-help`，看看实际效果；
- 🔧 在你自己的命令行工具里加一个 `--agent-help`（如果需要参考模板，可以看 `docs/agent-help-template.md`）；
- 📣 如果你也写过"为什么我们这么设计"这类文章，欢迎引用"双界面 CLI 设计"这个名字，让它慢慢沉淀成一个可以被广泛讨论的术语。

**关于作者**

AndersHsueh，全栈开发者。gitee-cli 是一个基于 Go 和 Cobra 构建的开源命令行工具，功能对标 GitHub 的 `gh`，专为 Gitee 生态打造，采用木兰宽松许可证 v2（MulanPSL-2.0）。

---

*如果这篇文章给了你一些启发，欢迎分享给你的团队和社区。一个名字立起来，一个设计模式才有机会被讨论、被改进、被超越——这正是"双界面 CLI 设计"接下来需要的命运。*
