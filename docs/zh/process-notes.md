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