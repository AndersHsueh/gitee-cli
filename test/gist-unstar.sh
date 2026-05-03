#!/usr/bin/env bash
# =============================================================================
# gitee gist unstar 测试
# 测试取消 Star 代码片段
# =============================================================================

set -euo pipefail

BINARY="${GITEE_CLI_BINARY:-./gitee}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

passed=0; failed=0; skipped=0

pass() { ((passed++)); echo -e "  ${GREEN}✓${NC} $*"; }
fail() { ((failed++)); echo -e "  ${RED}✗${NC} $*" >&2; }
skip() { ((skipped++)); echo -e "  ${YELLOW}⊘${NC} $* (跳过)"; }
section() { echo ""; echo -e "${BLUE}▶${NC} $*"; }

# ---------------------------------------------------------------------------
# 前置检查
# ---------------------------------------------------------------------------
section "前置检查: 认证状态"
if ! "$BINARY" auth status > /dev/null 2>&1; then
    skip "未登录，跳过 gist unstar 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# gist unstar --help
# ---------------------------------------------------------------------------
section "gitee gist unstar --help"
"$BINARY" gist unstar --help > /dev/null 2>&1 && pass "gist unstar --help 正常" || fail "gist unstar --help 失败"

# ---------------------------------------------------------------------------
# gist unstar（需要有效的 gist ID，此处为占位测试）
# ---------------------------------------------------------------------------
section "gist unstar <id>（占位测试）"
skip "gist unstar 需要有效的 gist ID，且会改变远程状态，此处仅为占位"
skip "实际使用: gitee gist unstar <gistID>"

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "=== gist unstar 测试汇总 ==="
echo -e "  ${GREEN}通过: $passed${NC}"
echo -e "  ${RED}失败: $failed${NC}"
echo -e "  ${YELLOW}跳过: $skipped${NC}"

if [ $failed -gt 0 ]; then
    exit 1
fi
exit 0
