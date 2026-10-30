#!/usr/bin/env bash
# =============================================================================
# gitee alias 模块测试
# 测试别名管理相关命令
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
# alias --help
# ---------------------------------------------------------------------------
section "gitee alias --help"
if "$BINARY" alias --help > /dev/null 2>&1; then
    pass "alias --help 正常显示"
else
    fail "alias --help 失败 (需重新编译二进制以启用 alias 模块)"
fi

# ---------------------------------------------------------------------------
# alias 子命令列表
# ---------------------------------------------------------------------------
section "gitee alias 子命令验证"
expected_commands=("set" "list" "delete")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" alias "$cmd" --help > /dev/null 2>&1; then
        pass "alias $cmd 存在"
    else
        fail "alias $cmd 不存在"
    fi
done

# ---------------------------------------------------------------------------
# alias list --help
# ---------------------------------------------------------------------------
section "gitee alias list --help"
if "$BINARY" alias list --help > /dev/null 2>&1; then
    pass "alias list --help 正常"
else
    fail "alias list --help 失败"
fi

# ---------------------------------------------------------------------------
# alias set --help 参数验证
# ---------------------------------------------------------------------------
section "gitee alias set --help"
help_output=$("$BINARY" alias set --help 2>&1)
if echo "$help_output" | grep -qE "alias|command|expansion"; then
    pass "alias set --help 包含参数说明"
else
    fail "alias set --help 缺少参数说明"
fi

# ---------------------------------------------------------------------------
# alias delete --help
# ---------------------------------------------------------------------------
section "gitee alias delete --help"
if "$BINARY" alias delete --help > /dev/null 2>&1; then
    pass "alias delete --help 正常"
else
    fail "alias delete --help 失败"
fi

# ---------------------------------------------------------------------------
# alias list --output json
# ---------------------------------------------------------------------------
section "gitee alias list --output json"
output=$("$BINARY" alias list --output json 2>&1 || true)
# Empty alias file is {} (not the table notice). A non-empty file is an object.
if echo "$output" | grep -qE '^\{\}|^\{'; then
    pass "alias list --output json 输出有效"
else
    fail "alias list --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ alias 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ alias 模块测试有失败项${NC}"
    exit 1
fi
