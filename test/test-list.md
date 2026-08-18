# 测试清单 (test-list)

> 本清单记录 issue 修复相关的测试脚本。每行一条记录。
> 字段：测试脚本文件名 | 功能描述 | 所属功能模块 | 关联PR（待定） | 修复前状态

| 测试脚本文件名 | 功能描述 | 所属功能模块 | 关联PR | 修复前状态 |
|---------------|---------|-------------|-------|-----------|
| cmd/issue/comment_test.go | issue comment 401-body 透传 + formData 契约（httptest 单测） | issue | #13 | 红 |
| test/test-IK9KXC-issue-comment-401.sh | issue comment 锁定评论不误报 401（功能验证） | issue | #13 | 红（未修复时误报） |
| test/test-IK923V-converge-repo-resolve.sh | repo 解析入口收敛：死代码包/冗余入口无引用 + 唯一入口三种路径解析 | repo/utils | #14 | 红（修复前 ResolveRepoFromFlag 存在） |
| internal/cmdkit/list_test.go | list 命令骨架边界：分页合并/limit截断/json-table分派/空结果/404→空/Footer/CustomJSON（httptest 单测） | cmdkit | #15 | 红（骨架不存在时无法编译） |
| test/test-IK923U-cmdkit-list-skeleton.sh | list 骨架迁移：cmdkit 包存在 + 11 个 API list 命令迁移 + alias/extension 保留 + CLI 级输出验证 | cmdkit/issue/pr/repo/gist/release/label/milestone/webhook/org/notification/ssh-key | #15 | 红（修复前无 cmdkit 包） |