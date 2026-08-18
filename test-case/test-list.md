# test-case 测试脚本清单

| 测试脚本文件名 | 功能描述 | 所属功能模块 | 关联PR（待定） | 修复前状态：红 |
|---|---|---|---|---|
| test-IK923S-runtime-context.sh | 验收 cmd.Runtime 运行时上下文收敛：Runtime 类型存在、全局变量声明保留、测试无 save/restore 仪式代码、CLI 行为不变、internal 不 import cmd | cmd（全模块） | 待定 | 红（迁移前测试文件含 189 处仪式代码） |