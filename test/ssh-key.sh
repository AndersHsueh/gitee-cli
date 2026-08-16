#!/usr/bin/env bash
# =============================================================================
# gitee ssh-key 模块测试
# 测试 SSH 密钥管理相关命令
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
    skip "未登录，跳过 ssh-key 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# ssh-key --help
# ---------------------------------------------------------------------------
section "gitee ssh-key --help"
if "$BINARY" ssh-key --help > /dev/null 2>&1; then
    pass "ssh-key --help 正常显示"
else
    fail "ssh-key --help 失败 (需重新编译二进制以启用 ssh-key 模块)"
fi

# ---------------------------------------------------------------------------
# ssh-key 子命令列表
# ---------------------------------------------------------------------------
section "gitee ssh-key 子命令验证"
expected_commands=("list" "add" "delete")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" ssh-key "$cmd" --help > /dev/null 2>&1; then
        pass "ssh-key $cmd 存在"
    else
        fail "ssh-key $cmd 不存在"
    fi
done

# ---------------------------------------------------------------------------
# ssh-key list --help
# ---------------------------------------------------------------------------
section "gitee ssh-key list --help"
if "$BINARY" ssh-key list --help > /dev/null 2>&1; then
    pass "ssh-key list --help 正常"
else
    fail "ssh-key list --help 失败"
fi

# ---------------------------------------------------------------------------
# ssh-key list --output json
# ---------------------------------------------------------------------------
section "gitee ssh-key list --output json"
output=$("$BINARY" ssh-key list --output json 2>&1 || true)
if echo "$output" | grep -qE '"title"|"key"|id|\['; then
    pass "ssh-key list --output json 输出有效"
else
    fail "ssh-key list --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# ssh-key add --help 参数验证
# ---------------------------------------------------------------------------
section "gitee ssh-key add --help"
help_output=$("$BINARY" ssh-key add --help 2>&1)
expected_args=("title" "key-file")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "ssh-key add --help 包含 $arg 参数"
    else
        fail "ssh-key add --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# ssh-key delete --help
# ---------------------------------------------------------------------------
section "gitee ssh-key delete --help"
if "$BINARY" ssh-key delete --help > /dev/null 2>&1; then
    pass "ssh-key delete --help 正常"
else
    fail "ssh-key delete --help 失败"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ ssh-key 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ ssh-key 模块测试有失败项${NC}"
    exit 1
fi
