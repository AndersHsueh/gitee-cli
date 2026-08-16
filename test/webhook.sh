#!/usr/bin/env bash
# =============================================================================
# gitee webhook 模块测试
# 测试 Webhook 管理相关命令
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
    skip "未登录，跳过 webhook 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# webhook --help
# ---------------------------------------------------------------------------
section "gitee webhook --help"
if "$BINARY" webhook --help > /dev/null 2>&1; then
    pass "webhook --help 正常显示"
else
    fail "webhook --help 失败 (需重新编译二进制以启用 webhook 模块)"
fi

# ---------------------------------------------------------------------------
# webhook 子命令列表
# ---------------------------------------------------------------------------
section "gitee webhook 子命令验证"
expected_commands=("list" "create" "view" "edit" "delete" "test")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" webhook "$cmd" --help > /dev/null 2>&1; then
        pass "webhook $cmd 存在"
    else
        fail "webhook $cmd 不存在"
    fi
done

# ---------------------------------------------------------------------------
# webhook list --help
# ---------------------------------------------------------------------------
section "gitee webhook list --help"
if "$BINARY" webhook list --help > /dev/null 2>&1; then
    pass "webhook list --help 正常"
else
    fail "webhook list --help 失败"
fi

# ---------------------------------------------------------------------------
# webhook list --output json
# ---------------------------------------------------------------------------
section "gitee webhook list --output json"
output=$("$BINARY" webhook list --repo gitee/gitee --output json --limit 5 2>&1 || true)
if echo "$output" | grep -qE '"id"|"url"|type|\['; then
    pass "webhook list --output json 输出有效"
else
    fail "webhook list --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# webhook create --help 参数验证
# ---------------------------------------------------------------------------
section "gitee webhook create --help"
help_output=$("$BINARY" webhook create --help 2>&1)
expected_args=("url" "push_events" "pull_requests_events")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "webhook create --help 包含 $arg 参数"
    else
        fail "webhook create --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# webhook view --help
# ---------------------------------------------------------------------------
section "gitee webhook view --help"
if "$BINARY" webhook view --help > /dev/null 2>&1; then
    pass "webhook view --help 正常"
else
    fail "webhook view --help 失败"
fi

# ---------------------------------------------------------------------------
# webhook edit --help
# ---------------------------------------------------------------------------
section "gitee webhook edit --help"
if "$BINARY" webhook edit --help > /dev/null 2>&1; then
    pass "webhook edit --help 正常"
else
    fail "webhook edit --help 失败"
fi

# ---------------------------------------------------------------------------
# webhook delete --help
# ---------------------------------------------------------------------------
section "gitee webhook delete --help"
if "$BINARY" webhook delete --help > /dev/null 2>&1; then
    pass "webhook delete --help 正常"
else
    fail "webhook delete --help 失败"
fi

# ---------------------------------------------------------------------------
# webhook test --help
# ---------------------------------------------------------------------------
section "gitee webhook test --help"
if "$BINARY" webhook test --help > /dev/null 2>&1; then
    pass "webhook test --help 正常"
else
    fail "webhook test --help 失败"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ webhook 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ webhook 模块测试有失败项${NC}"
    exit 1
fi
