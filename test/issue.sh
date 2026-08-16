#!/usr/bin/env bash
# =============================================================================
# gitee issue 模块测试
# 测试 Issue 管理相关命令
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
    skip "未登录，跳过 issue 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# issue --help
# ---------------------------------------------------------------------------
section "gitee issue --help"
"$BINARY" issue --help > /dev/null 2>&1 && pass "issue --help 正常" || fail "issue --help 失败"

# ---------------------------------------------------------------------------
# issue 子命令列表
# ---------------------------------------------------------------------------
section "gitee issue 子命令验证"
expected_commands=("list" "create" "view" "edit" "close" "reopen" "comment" "label")
for cmd in "${expected_commands[@]}"; do
    "$BINARY" issue "$cmd" --help > /dev/null 2>&1 && pass "issue $cmd 存在" || fail "issue $cmd 不存在"
done

# ---------------------------------------------------------------------------
# issue label --help (label 是 issue 的子命令)
# ---------------------------------------------------------------------------
section "gitee issue label --help"
"$BINARY" issue label --help > /dev/null 2>&1 && pass "issue label --help 正常" || fail "issue label --help 失败"

# ---------------------------------------------------------------------------
# issue list --output json
# ---------------------------------------------------------------------------
section "gitee issue list --output json"
output=$("$BINARY" issue list --repo andershsueh/outsourcing-ms --output json --limit 5 2>&1 || true)
if echo "$output" | grep -qE '"number"|"title"|"state"|\['; then
    pass "issue list --output json 输出正确"
else
    fail "issue list --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# issue list --output table
# ---------------------------------------------------------------------------
section "gitee issue list --output table"
output=$("$BINARY" issue list --repo andershsueh/outsourcing-ms --limit 5 --output table 2>&1 || true)
if echo "$output" | grep -qi "#\|编号\|TITLE\|Issue"; then
    pass "issue list --output table 输出包含表头或编号"
else
    fail "issue list --output table 输出缺少表头"
fi

# ---------------------------------------------------------------------------
# issue list --state
# ---------------------------------------------------------------------------
section "gitee issue list --state"
for state in "open" "closed" "all"; do
    if output=$("$BINARY" issue list --repo andershsueh/outsourcing-ms --state "$state" --limit 3 2>&1); then
        pass "issue list --state $state 执行成功"
    else
        fail "issue list --state $state 执行失败"
    fi
done

# ---------------------------------------------------------------------------
# issue list --labels
# ---------------------------------------------------------------------------
section "gitee issue list --labels"
if output=$("$BINARY" issue list --repo andershsueh/outsourcing-ms --labels "bug" --limit 3 2>&1); then
    pass "issue list --labels bug 执行成功"
else
    fail "issue list --labels bug 执行失败"
fi

# ---------------------------------------------------------------------------
# issue list --assignee
# ---------------------------------------------------------------------------
section "gitee issue list --assignee"
if output=$("$BINARY" issue list --repo andershsueh/outsourcing-ms --assignee gitee --limit 3 2>&1); then
    pass "issue list --assignee 执行成功"
else
    fail "issue list --assignee 执行失败"
fi

# ---------------------------------------------------------------------------
# issue list --milestone
# ---------------------------------------------------------------------------
section "gitee issue list --milestone"
if output=$("$BINARY" issue list --repo andershsueh/outsourcing-ms --milestone "v1.0" --limit 3 2>&1); then
    pass "issue list --milestone 执行成功"
else
    fail "issue list --milestone 执行失败"
fi

# ---------------------------------------------------------------------------
# issue create --help 参数验证
# ---------------------------------------------------------------------------
section "gitee issue create --help"
help_output=$("$BINARY" issue create --help 2>&1)
expected_args=("title" "body" "labels" "assignees" "milestone")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "issue create --help 包含 $arg 参数"
    else
        fail "issue create --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# issue create 必填参数验证
# ---------------------------------------------------------------------------
section "gitee issue create 必填参数验证"

# 缺少 --title 应报错
if output=$("$BINARY" issue create --repo andershsueh/outsourcing-ms 2>&1); then
    fail "issue create 缺少 --title 应报错但未报错"
else
    pass "issue create 缺少 --title 正确报错"
fi

# ---------------------------------------------------------------------------
# issue view --help
# ---------------------------------------------------------------------------
section "gitee issue view --help"
"$BINARY" issue view --help > /dev/null 2>&1 && pass "issue view --help 正常" || fail "issue view --help 失败"

# ---------------------------------------------------------------------------
# issue edit --help
# ---------------------------------------------------------------------------
section "gitee issue edit --help"
"$BINARY" issue edit --help > /dev/null 2>&1 && pass "issue edit --help 正常" || fail "issue edit --help 失败"

# ---------------------------------------------------------------------------
# issue close --help
# ---------------------------------------------------------------------------
section "gitee issue close --help"
"$BINARY" issue close --help > /dev/null 2>&1 && pass "issue close --help 正常" || fail "issue close --help 失败"

# ---------------------------------------------------------------------------
# issue reopen --help
# ---------------------------------------------------------------------------
section "gitee issue reopen --help"
"$BINARY" issue reopen --help > /dev/null 2>&1 && pass "issue reopen --help 正常" || fail "issue reopen --help 失败"

# ---------------------------------------------------------------------------
# issue comment --help
# ---------------------------------------------------------------------------
section "gitee issue comment --help"
"$BINARY" issue comment --help > /dev/null 2>&1 && pass "issue comment --help 正常" || fail "issue comment --help 失败"

# ---------------------------------------------------------------------------
# issue label list --help
# ---------------------------------------------------------------------------
section "gitee issue label list --help"
"$BINARY" issue label list --help > /dev/null 2>&1 && pass "issue label list --help 正常" || fail "issue label list --help 失败"

# ---------------------------------------------------------------------------
# 全局 --paginate flag
# ---------------------------------------------------------------------------
section "全局 --paginate flag"
if output=$("$BINARY" issue list --repo andershsueh/outsourcing-ms --paginate --limit 20 2>&1); then
    pass "--paginate flag 接受"
else
    fail "--paginate flag 不被接受"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"
if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ issue 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ issue 模块测试有失败项${NC}"
    exit 1
fi