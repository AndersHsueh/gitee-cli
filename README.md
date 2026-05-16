# gitee-cli

> Gitee 官方命令行工具 — 在终端中完整操作 Gitee，功能对标 GitHub `gh`

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()
[![Version](https://img.shields.io/badge/version-v0.3.4-orange.svg)]()

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

## License

[Apache 2.0](LICENSE)

---

## Buy Me Coffee

如果你觉得这个项目对你有帮助，可以请我喝杯咖啡：

![Buy Me Coffee](./etc/BuyMeCoffe.jpg)
