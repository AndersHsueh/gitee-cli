# 测试清单 (test-list)

> 本清单记录 issue 修复相关的测试脚本。每行一条记录。
> 字段：测试脚本文件名 | 功能描述 | 所属功能模块 | 关联PR（待定） | 修复前状态

| 测试脚本文件名 | 功能描述 | 所属功能模块 | 关联PR（待定） | 修复前状态 |
|---------------|---------|-------------|---------------|-----------|
| cmd/issue/comment_test.go | issue comment 401-body 透传 + formData 契约（httptest 单测） | issue | 待定 | 红 |
| test/test-IK9KXC-issue-comment-401.sh | issue comment 锁定评论不误报 401（功能验证） | issue | 待定 | 红（未修复时误报） |