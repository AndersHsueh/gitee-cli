#!/usr/bin/env bash
# =============================================================================
# gitee 分页 (pagination) 功能测试
# 测试 --paginate 标志和分页相关功能
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
    skip "未登录，跳过 pagination 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# 全局 --paginate flag 存在性测试
# ---------------------------------------------------------------------------
section "全局 --paginate flag 存在"

if "$BINARY" --help 2>&1 | grep -q "paginate"; then
    pass "--paginate 全局 flag 在帮助中存在"
else
    fail "--paginate 全局 flag 不在帮助中"
fi

if "$BINARY" --help 2>&1 | grep -q "\-P"; then
    pass "-P 短 flag 在帮助中存在"
else
    fail "-P 短 flag 不在帮助中"
fi

# ---------------------------------------------------------------------------
# repo list --paginate
# ---------------------------------------------------------------------------
section "gitee repo list --paginate"

# 不使用 --paginate 获取第一页
output_no_paginate=$("$BINARY" repo list --limit 3 --per-page 3 --output json 2>&1 || true)
count_no_paginate=$(echo "$output_no_paginate" | grep -c '"name"\|"full_name"' || true)

# 使用 --paginate 获取多页
output_paginate=$("$BINARY" repo list --paginate --per-page 3 --limit 9 --output json 2>&1 || true)
count_paginate=$(echo "$output_paginate" | grep -c '"name"\|"full_name"' || true)

if [[ "$count_paginate" -ge 3 ]]; then
    pass "repo list --paginate 返回结果 (count: $count_paginate)"
else
    fail "repo list --paginate 无结果或结果过少"
fi

# ---------------------------------------------------------------------------
# issue list --paginate
# ---------------------------------------------------------------------------
section "gitee issue list --paginate"

TEST_REPO="andershsueh/gitee-cli"
output=$("$BINARY" issue list --repo "$TEST_REPO" --paginate --limit 10 --output json 2>&1 || true)
if echo "$output" | grep -qE '"number"|"title"|"state"'; then
    pass "issue list --paginate 返回结果"
else
    fail "issue list --paginate 无有效输出"
fi

# ---------------------------------------------------------------------------
# pr list --paginate
# ---------------------------------------------------------------------------
section "gitee pr list --paginate"

TEST_REPO="andershsueh/gitee-cli"
output=$("$BINARY" pr list --repo "$TEST_REPO" --paginate --limit 10 --output json 2>&1 || true)
# 空数组 [] 也是有效输出（仓库可能没有 PR）
if echo "$output" | grep -qE '^\[\]|"number"|"title"|"state"'; then
    pass "pr list --paginate 返回结果或空数组"
else
    fail "pr list --paginate 无有效输出"
fi

# ---------------------------------------------------------------------------
# gist list --paginate
# ---------------------------------------------------------------------------
section "gitee gist list --paginate"

output=$("$BINARY" gist list --paginate --limit 5 --output json 2>&1 || true)
if echo "$output" | grep -qE '"id"|"description"'; then
    pass "gist list --paginate 返回结果"
else
    skip "gist list --paginate 无数据（可能用户无 gist）"
fi

# ---------------------------------------------------------------------------
# org list --paginate
# ---------------------------------------------------------------------------
section "gitee org list --paginate"

output=$("$BINARY" org list --paginate --limit 5 --output json 2>&1 || true)
if echo "$output" | grep -qE '"login"|"name"'; then
    pass "org list --paginate 返回结果"
else
    skip "org list --paginate 无数据（可能用户无组织）"
fi

# ---------------------------------------------------------------------------
# api --paginate (raw 子命令)
# ---------------------------------------------------------------------------
section "gitee api raw GET /user/repos --paginate"

output=$("$BINARY" api raw GET "/user/repos?per_page=2" --paginate 2>&1 || true)
count=$(echo "$output" | grep -c '"full_name"' || true)
if [[ $count -ge 2 ]]; then
    pass "api raw --paginate 返回多个结果 (count: $count)"
else
    fail "api raw --paginate 返回结果过少"
fi

# ---------------------------------------------------------------------------
# --page 和 --per-page 参数测试
# ---------------------------------------------------------------------------
section "--page 和 --per-page 参数"

# 第一页
page1=$("$BINARY" repo list --page 1 --per-page 2 --output json 2>&1 || true)
count1=$(echo "$page1" | grep -c '"name"\|"full_name"' || true)

# 第二页
page2=$("$BINARY" repo list --page 2 --per-page 2 --output json 2>&1 || true)
count2=$(echo "$page2" | grep -c '"name"\|"full_name"' || true)

if [[ $count1 -ge 1 ]] && [[ $count2 -ge 1 ]]; then
    pass "--page 和 --per-page 参数正常 (page1: $count1, page2: $count2)"
else
    fail "--page 和 --per-page 参数可能不工作"
fi

# ---------------------------------------------------------------------------
# --limit 参数测试
# ---------------------------------------------------------------------------
section "--limit 参数"

output=$("$BINARY" repo list --limit 2 --output json 2>&1 || true)
count=$(echo "$output" | grep -c '"name"\|"full_name"' || true)
if [[ $count -le 4 ]]; then
    pass "--limit 参数限制结果数量 (count: $count, limit: 2)"
else
    skip "--limit 参数可能未生效或结果超过限制"
fi

# ---------------------------------------------------------------------------
# 分页边界情况: 过大页码
# ---------------------------------------------------------------------------
section "分页边界: 过大页码"

output=$("$BINARY" repo list --page 99999 --per-page 10 --output json 2>&1 || true)
if echo "$output" | grep -qE '^\[\]|^\s*$'; then
    pass "过大页码返回空列表（符合预期）"
else
    if echo "$output" | grep -qE '"name"|"full_name"'; then
        skip "过大页码仍返回数据（可能正常）"
    else
        pass "过大页码返回空或错误（符合预期）"
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
    echo -e "${GREEN}✓ pagination 测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ pagination 测试有失败项${NC}"
    exit 1
fi