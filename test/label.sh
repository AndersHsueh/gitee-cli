#!/usr/bin/env bash
# =============================================================================
# gitee label 模块测试
# 测试标签管理相关命令
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
    skip "未登录，跳过 label 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# label --help
# ---------------------------------------------------------------------------
section "gitee label --help"
if "$BINARY" label --help > /dev/null 2>&1; then
    pass "label --help 正常显示"
else
    fail "label --help 失败 (需重新编译二进制以启用 label 模块)"
fi

# ---------------------------------------------------------------------------
# label 子命令列表
# ---------------------------------------------------------------------------
section "gitee label 子命令验证"
expected_commands=("list" "create" "view" "edit" "delete" "clone")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" label "$cmd" --help > /dev/null 2>&1; then
        pass "label $cmd 存在"
    else
        fail "label $cmd 不存在"
    fi
done

# ---------------------------------------------------------------------------
# label list --help
# ---------------------------------------------------------------------------
section "gitee label list --help"
if "$BINARY" label list --help > /dev/null 2>&1; then
    pass "label list --help 正常"
else
    fail "label list --help 失败"
fi

# ---------------------------------------------------------------------------
# label list --output json
# ---------------------------------------------------------------------------
section "gitee label list --output json"
TEST_REPO="${GITEE_TEST_REPO:-andershsueh/gitee-cli}"
output=$("$BINARY" label list --repo "$TEST_REPO" --output json --limit 5 2>&1 || true)
if echo "$output" | grep -qiE '404|403|not found|forbidden'; then
    skip "无法读取 $TEST_REPO 的 label（设置 GITEE_TEST_REPO 指向自己的仓库）"
elif echo "$output" | grep -qE '"name"|"color"|^\['; then
    pass "label list --output json 输出有效"
else
    fail "label list --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# label create --help 参数验证
# ---------------------------------------------------------------------------
section "gitee label create --help"
help_output=$("$BINARY" label create --help 2>&1)
expected_args=("--name" "--color")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q -- "$arg"; then
        pass "label create --help 包含 $arg 参数"
    else
        fail "label create --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# label view --help
# ---------------------------------------------------------------------------
section "gitee label view --help"
if "$BINARY" label view --help > /dev/null 2>&1; then
    pass "label view --help 正常"
else
    fail "label view --help 失败"
fi

# ---------------------------------------------------------------------------
# label edit --help
# ---------------------------------------------------------------------------
section "gitee label edit --help"
if "$BINARY" label edit --help > /dev/null 2>&1; then
    pass "label edit --help 正常"
else
    fail "label edit --help 失败"
fi

# ---------------------------------------------------------------------------
# label delete --help
# ---------------------------------------------------------------------------
section "gitee label delete --help"
if "$BINARY" label delete --help > /dev/null 2>&1; then
    pass "label delete --help 正常"
else
    fail "label delete --help 失败"
fi

# ---------------------------------------------------------------------------
# label clone --help
# ---------------------------------------------------------------------------
section "gitee label clone --help"
if "$BINARY" label clone --help > /dev/null 2>&1; then
    pass "label clone --help 正常"
else
    fail "label clone --help 失败"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ label 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ label 模块测试有失败项${NC}"
    exit 1
fi
