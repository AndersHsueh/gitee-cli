#!/usr/bin/env bash
# test-IK923S-runtime-context.sh
#
# 功能：验证 cmd.Runtime 运行时上下文基础设施（IK923S）：
#   - Runtime 类型定义存在且包含所有必需字段
#   - RuntimeFromCmd / NewAPIClientFrom / NewCmdkitOptionsFrom 可用
#   - 测试可安全使用 Runtime 构造替代全局 save/restore 仪式代码
#   - CLI 外在行为（编译、测试、vet）完全不变
#
# 关联 issue：IK923S（[refactor] 收敛 cmd.Global* 全局状态为显式注入的运行时上下文）
#
# 用法：GITEE_CLI_BINARY=./gitee bash test-case/test-IK923S-runtime-context.sh
# 前置：需要先编译二进制（Go 静态检查无需登录）

set -euo pipefail

BIN="${GITEE_CLI_BINARY:-./gitee}"
PASS=0
FAIL=0

pass() { echo -e "\033[32mPASS\033[0m $1"; PASS=$((PASS+1)); }
fail() { echo -e "\033[31mFAIL\033[0m $1"; FAIL=$((FAIL+1)); }

echo "===== 阶段 1：Runtime 类型存在性 ====="
echo ""

echo "==> 1.1 cmd.Runtime 类型定义存在"
if grep -q "type Runtime struct" cmd/runtime.go 2>/dev/null; then
  pass "Runtime 类型定义存在（cmd/runtime.go）"
else
  fail "Runtime 类型定义不存在"
fi

echo "==> 1.2 Runtime 包含全部必需字段"
REQUIRED_FIELDS=("Token" "Hostname" "Output" "Verbose" "APIURL" "Timeout" "Repo" "NoColor" "AgentHelp" "Paginate" "Page" "PerPage" "Limit" "NoExtensions")
ALL_FIELDS=1
for field in "${REQUIRED_FIELDS[@]}"; do
  if grep -q "^\s*$field\s" cmd/runtime.go 2>/dev/null; then
    pass "  Runtime.$field 存在"
  else
    fail "  Runtime.$field 缺失"
    ALL_FIELDS=0
  fi
done

echo ""
echo "===== 阶段 2：RuntimeFromCmd / NewAPIClientFrom 可用 ====="
echo ""

echo "==> 2.1 RuntimeFromCmd 函数存在"
if grep -q "func RuntimeFromCmd" cmd/runtime.go 2>/dev/null; then
  pass "RuntimeFromCmd 存在"
else
  fail "RuntimeFromCmd 缺失"
fi

echo "==> 2.2 NewAPIClientFrom 函数存在"
if grep -q "func NewAPIClientFrom" cmd/runtime.go 2>/dev/null; then
  pass "NewAPIClientFrom 存在"
else
  fail "NewAPIClientFrom 缺失"
fi

echo "==> 2.3 NewCmdkitOptionsFrom 函数存在"
if grep -q "func NewCmdkitOptionsFrom" cmd/runtime.go 2>/dev/null; then
  pass "NewCmdkitOptionsFrom 存在"
else
  fail "NewCmdkitOptionsFrom 缺失"
fi

echo "==> 2.4 NewDomainClientFrom 函数存在"
if grep -q "func NewDomainClientFrom" cmd/runtime.go 2>/dev/null; then
  pass "NewDomainClientFrom 存在"
else
  fail "NewDomainClientFrom 缺失"
fi

echo "==> 2.5 参考实现：ssh-key list 使用 RuntimeFromCmd"
if grep -q "RuntimeFromCmd" cmd/ssh-key/list.go 2>/dev/null; then
  pass "ssh-key list 已迁移到 RuntimeFromCmd"
else
  fail "ssh-key list 未使用 RuntimeFromCmd"
fi

echo "==> 2.6 参考测试：ssh-key list_test 使用 Runtime 构造"
if grep -q "cmd\.SetContext.*cmd\.Runtime" cmd/ssh-key/list_test.go 2>/dev/null; then
  pass "ssh-key list_test 使用 Runtime 构造代替 save/restore"
else
  fail "ssh-key list_test 未使用 Runtime 构造"
fi

echo ""
echo "===== 阶段 3：编译与测试 ====="
echo ""

echo "==> 3.1 编译通过"
if go build ./... 2>/dev/null; then
  pass "go build ./... 通过"
else
  fail "go build ./... 失败"
fi

echo "==> 3.2 Go 单元测试全绿"
if go test ./... -count=1 2>/dev/null; then
  pass "go test ./... 全绿"
else
  fail "go test ./... 存在失败"
fi

echo "==> 3.3 go vet 通过"
if go vet ./... 2>/dev/null; then
  pass "go vet ./... 通过"
else
  fail "go vet ./... 存在警告"
fi

echo "==> 3.4 --help 输出正常"
if "$BIN" --help >/dev/null 2>&1; then
  pass "gitee --help 正常"
else
  fail "gitee --help 失败"
fi

echo "==> 3.5 --version 输出正常"
if "$BIN" --version >/dev/null 2>&1; then
  pass "gitee --version 正常"
else
  fail "gitee --version 失败"
fi

echo "==> 3.6 internal 包不再 import cmd"
if grep -r '"gitee-cli/cmd"' internal/ 2>/dev/null; then
  fail "internal/ 包仍引用 gitee-cli/cmd"
else
  pass "internal/ 包无 gitee-cli/cmd 引用"
fi

echo ""
echo "=================================="
echo "结果: $PASS 通过, $FAIL 失败"
echo "=================================="
[ "$FAIL" -eq 0 ]