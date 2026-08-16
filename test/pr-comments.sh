#!/usr/bin/env bash
# =============================================================================
# gitee pr comments 测试
# 测试列出 Pull Request 评论
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
    skip "未登录，跳过 pr comments 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# pr comments --help
# ---------------------------------------------------------------------------
section "gitee pr comments --help"
"$BINARY" pr comments --help > /dev/null 2>&1 && pass "pr comments --help 正常" || fail "pr comments --help 失败"

# ---------------------------------------------------------------------------
# pr comments（需要有效的 PR 编号和仓库）
# ---------------------------------------------------------------------------
section "pr comments <number> --repo owner/repo（占位测试）"
skip "pr comments 需要有效的 PR 编号和仓库，此处仅为占位"
skip "实际使用: gitee pr comments 1 --repo owner/repo"

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "=== pr comments 测试汇总 ==="
echo -e "  ${GREEN}通过: $passed${NC}"
echo -e "  ${RED}失败: $failed${NC}"
echo -e "  ${YELLOW}跳过: $skipped${NC}"

if [ $failed -gt 0 ]; then
    exit 1
fi
exit 0
