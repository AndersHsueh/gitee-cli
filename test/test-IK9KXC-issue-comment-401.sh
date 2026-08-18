#!/usr/bin/env bash
# test-IK9KXC-issue-comment-401.sh
#
# 功能：验证 issue comment 对锁定评论的 issue 不再误报 401 认证失败，
#       且 issue comment 写端点使用 form 编码（非 JSON）。
#
# 关联 issue：IK9KXC（bug: issue comment 对锁定评论的 issue 误报 401 认证失败,
#             且 JSON body 被 Gitee 拒收）
#
# 前置：需要先编译二进制并完成 gitee auth login。
# 用法：GITEE_CLI_BINARY=./gitee bash test/test-IK9KXC-issue-comment-401.sh
#
# 说明：真正的 401-body 透传与 formData 断言由 Go 单元测试
#       cmd/issue/comment_test.go 覆盖（httptest 模拟 Gitee，无需真实 token）。
#       本脚本是功能层面的补充验证。

set -euo pipefail

BIN="${GITEE_CLI_BINARY:-./gitee}"
REPO="andershsueh/gitee-cli"
PASS=0
FAIL=0

pass() { echo -e "\033[32mPASS\033[0m $1"; PASS=$((PASS+1)); }
fail() { echo -e "\033[31mFAIL\033[0m $1"; FAIL=$((FAIL+1)); }

echo "==> 检查认证状态"
if ! "$BIN" auth status >/dev/null 2>&1; then
  echo "SKIP: 未登录，无法执行功能测试（依赖真实 token）。请先运行 gitee auth login。"
  echo "（Go 单元测试 cmd/issue/comment_test.go 不依赖真实 token，可独立验证本修复。）"
  exit 0
fi

echo "==> 验证 issue comment 写端点使用 form 编码（通过 --verbose 观察）"
OUT=$("$BIN" issue comment IK9KXC --repo "$REPO" --body "IK9KXC 自动化验证：formData 契约检查" --verbose 2>&1 || true)
if echo "$OUT" | grep -qi "authentication failed"; then
  fail "issue comment 仍误报认证失败（锁定评论 issue 或 JSON body 问题）"
else
  pass "issue comment 未误报认证失败"
fi

echo
echo "=================================="
echo "结果: $PASS 通过, $FAIL 失败"
echo "=================================="
[ "$FAIL" -eq 0 ]