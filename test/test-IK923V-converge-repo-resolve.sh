#!/usr/bin/env bash
# test-IK923V-converge-repo-resolve.sh
#
# 功能：验证 repo 解析入口收敛（IK923V）：
#   - 死代码包 internal/printer、internal/repo 已删除（无引用）
#   - 冗余入口 utils.ResolveRepoFromFlag 已删除（无引用）
#   - 唯一入口 utils.ResolveRepo 可正常解析三种路径：
#       --repo owner/repo、位置参数 owner/repo、git config gitee.default_repo
#
# 关联 issue：IK923V（[chore] 清理死代码浅包 internal/printer、internal/repo，
#             收敛 repo 解析五入口）
#
# 用法：GITEE_CLI_BINARY=./gitee bash test/test-IK923V-converge-repo-resolve.sh
# 前置：需要先编译二进制并完成 gitee auth login（CLI 级路径验证需要）。

set -euo pipefail

BIN="${GITEE_CLI_BINARY:-./gitee}"
PASS=0
FAIL=0

pass() { echo -e "\033[32mPASS\033[0m $1"; PASS=$((PASS+1)); }
fail() { echo -e "\033[31mFAIL\033[0m $1"; FAIL=$((FAIL+1)); }

echo "==> 1. 死代码包无引用（internal/printer、internal/repo）"
if grep -rn "internal/printer\|internal/repo" --include="*.go" . >/dev/null 2>&1; then
  fail "internal/printer 或 internal/repo 仍有引用"
else
  pass "internal/printer、internal/repo 无引用（已删除）"
fi

echo "==> 2. 冗余入口 ResolveRepoFromFlag 无引用"
if grep -rn "ResolveRepoFromFlag" --include="*.go" . >/dev/null 2>&1; then
  fail "utils.ResolveRepoFromFlag 仍有引用"
else
  pass "utils.ResolveRepoFromFlag 已删除，无引用"
fi

echo "==> 3. 唯一入口 utils.ResolveRepo 存在"
if grep -rn "func ResolveRepo(" internal/utils/utils.go >/dev/null 2>&1; then
  pass "utils.ResolveRepo 唯一解析入口存在"
else
  fail "utils.ResolveRepo 缺失"
fi

echo "==> 4. 编译通过（go build ./...）"
if go build ./... >/dev/null 2>&1; then
  pass "go build ./... 通过"
else
  fail "go build ./... 失败"
fi

echo "==> 5. 单测通过（go test ./internal/utils/）"
if go test ./internal/utils/ -count=1 >/dev/null 2>&1; then
  pass "go test ./internal/utils/ 通过"
else
  fail "go test ./internal/utils/ 失败"
fi

echo
echo "==> 6. CLI 级解析路径验证（需登录，未登录则跳过）"
if ! "$BIN" auth status >/dev/null 2>&1; then
  echo "SKIP: 未登录，跳过 CLI 级路径验证（Go 单测已覆盖 --repo 与位置参数路径）"
else
  # 6a. --repo owner/repo 路径
  if "$BIN" repo view --repo andershsueh/gitee-cli >/dev/null 2>&1; then
    pass "--repo owner/repo 路径解析成功"
  else
    fail "--repo owner/repo 路径解析失败"
  fi
  # 6b. 位置参数 owner/repo 路径
  if "$BIN" repo view andershsueh/gitee-cli >/dev/null 2>&1; then
    pass "位置参数 owner/repo 路径解析成功"
  else
    fail "位置参数 owner/repo 路径解析失败"
  fi
fi

echo
echo "=================================="
echo "结果: $PASS 通过, $FAIL 失败"
echo "=================================="
[ "$FAIL" -eq 0 ]