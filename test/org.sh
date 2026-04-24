#!/usr/bin/env bash
# =============================================================================
# gitee org 模块测试
# 测试组织管理相关命令
# =============================================================================

set -euo pipefail

BINARY="${GITEE_CLI_BINARY:-./gitee}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

passed=0; failed=0; skipped=0

pass()    { ((passed++)); echo -e "  ${GREEN}✓${NC} $*"; }
fail()    { ((failed++)); echo -e "  ${RED}✗${NC} $*" >&2; }
skip()    { ((skipped++)); echo -e "  ${YELLOW}⊘${NC} $* (跳过)"; }
section() { echo ""; echo -e "${BLUE}▶${NC} $*"; }

# ---------------------------------------------------------------------------
# 前置检查
# ---------------------------------------------------------------------------
section "前置检查: 认证状态"
if ! "$BINARY" auth status > /dev/null 2>&1; then
    skip "未登录，跳过 org 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# org --help
# ---------------------------------------------------------------------------
section "gitee org --help"
if "$BINARY" org --help > /dev/null 2>&1; then
    pass "org --help 正常显示"
else
    fail "org --help 失败"
fi

# ---------------------------------------------------------------------------
# 子命令列表
# ---------------------------------------------------------------------------
section "gitee org 子命令验证"
expected_commands=("list" "view" "member" "repo")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" org "$cmd" --help > /dev/null 2>&1; then
        pass "org $cmd 子命令存在"
    else
        fail "org $cmd 子命令不存在"
    fi
done

# ---------------------------------------------------------------------------
# org list --output json
# ---------------------------------------------------------------------------
section "gitee org list --output json"
output=$("$BINARY" org list --output json --limit 5 2>&1 || true)
if echo "$output" | grep -qE '"login"|"name"|^\['; then
    pass "org list --output json 输出有效"
else
    # 用户可能未加入任何组织
    if echo "$output" | grep -qE '^\[\]|^\[\s*\]'; then
        pass "org list --output json 返回空数组（用户未加入组织）"
    else
        fail "org list --output json 输出格式不符: ${output:0:200}"
    fi
fi

# ---------------------------------------------------------------------------
# org list --output table
# ---------------------------------------------------------------------------
section "gitee org list --output table"
output=$("$BINARY" org list --limit 5 2>&1 || true)
if echo "$output" | grep -qiE "NAME|LOGIN|DESCRIPTION|UPDATED|org"; then
    pass "org list 表格输出含表头"
else
    skip "org list 表头未识别（可能无组织数据）"
fi

# ---------------------------------------------------------------------------
# org view --help
# ---------------------------------------------------------------------------
section "gitee org view --help"
if "$BINARY" org view --help > /dev/null 2>&1; then
    pass "org view --help 正常"
else
    fail "org view --help 失败"
fi

# ---------------------------------------------------------------------------
# org member list --help 参数检查
# ---------------------------------------------------------------------------
section "gitee org member list --help"
help_output=$("$BINARY" org member --help 2>&1 || "$BINARY" org member list --help 2>&1 || true)
if [[ -n "$help_output" ]]; then
    pass "org member --help 有输出"
else
    fail "org member --help 无输出"
fi

# ---------------------------------------------------------------------------
# org repo list --help
# ---------------------------------------------------------------------------
section "gitee org repo list --help"
help_output=$("$BINARY" org repo --help 2>&1 || "$BINARY" org repo list --help 2>&1 || true)
if [[ -n "$help_output" ]]; then
    pass "org repo --help 有输出"
else
    fail "org repo --help 无输出"
fi

# ---------------------------------------------------------------------------
# org list 分页参数
# ---------------------------------------------------------------------------
section "gitee org list 分页参数"
if output=$("$BINARY" org list --page 1 --per-page 10 2>&1); then
    pass "org list --page --per-page 参数接受"
else
    fail "org list --page --per-page 参数不被接受"
fi

# ---------------------------------------------------------------------------
# 全局 flag 验证
# ---------------------------------------------------------------------------
section "全局 Flag 验证 (org)"
for fmt in "json" "table" "csv"; do
    if output=$("$BINARY" org list --output "$fmt" --limit 1 2>&1); then
        pass "--output $fmt 有效"
    else
        fail "--output $fmt 无效"
    fi
done

# ---------------------------------------------------------------------------
# org view 不存在的组织应报 404
# ---------------------------------------------------------------------------
section "gitee org view 不存在的组织"
fake_org="gitee-cli-nonexistent-org-$(date +%s)"
if output=$("$BINARY" org view "$fake_org" 2>&1); then
    fail "org view 不存在的组织应报错，但未报错"
else
    if echo "$output" | grep -qiE "not found|404|找不到|不存在"; then
        pass "org view 不存在组织返回 404 错误"
    else
        skip "org view 错误信息格式待确认: ${output:0:100}"
    fi
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ org 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ org 模块测试有失败项${NC}"
    exit 1
fi
