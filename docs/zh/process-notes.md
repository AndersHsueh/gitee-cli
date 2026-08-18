# 处理记录 (process-notes)

> 记录 issue 处理过程中的决策、特殊依赖、遗留问题、原则校验结果。
> 由 ALICE 自动化流程维护。

## 已处理 issue

### IK9KXC — issue comment 误报 401 认证失败 + JSON body 被拒收

- **状态**：已提交 PR #13，等待人工审核合并
- **修复 PR**：[#13](https://gitee.com/andershsueh/gitee-cli/pulls/13)
- **测试结果**：cmd/issue/comment_test.go 新增 2 个 httptest 单测，修复前红、修复后绿；全量 `go test ./...` 通过；golangci-lint 0 issues
- **原则校验**：跳过（《产品原则-不做的事》与《不抄清单》文档在当前仓库不存在，用户确认跳过）
- **决策**：
  - Gitee v5 对锁定评论的 issue 返回 HTTP 401 + `{"error":"当前issue已关闭评论功能"}`，CLI 此前只解析 `message` 字段导致真实原因被吞掉。修复：APIError 新增 `GiteeError` 字段（json tag `error`），checkStatus 优先透传 `error` 字段，仅当 body 无具体信息时才回退到默认状态提示。
  - issue comment 端点拒收 JSON body（401），只认 formData。修复：Client 新增 `PostForm` 方法，runComment 改用 formData 发送 body，与 v0.12.6 close/reopen/edit 的 formData 契约保持一致。
- **遗留问题**：issue 描述提到"顺带排查其他 issue 系写端点（issue edit 的标签/指派人子路径等）是否有同样的 JSON→401 问题"，本次 PR 未覆盖，建议后续单独跟进。
- **特殊依赖**：无

### IK923V — 清理死代码包，收敛 repo 解析五入口

- **状态**：已提交 PR #14，等待人工审核合并
- **修复 PR**：[#14](https://gitee.com/andershsueh/gitee-cli/pulls/14)
- **测试结果**：test/test-IK923V-converge-repo-resolve.sh 全绿（死代码包/冗余入口无引用 + 唯一入口三种路径解析）；go build/vet/test ./... 全绿；golangci-lint 0 issues
- **原则校验**：跳过（原则文档不存在，用户确认跳过）
- **决策**：
  - `internal/printer`、`internal/repo` 死代码包已由先前提交删除（eb083d7/ce9adc5），不在本 PR 内。
  - 删除 `utils.ResolveRepoFromFlag` 薄包装（冗余入口），统一到唯一入口 `utils.ResolveRepo`（含 `gitee.default_repo` git-config fallback）。
  - 4 个调用方（release/list、webhook、label、milestone 的包装函数）内联"本地 flag 为空时回退全局 --repo"逻辑后改调 `ResolveRepo`，行为等价。
  - `internal/output` 极浅包（36 文件引用 NewTableWriter）留待骨架抽象（IK923U）落地后统一接管，本次不处理以免扩大范围。
- **遗留问题（潜在 bug，建议单独建 issue）**：`gitee.default_repo` 配置 key 含下划线，git config 拒绝该 key（invalid key），导致 `gitee.default_repo` fallback 从未真正生效。涉及 `cmd/status/run.go`、`cmd/repo/set_default.go`、`internal/utils/utils.go`。建议改为 `gitee.default-repo`（连字符）。
- **特殊依赖**：无

### IK923U — 抽象命令执行骨架，消除 list/view 命令"五段式"样板重复

- **状态**：已提交 PR #15，等待人工审核合并
- **修复 PR**：[#15](https://gitee.com/andershsueh/gitee-cli/pulls/15)
- **测试结果**：internal/cmdkit/list_test.go 8 个 httptest 边界单测通过（分页合并/limit截断/json-table分派/空结果/404→空/Footer/CustomJSON）；test/test-IK923U-cmdkit-list-skeleton.sh 全绿（11 通过）；现有各模块 *_test.go 全部继续通过（输出逐字节一致）；go build/vet/test ./... 全绿；golangci-lint 0 issues
- **原则校验**：跳过（原则文档不存在，用户确认跳过）
- **决策**：
  - 新增 `internal/cmdkit` 深模块骨架（`RunList` + `ListSpec`），统一 repo 解析、client 构造、超时、分页（--paginate/--page/--per-page/--limit）、json/table 输出分派、空结果提示、404→空列表。
  - 业务差异通过 `ListSpec` 声明式配置：`ResolveOwnerRepo` / `Path` / `Limit` / `Columns` / `Row` / `EmptyText`，以及扩展的 `Fetch`（自定义数据获取）/ `Footer`（表尾 Total 行）/ `JSON`（自定义输出）。
  - **扩展迁移（用户要求彻底完成）**：全部 11 个 API list 命令迁移到骨架——issue/pr/repo/gist/release/label/milestone/webhook/org/notification/ssh-key。runList 从约 110 行降为约 40 行声明式配置。
  - 全局行为不变，现有测试全部继续通过（输出逐字节一致）。
- **保留原样**：alias list、extension list（本地数据源 + map JSON 输出，无 client/超时/分页样板，不适用 API 骨架）。
- **遗留问题**：
  - 改动超过 300 行（骨架实现 + 配套测试），已在 PR 描述注明建议先做设计评审再合并。Gitee CLI 无给 PR 打标签的命令，故以描述提示代替 large-change-needs-design-review 标签。
- **特殊依赖**：无
### IK923S — 收敛 cmd.Global* 全局状态为显式注入的运行时上下文（完整迁移）

- **状态**：已提交 PR #18，等待人工审核合并
- **修复 PR**：[#18](https://gitee.com/andershsueh/gitee-cli/pulls/18)
- **测试结果**：test-case/test-IK923S-runtime-context.sh 27 PASS 0 FAIL；go test ./... 全绿（27 个包）；go vet 通过
- **原则校验**：跳过（原则文档不存在）
- **决策**：
  - 全量迁移 110 个文件，607 行增量
  - 所有 RunE/PreRunE 函数使用 `rt := cmd.RuntimeFromCmd(cobraCmd)`
  - 辅助函数保留 `cmd.Global*`（向后兼容，无 cobraCmd 参数时不可用）
  - `NewAPIClient()` → `NewAPIClientFrom(rt)`
  - `NewCmdkitOptions(client)` → `NewCmdkitOptionsFrom(rt, client)`
  - `NewDomainClient()` → `NewDomainClientFrom(rt)`
  - `RuntimeFromCmd` 和 `RuntimeFrom` 增加 nil 保护
- **遗留问题**：
  - 测试文件中的 save/restore 仪式代码未迁移
  - 改动 607 行超阈值，PR 已打 large-change 标签

### IK923T — 统一命令组织模式：消除 internal/handler 分层倒置与双轨制

- **状态**：已提交 PR #16，等待人工审核合并
- **修复 PR**：[#16](https://gitee.com/andershsueh/gitee-cli/pulls/16)
- **测试结果**：grep '"gitee-cli/cmd"' internal/ 为空（internal 不再 import cmd）；go build/test ./... 全绿；golangci-lint 0 issues；gofmt 通过；notification_test.go 迁移后 3 个用例继续通过
- **原则校验**：已核对（纯架构重构，无冲突）
- **决策**：
  - 新增 `internal/domain` 包：`Client` 结构（`API`/`Timeout`/`Output`/`Repo` 显式依赖）+ label/milestone/notification/webhook 四个 domain 文件（原 handler 业务逻辑迁移，Git 识别为 rename 68%/67%/76%/66%）。
  - `internal/cmdkit` 解耦：`RunList` 改为接受显式 `Options`（Client/Timeout/Output/Paginate/PerPage/Limit/Page/Repo），不再读取 cmd 全局状态；11 个 list 命令调用方更新。
  - `internal/output` 新增 `PrintJSON`（原 handler.PrintJSON 迁移，cmd 层与 domain 共用）。
  - cmd 层新增 `NewDomainClient()`/`NewCmdkitOptions()` 依赖注入辅助，cmd 成为唯一读取全局状态的层。
  - 删除 `internal/handler` 4 个文件，消除 internal → cmd 逆向依赖。
- **遗留问题**：
  - 改动 39 文件超阈值，PR 描述已注明建议先做设计评审再合并（Gitee CLI 无打标签命令，以描述提示代替）。
- **特殊依赖**：无
