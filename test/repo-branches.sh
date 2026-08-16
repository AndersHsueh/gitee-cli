#!/usr/bin/env bash
# =============================================================================
# gitee repo branches 测试
# 测试列出仓库分支
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
    skip "未登录，跳过 repo branches 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# repo branches --help
# ---------------------------------------------------------------------------
section "gitee repo branches --help"
"$BINARY" repo branches --help > /dev/null 2>&1 && pass "repo branches --help 正常" || fail "repo branches --help 失败"

# ---------------------------------------------------------------------------
# repo branches（需要有效的仓库）
# ---------------------------------------------------------------------------
section "repo branches --repo owner/repo（占位测试）"
skip "repo branches 需要有效的仓库参数，此处仅为占位"
skip "实际使用: gitee repo branches --repo owner/repo"

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "=== repo branches 测试汇总 ==="
echo -e "  ${GREEN}通过: $passed${NC}"
echo -e "  ${RED}失败: $failed${NC}"
echo -e "  ${YELLOW}跳过: $skipped${NC}"

if [ $failed -gt 0 ]; then
    exit 1
fi
exit 0
