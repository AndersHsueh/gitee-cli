#!/usr/bin/env bash
# =============================================================================
# gitee notification 模块测试
# 测试通知管理相关命令
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
    skip "未登录，跳过 notification 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# notification --help
# ---------------------------------------------------------------------------
section "gitee notification --help"
if "$BINARY" notification --help > /dev/null 2>&1; then
    pass "notification --help 正常显示"
else
    fail "notification --help 失败 (需重新编译二进制以启用 notification 模块)"
fi

# ---------------------------------------------------------------------------
# notification 子命令列表
# ---------------------------------------------------------------------------
section "gitee notification 子命令验证"
expected_commands=("list" "count" "read")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" notification "$cmd" --help > /dev/null 2>&1; then
        pass "notification $cmd 存在"
    else
        fail "notification $cmd 不存在"
    fi
done

# ---------------------------------------------------------------------------
# notification list --help
# ---------------------------------------------------------------------------
section "gitee notification list --help"
if "$BINARY" notification list --help > /dev/null 2>&1; then
    pass "notification list --help 正常"
else
    fail "notification list --help 失败"
fi

# ---------------------------------------------------------------------------
# notification list --output json
# ---------------------------------------------------------------------------
section "gitee notification list --output json"
output=$("$BINARY" notification list --output json --limit 5 2>&1 || true)
if echo "$output" | grep -qE '"id"|"unread"|\['; then
    pass "notification list --output json 输出有效"
else
    fail "notification list --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# notification count --help
# ---------------------------------------------------------------------------
section "gitee notification count --help"
if "$BINARY" notification count --help > /dev/null 2>&1; then
    pass "notification count --help 正常"
else
    fail "notification count --help 失败"
fi

# ---------------------------------------------------------------------------
# notification read --help
# ---------------------------------------------------------------------------
section "gitee notification read --help"
if "$BINARY" notification read --help > /dev/null 2>&1; then
    pass "notification read --help 正常"
else
    fail "notification read --help 失败"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ notification 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ notification 模块测试有失败项${NC}"
    exit 1
fi
