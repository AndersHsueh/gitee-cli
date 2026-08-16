#!/usr/bin/env bash
# =============================================================================
# gitee milestone 模块测试
# 测试里程碑管理相关命令
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
    skip "未登录，跳过 milestone 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# milestone --help
# ---------------------------------------------------------------------------
section "gitee milestone --help"
if "$BINARY" milestone --help > /dev/null 2>&1; then
    pass "milestone --help 正常显示"
else
    fail "milestone --help 失败 (需重新编译二进制以启用 milestone 模块)"
fi

# ---------------------------------------------------------------------------
# milestone 子命令列表
# ---------------------------------------------------------------------------
section "gitee milestone 子命令验证"
expected_commands=("list" "create" "view" "edit" "delete")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" milestone "$cmd" --help > /dev/null 2>&1; then
        pass "milestone $cmd 存在"
    else
        fail "milestone $cmd 不存在"
    fi
done

# ---------------------------------------------------------------------------
# milestone list --help
# ---------------------------------------------------------------------------
section "gitee milestone list --help"
if "$BINARY" milestone list --help > /dev/null 2>&1; then
    pass "milestone list --help 正常"
else
    fail "milestone list --help 失败"
fi

# ---------------------------------------------------------------------------
# milestone list --output json
# ---------------------------------------------------------------------------
section "gitee milestone list --output json"
output=$("$BINARY" milestone list --repo gitee/gitee --output json --limit 5 2>&1 || true)
if echo "$output" | grep -qE '"title"|"number"|state|\['; then
    pass "milestone list --output json 输出有效"
else
    fail "milestone list --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# milestone create --help 参数验证
# ---------------------------------------------------------------------------
section "gitee milestone create --help"
help_output=$("$BINARY" milestone create --help 2>&1)
expected_args=("title" "description" "due-on" "state")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "milestone create --help 包含 $arg 参数"
    else
        fail "milestone create --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# milestone view --help
# ---------------------------------------------------------------------------
section "gitee milestone view --help"
if "$BINARY" milestone view --help > /dev/null 2>&1; then
    pass "milestone view --help 正常"
else
    fail "milestone view --help 失败"
fi

# ---------------------------------------------------------------------------
# milestone edit --help
# ---------------------------------------------------------------------------
section "gitee milestone edit --help"
if "$BINARY" milestone edit --help > /dev/null 2>&1; then
    pass "milestone edit --help 正常"
else
    fail "milestone edit --help 失败"
fi

# ---------------------------------------------------------------------------
# milestone delete --help
# ---------------------------------------------------------------------------
section "gitee milestone delete --help"
if "$BINARY" milestone delete --help > /dev/null 2>&1; then
    pass "milestone delete --help 正常"
else
    fail "milestone delete --help 失败"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ milestone 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ milestone 模块测试有失败项${NC}"
    exit 1
fi
