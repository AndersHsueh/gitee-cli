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

echo "==> 2. 所有 API list 命令已迁移到骨架（引用 cmdkit.RunList）"
MIGRATED_CMDS=(
  "cmd/issue/list.go"
  "cmd/pr/list.go"
  "cmd/repo/list.go"
  "cmd/gist/list.go"
  "cmd/release/list.go"
  "cmd/org/list.go"
  "cmd/ssh-key/list.go"
  "internal/handler/label.go"
  "internal/handler/milestone.go"
  "internal/handler/webhook.go"
  "internal/handler/notification.go"
)
ALL_MIGRATED=1
for f in "${MIGRATED_CMDS[@]}"; do
  if ! grep -q "cmdkit.RunList" "$f"; then
    fail "$f 未使用 cmdkit.RunList"
    ALL_MIGRATED=0
  fi
done
if [ "$ALL_MIGRATED" -eq 1 ]; then
  pass "${#MIGRATED_CMDS[@]} 个 API list 命令均使用 cmdkit.RunList"
fi

echo "==> 3. alias/extension list 保留原样（本地数据源，不适用 API 骨架）"
if grep -q "cmdkit.RunList" cmd/alias/list.go || grep -q "cmdkit.RunList" cmd/extension/list.go; then
  fail "alias/extension 不应迁移到 cmdkit（本地数据源）"
else
  pass "alias/extension list 保留本地实现（合理）"
fi

echo "==> 4. 编译通过（go build ./...）"
if go build ./... >/dev/null 2>&1; then
  pass "go build ./... 通过"
else
  fail "go build ./... 失败"
fi

echo "==> 5. 骨架边界单测通过（go test ./internal/cmdkit/）"
if go test ./internal/cmdkit/ -count=1 >/dev/null 2>&1; then
  pass "go test ./internal/cmdkit/ 通过"
else
  fail "go test ./internal/cmdkit/ 失败"
fi

echo
echo "==> 6. CLI 级输出验证（需登录，未登录则跳过）"
if ! "$BIN" auth status >/dev/null 2>&1; then
  echo "SKIP: 未登录，跳过 CLI 级验证（Go 单测已覆盖骨架边界）"
else
  # 6a. issue list table 模式
  if "$BIN" issue list --repo andershsueh/gitee-cli --state open >/dev/null 2>&1; then
    pass "gitee issue list table 模式正常"
  else
    fail "gitee issue list table 模式失败"
  fi
  # 6b. issue list json 模式（验证 JSON 可解析）
  if "$BIN" issue list --repo andershsueh/gitee-cli --state open -o json | python3 -m json.tool >/dev/null 2>&1; then
    pass "gitee issue list json 模式输出合法 JSON"
  else
    fail "gitee issue list json 模式输出非法"
  fi
  # 6c. pr list table 模式
  if "$BIN" pr list --repo andershsueh/gitee-cli --state open >/dev/null 2>&1; then
    pass "gitee pr list table 模式正常"
  else
    fail "gitee pr list table 模式失败"
  fi
  # 6d. pr list json 模式
  if "$BIN" pr list --repo andershsueh/gitee-cli --state open -o json | python3 -m json.tool >/dev/null 2>&1; then
    pass "gitee pr list json 模式输出合法 JSON"
  else
    fail "gitee pr list json 模式输出非法"
  fi
  # 6e. label list table 模式（handler 层迁移）
  if "$BIN" label list --repo andershsueh/gitee-cli >/dev/null 2>&1; then
    pass "gitee label list table 模式正常"
  else
    fail "gitee label list table 模式失败"
  fi
  # 6f. org list table 模式
  if "$BIN" org list >/dev/null 2>&1; then
    pass "gitee org list table 模式正常"
  else
    fail "gitee org list table 模式失败"
  fi
fi

echo
echo "=================================="
echo "结果: $PASS 通过, $FAIL 失败"
echo "=================================="
[ "$FAIL" -eq 0 ]