# gitee-cli

> Gitee 官方命令行工具 — 在终端中完整操作 Gitee，功能对标 GitHub `gh`

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.4.1-orange.svg)]()

---

## 功能概览

```
gitee auth       # 认证管理（login / logout / status / token）            ✅ 已实现
gitee issue      # Issue 管理（list / create / view / edit / close / reopen / comment） ✅ 已实现
gitee config     # 配置管理（set / get / list / unset）                  ✅ 已实现
gitee search     # 搜索仓库 / Issue / 用户                             ✅ 已实现
gitee completion # Shell 自动补全（bash / zsh / fish / powershell）  ✅ 已实现
gitee version    # 版本信息                                            ✅ 已实现
gitee repo       # 仓库管理（list / create / view / fork / edit / delete / clone / browse / branches） ✅ 已实现
gitee pr         # Pull Request（list / create / view / merge / review / comments / diff） ✅ 已实现
gitee gist       # 代码片段（list / create / view / edit / delete / star / unstar） ✅ 已实现
gitee org        # 组织管理（list / view / members / repos / issues）    ✅ 已实现
gitee api        # 直接调用 Gitee API v5（GET / POST / PUT / PATCH / DELETE） ✅ 已实现
```

---

## 🆕 `--agent-help`: 给 AI Agent 看的帮助系统 (v0.4.0 新增 · **行业首创**)

> gitee-cli 是**第一个**把 CLI 帮助系统拆成"人类版 + Agent 版"的开源项目。

**为什么需要这个？**

CLI 工具的真实用户越来越多是 **AI Agent**，不是人。但传统的 `--help` 想同时讨好两类用户，结果两边都不爽：
- 人类嫌啰嗦、抓不到重点
- Agent 嫌不够、缺机器需要的关键信息（idempotency / side effects / orchestration hints）

**所以我们做了两套帮助系统并排运行**：

| Flag | 给谁看 | 风格 |
|------|-------|------|
| `--help` | 人类 | 短、干净、符合传统 CLI 习惯 |
| `--agent-help` | AI Agent | 长、结构化、包含编排提示和 pre/post-conditions |

### 用法

```bash
# 看某个命令的 agent 帮助（不执行命令，required flag 缺失也能跑）
gitee issue create --agent-help
gitee pr create --agent-help

# 看所有命令的 agent 帮助覆盖情况
gitee --agent-help
```

### `--agent-help` 比 `--help` 多了什么

- **Pre-conditions** — 执行前需要满足什么
- **Post-conditions** — 执行后系统处于什么状态
- **Idempotency** — 能重跑吗？重复会怎样？
- **Atomicity** — 有没有半成功状态？
- **Side effects** — 会触发哪些 webhook / 邮件 / 通知
- **Common errors** — 常见错误码 + 怎么恢复
- **OAuth scope required**
- **Rate limit**
- **Race conditions** — 已知的数据竞争陷阱和规避方法
- **Next steps (orchestration hints)** — 下一步该跑什么命令（**这是 Agent 最需要的**）
- **Parsing hints** — stdout 怎么解析出 issue/PR 号

### 真实场景：Agent 修完 bug 后忘了 close the loop

最常见的 Agent 流水线事故：**coder-agent 提了 PR 就不管了，issue 永远 open 着**。

`gitee pr create --agent-help` 里专门标了 ⚠️ 警告：

> ⚠️ **CRITICAL**: After a successful PR create, an agent MUST close the loop with the originating issue...

并给出可复制的 shell 片段：

```bash
PR_OUT=$(gitee pr create --repo owner/repo --title "fix: ..." \
  --source fix/x --target main --body "Closes #42")
PR_NUM=$(echo "$PR_OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')
gitee issue comment 42 --body "Fixed in PR #${PR_NUM}"  # 闭环！
```

### 当前覆盖（v0.4.0）

| 命令 | 状态 |
|------|------|
| `gitee --agent-help` | ✅ 可用（命令覆盖总览） |
| `gitee issue create --agent-help` | ✅ 可用 |
| `gitee pr create --agent-help` | ✅ 可用 |
| 其它命令 | 🛣️ 后续版本扩展 |

### 致其他 CLI 维护者

如果你的工具被 AI Agent 调用（Claude Code / Cursor / Cline / ...），考虑加一个 `--agent-help`。这是一次几个小时的工作 + 几百行结构化文本，能让你的工具被 Agent 使用的可靠性提升一个量级。

---

## 快速开始

### 安装

**从源码构建**

```bash
git clone https://gitee.com/andershsueh/gitee-cli.git
cd gitee-cli
go build -o gitee ./cmd/gitee
sudo mv gitee /usr/local/bin/
```

**全局安装（Go install）**

```bash
go install ./cmd/gitee
```

### 认证

```bash
gitee auth login
# 按提示在浏览器中完成授权，Token 自动保存到 ~/.gitee-cli/settings.json
```

也可以直接使用已有 Token：

```bash
echo "your_access_token" | gitee auth login --with-token
```

或在 `~/.gitee-cli/settings.json` 中手动配置（参考 `settings.json.sample`）：

```json
{
  "hosts": {
    "gitee.com": {
      "token": "your_access_token",
      "username": "your_username"
    }
  }
}
```

或使用环境变量：

```bash
export GITEE_TOKEN=your_access_token
```

---

## 使用示例

### 仓库管理

```bash
# 列出我的仓库
gitee repo list

# 列出指定用户的公开仓库
gitee repo list --owner torvalds --visibility public

# 创建仓库
gitee repo create --name my-project --description "My new project" --private

# 克隆仓库
gitee repo clone owner/repo-name

# Fork 仓库
gitee repo fork owner/repo-name

# 在浏览器打开仓库
gitee repo browse owner/repo-name
```

### Issue 管理

```bash
# 列出当前仓库的 open Issue
gitee issue list

# 列出所有状态，按标签过滤
gitee issue list --state all --labels bug,enhancement

# 创建 Issue
gitee issue create --title "发现一个 Bug" --body "复现步骤：..." --labels bug

# 查看 Issue 详情
gitee issue view 42

# 关闭 Issue
gitee issue close 42

# 发表评论
gitee issue comment 42 --body "感谢报告，已修复"
```

### Pull Request

```bash
# 列出所有 open PR
gitee pr list

# 创建 PR
gitee pr create --title "feat: 新增搜索功能" --source feature/search --target main

# 查看 PR diff
gitee pr diff 18

# 合并 PR（squash 方式）
gitee pr merge 18 --squash --delete-branch

# 审查 PR
gitee pr review 18 --approve
gitee pr review 18 --request-changes --body "请补充单元测试"
```

### 搜索

```bash
# 搜索 Go 语言仓库，按 Stars 排序
gitee search repos "web framework" --language go --sort stars

# 搜索用户
gitee search users "张三"

# 搜索 Issue
gitee search issues "内存泄漏" --state open
```

### 原始 API 访问

```bash
# GET 请求
gitee api GET /user
gitee api GET /user/repos?type=owner&per_page=10

# POST 请求（--field 传参）
gitee api POST /user/repos --field name=test --field private=true

# 分页遍历所有结果
gitee api GET /user/repos --paginate

# 查看响应头
gitee api GET /user --include
```

### 配置管理

```bash
gitee config set editor vim         # 设置编辑器
gitee config set protocol ssh       # 使用 SSH 协议 clone
gitee config get protocol           # 读取配置
gitee config list                   # 查看所有配置
gitee config unset editor           # 删除配置项
```

### 认证状态

```bash
gitee auth status          # 查看当前登录状态
gitee auth token           # 显示当前 Token
gitee auth logout          # 登出（--yes 跳过确认）
```

---

## 全局选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `--token string` | Access Token（覆盖配置文件） | 读取配置 |
| `--hostname string` | Gitee 实例地址（企业私有部署） | `gitee.com` |
| `--api-url string` | API 基础 URL | `https://{hostname}/api/v5` |
| `--repo string` | 目标仓库（owner/repo 格式） | 读取 Git 远程 |
| `-o, --output string` | 输出格式：`table` \| `json` \| `csv` | `table` |
| `-P, --paginate` | 自动分页，获取全部结果 | `false` |
| `--limit int` | 结果总数上限 | `0`（不限制） |
| `--page int` | 指定页码 | `1` |
| `--per-page int` | 每页数量（最大 100） | `30` |
| `--verbose` | 打印 HTTP 请求/响应详情 | `false` |
| `--no-color` | 关闭 ANSI 颜色输出 | `false` |
| `--timeout int` | 网络超时秒数 | `30` |

**环境变量：**

| 变量 | 说明 |
|------|------|
| `GITEE_TOKEN` | Access Token（优先级高于配置文件） |
| `GITEE_HOSTNAME` | Gitee 实例地址 |
| `GITEE_OUTPUT` | 默认输出格式 |
| `GITEE_API_URL` | API 根地址 |
| `NO_COLOR` | 禁用颜色（标准） |

---

## Shell 自动补全

```bash
# Bash
echo 'eval "$(gitee completion bash)"' >> ~/.bashrc

# Zsh
gitee completion zsh > "${fpath[1]}/_gitee"

# Fish
gitee completion fish > ~/.config/fish/completions/gitee.fish

# PowerShell
gitee completion powershell >> $PROFILE
```

---

## 企业私有部署

```bash
# 登录企业实例
gitee auth login --hostname your-company.gitee.com

# 或通过环境变量
export GITEE_HOSTNAME=your-company.gitee.com
gitee repo list
```

---

## 开发

详见 [`documents/developer-guide.md`](documents/developer-guide.md)

```bash
# 编译
go build -o gitee ./cmd/gitee

# 测试
go test ./... -race -count=1

# Lint
golangci-lint run ./...

# 功能测试
./test/run.sh
```

### 测试覆盖

| 包 | 测试函数 | 覆盖内容 |
|---|---------|---------|
| `internal/api` | 21 | HTTP 客户端、错误处理、端点构建 |
| `internal/utils` | 4 | 字符串截断、日期处理、仓库名解析 |
| `internal/config` | 5 | 配置路径、加载/保存、偏好设置 |
| `cmd/pr` | 2 | PR 审批事件值、API 端点 |
| `pkg/types` | 2 | PRReviewRequest 序列化/反序列化 |
| **合计** | **34+** | **5 包、7 测试文件** |

### 项目文档

| 文档 | 说明 |
|------|------|
| [`documents/FSD.md`](documents/FSD.md) | 功能规格文档 |
| [`documents/SDD.md`](documents/SDD.md) | 系统设计文档 |
| [`documents/developer-guide.md`](documents/developer-guide.md) | 开发指导文档 |

---

## 参考资料

- [Gitee API v5 文档](https://gitee.com/api/v5/swagger)
- [Gitee OAuth2 文档](https://gitee.com/api/v5/oauth_doc)
- [GitHub gh CLI](https://github.com/cli/cli)（设计参考）
- [GitLab glab CLI](https://github.com/gitlab-org/cli)（设计参考）

---

## 更新日志

### v0.4.1 (2026-06-02)

- **🐛 修复**: `gitee issue list` 和 `gitee pr list` 之前不真正分页——只发一次请求，多页内容会被静默丢弃
  - `--limit` 之前是"每页大小"（错误），现在是"**总数上限**"（与 flag 名一致）
  - `--paginate` 之前存在但 list 命令没接，**现在真正生效**
  - `--per-page` 全局 flag 之前未绑定变量，现在可以真正控制每页大小
- **🆕 新增**: `internal/api.FetchAllPages[T]` 泛型分页辅助函数，所有 list 类命令的"取全部"标准实现
- **测试**: 7 个新的 `FetchAllPages` 测试覆盖单页 / 多页 / 短尾页 / 限额 / 空结果 / 保留现有 query 参数 / 验证错误
- **使用示例**（修复后的推荐用法）：
  ```bash
  # 取所有 open issues（agent 流水线场景）
  gitee issue list --repo owner/repo --paginate --limit 0

  # 取所有 PR
  gitee pr list --repo owner/repo --paginate --limit 0

  # 还是可以单页（行为不变）
  gitee issue list --repo owner/repo
  ```
- **Default 行为不变**: 不加 `--paginate` 时仍只取 1 页 30 条，与之前完全一致

### v0.4.0 (2026-06-02)

- **🆕 新增**: `--agent-help` 全局 flag + 完整的 agent 文档系统（**行业首创**）
  - `gitee --agent-help` 列出所有命令的 agent 帮助覆盖情况
  - `gitee issue create --agent-help` 详细 agent 帮助（idempotency / side effects / orchestration hints / race conditions / 解析示例）
  - `gitee pr create --agent-help` 同上，特别强调 "PR 创建后必须 comment issue 闭环"
  - 在 Cobra 的 `PreRunE` 阶段拦截，**在 required flag 校验之前触发**，agent 即使缺参数也能拿到帮助
- **🆕 新增**: rootCmd 的 `Long` 文案说明 gitee-cli 同时为人类和 Agent 设计
- **测试**: 新增 2 个 agent-help 内容测试（cmd/issue + cmd/pr），覆盖所有承诺的 section

### v0.3.5 (2026-05-22)

- **修复**: PR review 端点 `/reviews` → `/review`（单数），修复 API 404 问题
- **修复**: PR review 事件值规范化（`APPROVE` → `approve` 等）
- **优化**: `APIError` 实现 `Unwrap()` / `Is()` 方法，`IsNotFound` 改用 `errors.As`
- **优化**: 统一 `resolveRepo` 实现到 `utils.ResolveRepo`，消除 4 处重复代码
- **优化**: `pr/diff` 复用 resty 客户端连接池，替代每次新建 `http.Client`
- **优化**: `config.UserHomeDir()` 错误处理，容器环境兼容
- **优化**: 统一 `userResponse` 到 `types.User`，消除重复类型定义
- **优化**: 合并 `search.go` 中三个 `init()` 函数为一个
- **优化**: 移除 `pagination.go` 中未使用的类型
- **新增**: `internal/output` 包，提供统一 `NewTableWriter()`
- **新增**: 7 个测试文件，测试覆盖从 1→5 包，共 34+ 测试函数

---

## License

[Apache 2.0](LICENSE)

---

## Buy Me Coffee

如果你觉得这个项目对你有帮助，可以请我喝杯咖啡：

![Buy Me Coffee](./etc/BuyMeCoffe.jpg)
