#!/usr/bin/env bash
# =============================================================================
# gitee extension 模块测试
# 测试扩展管理相关命令
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
# extension --help
# ---------------------------------------------------------------------------
section "gitee extension --help"
if "$BINARY" extension --help > /dev/null 2>&1; then
    pass "extension --help 正常显示"
else
    fail "extension --help 失败 (需重新编译二进制以启用 extension 模块)"
fi

# ---------------------------------------------------------------------------
# extension 子命令列表
# ---------------------------------------------------------------------------
section "gitee extension 子命令验证"
expected_commands=("list" "install" "remove" "upgrade" "create")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" extension "$cmd" --help > /dev/null 2>&1; then
        pass "extension $cmd 存在"
    else
        fail "extension $cmd 不存在"
    fi
done

# ---------------------------------------------------------------------------
# extension list --help
# ---------------------------------------------------------------------------
section "gitee extension list --help"
if "$BINARY" extension list --help > /dev/null 2>&1; then
    pass "extension list --help 正常"
else
    fail "extension list --help 失败"
fi

# ---------------------------------------------------------------------------
# extension install --help 参数验证
# ---------------------------------------------------------------------------
section "gitee extension install --help"
help_output=$("$BINARY" extension install --help 2>&1)
if echo "$help_output" | grep -qE "repo|url|local"; then
    pass "extension install --help 包含参数说明"
else
    fail "extension install --help 缺少参数说明"
fi

# ---------------------------------------------------------------------------
# extension remove --help
# ---------------------------------------------------------------------------
section "gitee extension remove --help"
if "$BINARY" extension remove --help > /dev/null 2>&1; then
    pass "extension remove --help 正常"
else
    fail "extension remove --help 失败"
fi

# ---------------------------------------------------------------------------
# extension upgrade --help
# ---------------------------------------------------------------------------
section "gitee extension upgrade --help"
if "$BINARY" extension upgrade --help > /dev/null 2>&1; then
    pass "extension upgrade --help 正常"
else
    fail "extension upgrade --help 失败"
fi

# ---------------------------------------------------------------------------
# extension create --help
# ---------------------------------------------------------------------------
section "gitee extension create --help"
if "$BINARY" extension create --help > /dev/null 2>&1; then
    pass "extension create --help 正常"
else
    fail "extension create --help 失败"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ extension 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ extension 模块测试有失败项${NC}"
    exit 1
fi
