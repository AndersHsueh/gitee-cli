#!/usr/bin/env bash
# =============================================================================
# gitee pr (Pull Request) 模块测试
# 测试 PR 管理相关命令
# =============================================================================

set -euo pipefail

BINARY="${GITEE_CLI_BINARY:-./gitee}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

passed=0; failed=0; skipped=0

pass() { passed=$((passed+1)); echo -e "  ${GREEN}✓${NC} $*"; }
fail() { failed=$((failed+1)); echo -e "  ${RED}✗${NC} $*" >&2; }
skip() { skipped=$((skipped+1)); echo -e "  ${YELLOW}⊘${NC} $* (跳过)"; }
section() { echo ""; echo -e "${BLUE}▶${NC} $*"; }

# ---------------------------------------------------------------------------
# 前置检查
# ---------------------------------------------------------------------------
section "前置检查: 认证状态"
if ! "$BINARY" auth status > /dev/null 2>&1; then
    skip "未登录，跳过 pr 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# pr --help
# ---------------------------------------------------------------------------
section "gitee pr --help"
"$BINARY" pr --help > /dev/null 2>&1 && pass "pr --help 正常" || fail "pr --help 失败"

# ---------------------------------------------------------------------------
# pr 子命令列表
# ---------------------------------------------------------------------------
section "gitee pr 子命令验证"
expected_commands=("list" "create" "view" "merge" "close" "reopen" "review" "diff" "checks")
for cmd in "${expected_commands[@]}"; do
    "$BINARY" pr "$cmd" --help > /dev/null 2>&1 && pass "pr $cmd 存在" || fail "pr $cmd 不存在"
done

# ---------------------------------------------------------------------------
# pr list --output json
# ---------------------------------------------------------------------------
section "gitee pr list --output json"
TEST_REPO="andershsueh/gitee-cli"
output=$("$BINARY" pr list --repo "$TEST_REPO" --output json --limit 5 2>&1 || true)
if echo "$output" | grep -qE '"pull_requests"|"number"|"title"|"state"|^\['; then
    pass "pr list --output json 输出正确"
else
    fail "pr list --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# pr list --state
# ---------------------------------------------------------------------------
section "gitee pr list --state"
for state in "open" "closed" "all"; do
    if output=$("$BINARY" pr list --repo "$TEST_REPO" --state "$state" --limit 3 2>&1); then
        pass "pr list --state $state 执行成功"
    else
        fail "pr list --state $state 执行失败"
    fi
done

# ---------------------------------------------------------------------------
# pr list --output table
# ---------------------------------------------------------------------------
section "gitee pr list --output table"
output=$("$BINARY" pr list --repo "$TEST_REPO" --limit 5 --output table 2>&1 || true)
if echo "$output" | grep -qiE "#|TITLE|PR|pull_request|No pull requests"; then
    pass "pr list --output table 输出正常"
else
    fail "pr list --output table 输出缺少表头"
fi

# ---------------------------------------------------------------------------
# pr create --help 参数验证
# ---------------------------------------------------------------------------
section "gitee pr create --help"
help_output=$("$BINARY" pr create --help 2>&1)
expected_args=("title" "body" "source" "target" "draft")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "pr create --help 包含 $arg 参数"
    else
        fail "pr create --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# pr create 必填参数验证
# ---------------------------------------------------------------------------
section "gitee pr create 必填参数验证"

# 缺少 --title 应报错
if output=$("$BINARY" pr create --repo gitee/gitee 2>&1); then
    fail "pr create 缺少 --title 应报错但未报错"
else
    pass "pr create 缺少 --title 正确报错"
fi

# ---------------------------------------------------------------------------
# pr view --help
# ---------------------------------------------------------------------------
section "gitee pr view --help"
"$BINARY" pr view --help > /dev/null 2>&1 && pass "pr view --help 正常" || fail "pr view --help 失败"

# ---------------------------------------------------------------------------
# pr merge --help (应包含 squash, delete-branch 参数)
# ---------------------------------------------------------------------------
section "gitee pr merge --help"
help_output=$("$BINARY" pr merge --help 2>&1)
if echo "$help_output" | grep -q "squash\|delete"; then
    pass "pr merge --help 包含 squash/delete 参数"
else
    fail "pr merge --help 缺少 squash/delete 参数"
fi

# ---------------------------------------------------------------------------
# pr close --help
# ---------------------------------------------------------------------------
section "gitee pr close --help"
"$BINARY" pr close --help > /dev/null 2>&1 && pass "pr close --help 正常" || fail "pr close --help 失败"

# ---------------------------------------------------------------------------
# pr reopen --help
# ---------------------------------------------------------------------------
section "gitee pr reopen --help"
"$BINARY" pr reopen --help > /dev/null 2>&1 && pass "pr reopen --help 正常" || fail "pr reopen --help 失败"

# ---------------------------------------------------------------------------
# pr review --help
# ---------------------------------------------------------------------------
section "gitee pr review --help"
help_output=$("$BINARY" pr review --help 2>&1)
expected_review_actions=("approve" "request-changes" "comment")
for action in "${expected_review_actions[@]}"; do
    if echo "$help_output" | grep -q "$action"; then
        pass "pr review --help 包含 $action"
    else
        fail "pr review --help 缺少 $action"
    fi
done

# ---------------------------------------------------------------------------
# pr diff --help
# ---------------------------------------------------------------------------
section "gitee pr diff --help"
"$BINARY" pr diff --help > /dev/null 2>&1 && pass "pr diff --help 正常" || fail "pr diff --help 失败"

# ---------------------------------------------------------------------------
# pr checks --help
# ---------------------------------------------------------------------------
section "gitee pr checks --help"
"$BINARY" pr checks --help > /dev/null 2>&1 && pass "pr checks --help 正常" || fail "pr checks --help 失败"

# ---------------------------------------------------------------------------
# 全局 flag
# ---------------------------------------------------------------------------
section "全局 Flag"
for fmt in "json" "table"; do
    TEST_REPO="${GITEE_TEST_REPO:-andershsueh/gitee-cli}"; "$BINARY" pr list --repo "$TEST_REPO" --output "$fmt" --limit 1 > /dev/null 2>&1 && pass "--output $fmt 有效" || fail "--output $fmt 无效"
done

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"
if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ pr 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ pr 模块测试有失败项${NC}"
    exit 1
fi