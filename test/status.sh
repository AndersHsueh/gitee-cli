#!/usr/bin/env bash
# =============================================================================
# gitee status 模块测试
# 测试跨仓库状态查询命令
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
    skip "未登录，跳过 status 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# status --help (status 是 leaf 命令，无子命令)
# ---------------------------------------------------------------------------
section "gitee status --help"
if "$BINARY" status --help > /dev/null 2>&1; then
    pass "status --help 正常显示"
else
    fail "status --help 失败 (需重新编译二进制以启用 status 模块)"
fi

# ---------------------------------------------------------------------------
# status 直接执行
# ---------------------------------------------------------------------------
section "gitee status 执行"
output=$("$BINARY" status 2>&1 || true)
if echo "$output" | grep -qiE "issue|pull request|assigned|review"; then
    pass "status 输出包含状态信息"
else
    fail "status 输出格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# status --output json
# ---------------------------------------------------------------------------
section "gitee status --output json"
output=$("$BINARY" status --output json 2>&1 || true)
if echo "$output" | grep -qE '"issues"|"pull_requests"|\['; then
    pass "status --output json 输出有效"
else
    fail "status --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ status 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ status 模块测试有失败项${NC}"
    exit 1
fi
