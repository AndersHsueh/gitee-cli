#!/usr/bin/env bash
# =============================================================================
# gitee user 模块测试
# 测试用户相关命令
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
    skip "未登录，跳过 user 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# user --help
# ---------------------------------------------------------------------------
section "gitee user --help"
if "$BINARY" user --help > /dev/null 2>&1; then
    pass "user --help 正常显示"
else
    fail "user --help 失败 (需重新编译二进制以启用 user 模块)"
fi

# ---------------------------------------------------------------------------
# user 子命令列表
# ---------------------------------------------------------------------------
section "gitee user 子命令验证"
expected_commands=("view" "followers" "following" "follow" "unfollow")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" user "$cmd" --help > /dev/null 2>&1; then
        pass "user $cmd 存在"
    else
        fail "user $cmd 不存在"
    fi
done

# ---------------------------------------------------------------------------
# user view --help
# ---------------------------------------------------------------------------
section "gitee user view --help"
if "$BINARY" user view --help > /dev/null 2>&1; then
    pass "user view --help 正常"
else
    fail "user view --help 失败"
fi

# ---------------------------------------------------------------------------
# user view --output json
# ---------------------------------------------------------------------------
section "gitee user view --output json"
output=$("$BINARY" user view --output json 2>&1 || true)
if echo "$output" | grep -qE '"login"|"name"|"id"'; then
    pass "user view --output json 输出有效"
else
    fail "user view --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# user followers --help
# ---------------------------------------------------------------------------
section "gitee user followers --help"
if "$BINARY" user followers --help > /dev/null 2>&1; then
    pass "user followers --help 正常"
else
    fail "user followers --help 失败"
fi

# ---------------------------------------------------------------------------
# user following --help
# ---------------------------------------------------------------------------
section "gitee user following --help"
if "$BINARY" user following --help > /dev/null 2>&1; then
    pass "user following --help 正常"
else
    fail "user following --help 失败"
fi

# ---------------------------------------------------------------------------
# user follow --help
# ---------------------------------------------------------------------------
section "gitee user follow --help"
if "$BINARY" user follow --help > /dev/null 2>&1; then
    pass "user follow --help 正常"
else
    fail "user follow --help 失败"
fi

# ---------------------------------------------------------------------------
# user unfollow --help
# ---------------------------------------------------------------------------
section "gitee user unfollow --help"
if "$BINARY" user unfollow --help > /dev/null 2>&1; then
    pass "user unfollow --help 正常"
else
    fail "user unfollow --help 失败"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ user 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ user 模块测试有失败项${NC}"
    exit 1
fi
