#!/usr/bin/env bash
# test-IK923U-cmdkit-list-skeleton.sh
#
# 功能：验证 list 命令骨架抽象（IK923U）：
#   - internal/cmdkit 骨架包存在且编译通过
#   - issue list / pr list 已迁移到骨架，table 与 json 模式输出正常
#   - 骨架边界行为（分页/limit/json/table/空结果/404）由 Go 单测
#     internal/cmdkit/list_test.go 覆盖（httptest，无需真实 token）
#
# 关联 issue：IK923U（[refactor] 抽象命令执行骨架，消除 list/view
#             命令的"五段式"样板重复）
#
# 用法：GITEE_CLI_BINARY=./gitee bash test/test-IK923U-cmdkit-list-skeleton.sh
# 前置：需要先编译二进制并完成 gitee auth login（CLI 级验证需要）。

set -euo pipefail

BIN="${GITEE_CLI_BINARY:-./gitee}"
PASS=0
FAIL=0

pass() { echo -e "\033[32mPASS\033[0m $1"; PASS=$((PASS+1)); }
fail() { echo -e "\033[31mFAIL\033[0m $1"; FAIL=$((FAIL+1)); }

echo "==> 1. cmdkit 骨架包存在"
if ls internal/cmdkit/list.go >/dev/null 2>&1; then
  pass "internal/cmdkit/list.go 存在"
else
  fail "internal/cmdkit/list.go 缺失"
fi

echo "==> 2. issue list / pr list 已迁移到骨架（引用 cmdkit.RunList）"
if grep -q "cmdkit.RunList" cmd/issue/list.go && grep -q "cmdkit.RunList" cmd/pr/list.go; then
  pass "issue list 与 pr list 均使用 cmdkit.RunList"
else
  fail "issue list 或 pr list 未使用 cmdkit.RunList"
fi

echo "==> 3. 编译通过（go build ./...）"
if go build ./... >/dev/null 2>&1; then
  pass "go build ./... 通过"
else
  fail "go build ./... 失败"
fi

echo "==> 4. 骨架边界单测通过（go test ./internal/cmdkit/）"
if go test ./internal/cmdkit/ -count=1 >/dev/null 2>&1; then
  pass "go test ./internal/cmdkit/ 通过"
else
  fail "go test ./internal/cmdkit/ 失败"
fi

echo
echo "==> 5. CLI 级输出验证（需登录，未登录则跳过）"
if ! "$BIN" auth status >/dev/null 2>&1; then
  echo "SKIP: 未登录，跳过 CLI 级验证（Go 单测已覆盖骨架边界）"
else
  # 5a. issue list table 模式
  if "$BIN" issue list --repo andershsueh/gitee-cli --state open >/dev/null 2>&1; then
    pass "gitee issue list table 模式正常"
  else
    fail "gitee issue list table 模式失败"
  fi
  # 5b. issue list json 模式（验证 JSON 可解析）
  if "$BIN" issue list --repo andershsueh/gitee-cli --state open -o json | python3 -m json.tool >/dev/null 2>&1; then
    pass "gitee issue list json 模式输出合法 JSON"
  else
    fail "gitee issue list json 模式输出非法"
  fi
  # 5c. pr list table 模式
  if "$BIN" pr list --repo andershsueh/gitee-cli --state open >/dev/null 2>&1; then
    pass "gitee pr list table 模式正常"
  else
    fail "gitee pr list table 模式失败"
  fi
  # 5d. pr list json 模式
  if "$BIN" pr list --repo andershsueh/gitee-cli --state open -o json | python3 -m json.tool >/dev/null 2>&1; then
    pass "gitee pr list json 模式输出合法 JSON"
  else
    fail "gitee pr list json 模式输出非法"
  fi
fi

echo
echo "=================================="
echo "结果: $PASS 通过, $FAIL 失败"
echo "=================================="
[ "$FAIL" -eq 0 ]