#!/usr/bin/env bash
# =============================================================================
# gitee release 模块测试
# 测试 Release 管理相关命令
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
    skip "未登录，跳过 release 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# release --help
# ---------------------------------------------------------------------------
section "gitee release --help"
if "$BINARY" release --help > /dev/null 2>&1; then
    pass "release --help 正常显示"
else
    fail "release --help 失败 (需重新编译二进制以启用 release 模块)"
fi

# ---------------------------------------------------------------------------
# release 子命令列表
# ---------------------------------------------------------------------------
section "gitee release 子命令验证"
expected_commands=("list" "create" "view" "edit" "delete" "download" "upload" "delete-asset")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" release "$cmd" --help > /dev/null 2>&1; then
        pass "release $cmd 存在"
    else
        fail "release $cmd 不存在"
    fi
done

# ---------------------------------------------------------------------------
# release list --help
# ---------------------------------------------------------------------------
section "gitee release list --help"
if "$BINARY" release list --help > /dev/null 2>&1; then
    pass "release list --help 正常"
else
    fail "release list --help 失败"
fi

# ---------------------------------------------------------------------------
# release list --output json
# ---------------------------------------------------------------------------
section "gitee release list --output json"
output=$("$BINARY" release list --repo gitee/gitee --output json --limit 5 2>&1 || true)
if echo "$output" | grep -qE '"tag_name"|"name"|\['; then
    pass "release list --output json 输出有效"
else
    fail "release list --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# release create --help 参数验证
# ---------------------------------------------------------------------------
section "gitee release create --help"
help_output=$("$BINARY" release create --help 2>&1)
expected_args=("tag" "title" "notes" "draft" "prerelease")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "release create --help 包含 $arg 参数"
    else
        fail "release create --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# release view --help
# ---------------------------------------------------------------------------
section "gitee release view --help"
if "$BINARY" release view --help > /dev/null 2>&1; then
    pass "release view --help 正常"
else
    fail "release view --help 失败"
fi

# ---------------------------------------------------------------------------
# release delete --help
# ---------------------------------------------------------------------------
section "gitee release delete --help"
if "$BINARY" release delete --help > /dev/null 2>&1; then
    pass "release delete --help 正常"
else
    fail "release delete --help 失败"
fi

# ---------------------------------------------------------------------------
# release download --help
# ---------------------------------------------------------------------------
section "gitee release download --help"
if "$BINARY" release download --help > /dev/null 2>&1; then
    pass "release download --help 正常"
else
    fail "release download --help 失败"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ release 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ release 模块测试有失败项${NC}"
    exit 1
fi
