# GitHub 有 gh，Gitee 终于也有官方 CLI 了！

程序员们！你们的痛我懂 😤

每天在 Gitee 上：
- 改个 README 要开浏览器
- 看个 PR 状态要切网页
- 建个 Issue 要点点点
- 克隆个仓库要复制粘贴七八次

烦不烦？！

**直到我发现了这个——gitee-cli**

Gitee 官方出的命令行工具，功能直接对标 GitHub 的 `gh`！

---

## 真的有这么神？

来，我给你们演示几个场景 👇

### 场景1：上班前想看看项目的最新动态

以前：打开浏览器 → 登录 Gitee → 找到仓库 → 看一眼
现在：打开终端 → `gitee repo list` → 一目了然

### 场景2：突然想起有个 PR 需要 review

以前：打开网页 → 找到 PR → 看完还要切回去
现在：一条命令 `gitee pr view --web` 直接浏览器打开

### 场景3：想快速创建一个代码片段分享

以前：网页 → 新建 gist → 填标题 → 粘贴内容 → 保存
现在：`gitee gist create --public "Hello World"` 一气呵成

---

## 常用命令一览

```bash
# 仓库操作
gitee repo clone          # 克隆仓库（再也不用复制粘贴地址了！）
gitee repo list           # 查看自己的仓库列表
gitee repo create         # 快速创建新仓库

# Issue 管理
gitee issue list          # 查看 Issue 列表
gitee issue create        # 创建 Issue
gitee issue comment       # 评论 Issue

# PR 操作
gitee pr list             # 查看 PR 列表
gitee pr view             # 查看 PR 详情
gitee pr merge            # 合并 PR

# 代码片段
gitee gist create         # 创建代码片段
gitee gist list           # 查看自己的代码片段

# 组织操作
gitee org list            # 查看组织
gitee org members         # 查看组织成员
```

---

## 为什么我强烈推荐？

**1. 效率提升 10 倍不止 🚀**

再也不用在浏览器和终端之间来回切换了

**2. 键盘流选手的终极形态 ⌨️**

全程键盘操作，手不用离开键盘

**3. 自动化脚本的好帮手 📝**

结合 cron、脚本使用，简直绝配

**4. 官方出品，背书保障 💪**

Gitee 官方团队维护，稳定性有保证

---

## 安装超简单！

```bash
# macOS
brew install gitee-cli

# Linux
curl -fsSL https://gitee.com/gitee-cli/installer | bash

# Windows
scoop install gitee-cli

# 或者直接编译安装
go install github.com/gitee-cli/gitee-cli@latest
```

装完一行 `gitee auth login` 登录，马上就能用！

---

## 配图 Prompt（发给 Seedance）

**[prompt 程序员神器：终端里操作 Gitee 的神器 gitee-cli]**

A modern flat illustration of a cheerful developer sitting at a glowing terminal window. The terminal displays colorful command-line interface with Chinese characters suggesting Git/Gitee operations (repo, issue, pr, gist commands). Floating around the terminal are icons representing Git operations: a folder icon for repos, speech bubble for issues, merge arrows for pull requests, and code bracket icon for gists. The developer has a confident smile, wearing a casual hoodie. Color palette: deep navy blue background (#1a1a2e), vibrant cyan (#00d9ff) and magenta (#ff006e) accent colors for the terminal glow and icons, soft white text. The style is modern tech startup illustration, clean vector art with subtle gradients, no background noise. The composition is centered, suitable for social media cover image. High contrast, eye-catching, professional yet friendly vibe.

---

## 短视频 Prompt（15 秒，发给 Seedance）

**[prompt 15秒短视频：gitee-cli 终端效率神器]**

A cinematic 15-second promotional video for a developer's CLI tool called "gitee-cli". The video flows through these scenes in continuous motion:

**Scene 1 (0-3s) - HOOK:**
A developer at a desk, frustrated, repeatedly switching between browser tabs and terminal. We see the struggle: clicking, scrolling, logging in. Text overlay appears: "每天在 Gitee 上点点点？太累了！"

**Scene 2 (3-7s) - DEMONSTRATION:**
Quick cuts of terminal window opening. Code text typing fast: `gitee repo list` → colorful repo list appears. `gitee pr view --web` → browser opens with PR. `gitee gist create --public "Hello"` → code snippet created instantly. Each command shown with glowing cyan text on dark terminal. Fast-paced editing, 1.5s per cut.

**Scene 3 (7-11s) - REVEAL:**
The developer leans back, smiling, coffee in hand. Terminal displays multiple operations running smoothly. Floating icons: repo folder, issue bubbles, PR merge arrows, code brackets all animate around the terminal in magenta and cyan colors. Text: "Gitee 官方 CLI，效率提升 10 倍！"

**Scene 4 (11-15s) - CTA:**
Clean shot of terminal showing `gitee auth login`. Developer types login command. Final frame: project name "gitee-cli" with Gitee official badge, cyan glow effect. Text: "gitee.com*andershsueh*gitee-cli" fades in. Ends with developer giving thumbs up.

**Visual style:** Dark navy background (#1a1a2e), neon cyan (#00d9ff) and magenta (#ff006e) accent colors, modern tech aesthetic, smooth transitions, energetic pace. The video should feel modern, fast, and professional - like a tech startup product demo. No voiceover needed, just ambient tech sounds and text overlays in Chinese.

GitHub 用户早就用上了 `gh`，现在 Gitee 开发者也有自己的官方 CLI 了！

开源项目地址：**gitee.com*andershsueh*gitee-cli**

目前已经获得了 Gitee 官方的**推荐**认证 ✅

感兴趣的朋友可以去 GitHub 上看看源码，欢迎 star 和贡献代码！

---

你们平时喜欢用 CLI 还是 GUI？
评论区告诉我 👇

#程序员 #命令行工具 #Git #Gitee #效率神器 #开发工具 #编程 #代码管理
