# gitee-cli

> Gitee 官方命令行工具 — 在终端中完整操作 Gitee，功能对标 GitHub `gh`

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.10.1-orange.svg)](https://gitee.com/andershsueh/gitee-cli/releases/latest)
[![Downloads](https://img.shields.io/badge/downloads-releases-blue.svg)](https://gitee.com/andershsueh/gitee-cli/releases/latest)

---

## 功能概览

```
gitee auth       # 认证管理（login / logout / status / token / switch）  ✅ 已实现
gitee issue      # Issue 管理（list / create / view / edit / close / reopen / comment / status） ✅ 已实现
gitee pr         # Pull Request（list / create / view / merge / review / status / diff ...） ✅ 已实现
gitee repo       # 仓库管理（list / create / view / fork / edit / delete / clone / set-default ...） ✅ 已实现
gitee release    # 发布管理（list / create / view / edit / delete / download / upload） ✅ 已实现
gitee gist       # 代码片段（list / create / view / edit / delete / star / unstar / clone） ✅ 已实现
gitee search     # 搜索（repos / issues / users / code / commits）       ✅ 已实现
gitee org        # 组织管理（list / view / members / repos / issues）    ✅ 已实现
gitee webhook    # Webhook 管理（list / create / view / edit / delete / test） ✅ 已实现
gitee label      # 标签管理（list / create / view / edit / delete / clone） ✅ 已实现
gitee ssh-key    # SSH 公钥管理（list / add / delete）                   ✅ 已实现
gitee status     # 跨仓活动摘要（分配给您的 Issue + 待审查的 PR）       ✅ 已实现
gitee alias      # 命令别名（set / list / delete）                       ✅ 已实现
gitee extension  # 扩展管理（install / list / remove / upgrade / create） ✅ 已实现
gitee agent      # AI Agent 脚手架（onboard — 一键生成 agent.md）        ✅ 已实现
gitee config     # 配置管理（set / get / list / unset）                  ✅ 已实现
gitee completion # Shell 自动补全（bash / zsh / fish / powershell）     ✅ 已实现
gitee version    # 版本信息                                             ✅ 已实现
gitee api        # 直接调用 Gitee API v5                                 ✅ 已实现
```

---

## 🆕 Dual-surface CLI Design: 同一份 CLI，长两张脸 (v0.4.0 率先实践 · v0.10.0 完善)

> 📖 完整设计哲学：[Dual-surface CLI Design](docs/philosophy.md)

> gitee-cli 是**全球第一个**实现双面 CLI 架构的开源项目——把帮助系统拆成
> Human Surface 和 Agent Surface，两边各服务各的消费者。

**为什么需要这个？**

CLI 工具的真实用户越来越多是 **AI Agent**（Claude Code、Cursor、Cline...），不是人。
但传统的 `--help` 想同时讨好两类用户，结果两边都不爽：人类嫌啰嗦，Agent 嫌不够。

**所以我们做了两套并行的 Surface**：

| Surface | Flag | 给谁 | 设计目标 |
|---------|------|------|---------|
| Human | `--help` | 人 | 信息密度低，3 秒能扫到要点 |
| Agent | `--agent-help` | AI Agent | 信息密度高，结构稳定，机器可解析（`-o json`） |

### 快速上手

```bash
# Agent 查命令手册（不执行命令，required flag 缺失也能跑）
gitee pr create --agent-help

# Agent 查机器可读版（直接 parse JSON）
gitee pr create --agent-help -o json

# 看所有命令的 agent-help 覆盖矩阵
gitee --agent-help

# 一键为你的仓库生成 Agent 编排模板
gitee agent onboard
```

### Agent Surface 包含 11 个维度

Pre-conditions → Post-conditions → Idempotency → Atomicity → Side effects →
Common errors + recovery → OAuth scope → Rate limit → Race conditions →
**Orchestration hints**（下一步该跑什么）→ **Parsing hints**（stdout 怎么 parse）

### 真实场景：闭环遗漏症

最常见的 Agent 事故：**coder-agent 提了 PR 就不管了，issue 永远 open 着。**

`gitee pr create --agent-help` 直接标出 ⚠️ 警告并给出可复制的 shell 片段：

```bash
PR_OUT=$(gitee pr create --repo owner/repo --title "fix: ..." \
  --source fix/x --target main --body "Closes #42")
PR_NUM=$(echo "$PR_OUT" | grep -oE '#[0-9]+' | head -1 | tr -d '#')
gitee issue comment 42 --body "Fixed in PR #${PR_NUM}"  # 闭环！
```

### 当前覆盖（v0.10.1）

共 **42 个 verb** 已获得完整 `--agent-help`，覆盖 repo/issue/pr/gist/release/label/webhook/ssh-key/status/extension/alias/search 共 12 个命令组。

---

## 📦 下载安装

**不要 Go 环境？直接下编译好的二进制。**

前往 **https://gitee.com/andershsueh/gitee-cli/releases/latest** 下载对应平台：
- `gitee-linux-amd64.tar.gz` — Linux (Intel/AMD 服务器、桌面)
- `gitee-linux-arm64.tar.gz` — Linux (树莓派、Apple Silicon、ARM 服务器)
- `gitee-darwin-amd64.tar.gz` — macOS (Intel Mac)
- `gitee-darwin-arm64.tar.gz` — macOS (M1 / M2 / M3 / M4)
- `gitee-windows-amd64.exe.zip` — Windows 64-bit
- `SHA256SUMS.txt` — 校验和

校验下载完整性：
```bash
sha256sum -c SHA256SUMS.txt   # Linux / macOS
# 或 certutil -hashfile gitee-windows-amd64.exe.zip SHA256  (Windows PowerShell)
```

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

详见 [`docs/zh/developer-guide.md`](docs/zh/developer-guide.md)

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

| 范围 | 覆盖内容 |
|---|---------|
| `internal/api` | HTTP 客户端、错误处理、端点构建、分页 |
| `internal/config` / `internal/utils` / `internal/alias` / `internal/extension` / `internal/atomicfile` | 配置加载/保存、字符串与仓库名解析、别名解析、扩展加载、原子写文件 |
| `cmd/*`（issue / pr / repo / release / label / webhook / ssh-key / gist / status / help_topics …） | 命令行为、API 端点、agent-help section 契约 |
| `pkg/types` | 序列化/反序列化 |
| **合计** | **19 个包 · 53 个测试文件 · 145+ 测试函数** |

### 项目文档

| 文档 | 说明 |
|------|------|
| [`docs/zh/FSD.md`](docs/zh/FSD.md) | 功能规格文档 |
| [`docs/zh/SDD.md`](docs/zh/SDD.md) | 系统设计文档 |
| [`docs/zh/developer-guide.md`](docs/zh/developer-guide.md) | 开发指导文档 |

---

## 参考资料

- [Gitee API v5 文档](https://gitee.com/api/v5/swagger)
- [Gitee OAuth2 文档](https://gitee.com/api/v5/oauth_doc)
- [GitHub gh CLI](https://github.com/cli/cli)（设计参考）
- [GitLab glab CLI](https://github.com/gitlab-org/cli)（设计参考）

---

## 更新日志

### v0.10.1 (2026-06-08)

- **🆕 新增**: `gitee agent onboard` — 一键生成 `.gitee/agent.md`，预置 4 个 Agent 工作流模板
- **🆕 新增**: `--agent-help --output json` — Agent Surface 的机器可读 JSON 输出
- **📖 文档**: `docs/philosophy.md` — Dual-surface CLI Design 设计宣言
- **🔧 优化**: `PrintAgentHelp()` 统一分派函数（40+ PreRunE 钩子统一调用）
- **🐛 修复**: README License badge Apache 2.0 → MulanPSL-2.0（与仓库元数据一致）

### v0.10.0 (2026-06-08)

- **🆕 新增 8 个命令**: `auth switch`, `pr status`, `issue status`, `repo set-default`,
  `search code`, `search commits`, `release upload`, `release delete-asset`
- **🤖 Agent Help 覆盖**: 6 个新命令获得完整 `--agent-help`（11 维度 × 结构化文档）
- **🔧 基础设施**: `UploadFile` multipart 上传, `ReleaseAssetUpload/DeleteAsset` 端点,
  `ActiveHost` config, `GitConfigGet/Set`, `ResolveHostname`, `ResolveCurrentUsername`
- **📊 覆盖率**: ~105 子命令, 可比覆盖率 ~80%, release 命令组 100%

### v0.4.2 (2026-06-02)

- **🆕 新增**: 完整的 release 工作流（gitee-cli 现在用户**不用 Go 环境**也能用）
  - `scripts/build-release.sh` — 5 平台交叉编译（linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64）
  - `scripts/release.sh` — 一键 build + tar.gz/zip 打包 + SHA256SUMS
  - `cmd/version.go`: `const Version` → `var Version`，支持 `-ldflags` 注入（**不再需要手动改源码**来 bump 版本号）
  - `release-guide.md` (gitignored, 本地 only) — 给本地 AI agent fleet 看的发布手册，claude-code / codex / 未来的 agent 看到这个文档就知道怎么 release
  - README 新增 **📦 下载** 段，链接到 `https://gitee.com/andershsueh/gitee-cli/releases/latest`（**以后发新版本 README 一行不用改**）
- **工作流**: 以后切新版本只需要：
  ```bash
  bash scripts/release.sh           # build + package + checksums
  git tag -a vX.Y.Z -m "..."         # 标 tag
  git push origin vX.Y.Z             # push tag
  # 然后去 Gitee 网页 → 新建发布 → 选 tag → 拖 dist/ 里 6 个文件上传
  ```

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

- **🆕 新增**: `--agent-help` 全局 flag + 完整的 agent 文档系统（**一流的双表面设计 / first-class dual-surface design**）
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

[木兰宽松许可证 v2](LICENSE)

---

## Buy Me Coffee

如果你觉得这个项目对你有帮助，可以请我喝杯咖啡：

![Buy Me Coffee](./etc/BuyMeCoffe.jpg)
