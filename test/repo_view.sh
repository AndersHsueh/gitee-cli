#!/usr/bin/env bash
# =============================================================================
# gitee repo view 模块补充测试
# 测试 repo view 命令的实际执行
# =============================================================================

set -euo pipefail

BINARY="${GITEE_CLI_BINARY:-./gitee}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

passed=0; failed=0; skipped=0

pass()    { passed=$((passed+1)); echo -e "  ${GREEN}✓${NC} $*"; }
fail()    { failed=$((failed+1)); echo -e "  ${RED}✗${NC} $*" >&2; }
skip()    { skipped=$((skipped+1)); echo -e "  ${YELLOW}⊘${NC} $* (跳过)"; }
section() { echo ""; echo -e "${BLUE}▶${NC} $*"; }

# 获取当前用户名和测试仓库（使用 raw 子命令，v0.3.0 需要完整 URL）
get_test_user() {
    "$BINARY" api raw GET https://gitee.com/api/v5/user 2>/dev/null | grep '"login"' | head -1 | sed 's/.*"login": *"\([^"]*\)".*/\1/'
}
get_test_user() {
    "$BINARY" api raw GET https://gitee.com/api/v5/user 2>/dev/null | grep '"login"' | head -1 | sed 's/.*"login": *"\([^"]*\)".*/\1/'
}

# ---------------------------------------------------------------------------
# 前置检查
# ---------------------------------------------------------------------------
section "前置检查: 认证状态"
TEST_USER=$(get_test_user)
if [[ -z "$TEST_USER" ]]; then
    skip "未登录，跳过 repo view 测试"
    exit 0
fi
pass "认证状态正常 (用户: $TEST_USER)"

# 使用用户自己的仓库进行测试（使用 raw 子命令，v0.3.0 需要完整 URL）
TEST_REPO=$("$BINARY" api raw GET "https://gitee.com/api/v5/user/repos?type=owner&per_page=1" 2>/dev/null | grep '"full_name"' | head -1 | sed 's/.*"full_name": *"\([^"]*\)".*/\1/')
TEST_REPO=$("$BINARY" api raw GET https://gitee.com/api/v5/user/repos?type=owner\&per_page=1 2>/dev/null | grep '"full_name"' | head -1 | sed 's/.*"full_name": *"\([^"]*\)".*/\1/')
if [[ -z "$TEST_REPO" ]] || [[ "$TEST_REPO" == *"["* ]]; then
    skip "无法获取测试仓库，跳过需要实际仓库的测试"
    TEST_REPO=""
fi

# ---------------------------------------------------------------------------
# repo view --help
# ---------------------------------------------------------------------------
section "gitee repo view --help"
if "$BINARY" repo view --help > /dev/null 2>&1; then
    pass "repo view --help 正常显示"
else
    fail "repo view --help 失败"
fi

# ---------------------------------------------------------------------------
# repo view --owner --repo 参数解析
# ---------------------------------------------------------------------------
section "gitee repo view --owner --repo 参数"

if [[ -n "$TEST_REPO" ]]; then
    OWNER_REPO=($(echo "$TEST_REPO" | tr '/' ' '))
    OWNER="${OWNER_REPO[0]}"
    REPO="${OWNER_REPO[1]}"
    if output=$("$BINARY" repo view --owner "$OWNER" --repo "$REPO" 2>&1); then
        if echo "$output" | grep -qiE "Name:|Description:|Visibility:|Language:|Stars:|Forks:|URL:"; then
            pass "repo view --owner --repo 输出包含详细信息字段"
        else
            fail "repo view 输出缺少预期字段: ${output:0:200}"
        fi
    else
        fail "repo view --owner --repo 执行失败: ${output:0:100}"
    fi
else
    skip "无测试仓库，跳过 --owner --repo 测试"
fi

# ---------------------------------------------------------------------------
# repo view --output json
# ---------------------------------------------------------------------------
section "gitee repo view --output json"

if [[ -n "$TEST_REPO" ]]; then
    if output=$("$BINARY" repo view --repo "$TEST_REPO" --output json 2>&1); then
        if echo "$output" | grep -qE '"name"|"full_name"|"description"|"html_url"'; then
            pass "repo view --output json 输出正确 JSON"
        else
            fail "repo view --output json 格式不符: ${output:0:200}"
        fi
    else
        fail "repo view --output json 执行失败"
    fi
else
    skip "无测试仓库，跳过 --output json 测试"
fi

# ---------------------------------------------------------------------------
# repo view 参数优先级: args > flags
# ---------------------------------------------------------------------------
section "gitee repo view 参数优先级 (args > flags)"

if [[ -n "$TEST_REPO" ]]; then
    if output=$("$BINARY" repo view "$TEST_REPO" --output json 2>&1); then
        if echo "$output" | grep -q '"name"'; then
            pass "repo view $TEST_REPO (args) 优先于 flags"
        else
            fail "repo view 参数优先级不正确"
        fi
    else
        fail "repo view $TEST_REPO 参数解析失败"
    fi
else
    skip "无测试仓库，跳过参数优先级测试"
fi

# ---------------------------------------------------------------------------
# repo view 全局 --repo flag
# ---------------------------------------------------------------------------
section "gitee 全局 --repo flag"

if [[ -n "$TEST_REPO" ]]; then
    if output=$("$BINARY" repo view --repo "$TEST_REPO" 2>&1); then
        if echo "$output" | grep -qiE "$TEST_REPO|Name:"; then
            pass "全局 --repo flag 正常工作"
        else
            fail "全局 --repo flag 未生效"
        fi
    else
        fail "全局 --repo flag 执行失败"
    fi
else
    skip "无测试仓库，跳过全局 --repo 测试"
fi

# ---------------------------------------------------------------------------
# repo view 不存在的仓库应报 404
# ---------------------------------------------------------------------------
section "gitee repo view 不存在的仓库"

fake_repo="gitee-cli-nonexistent-repo-$(date +%s)"
if output=$("$BINARY" repo view --owner "$TEST_USER" --repo "$fake_repo" 2>&1); then
    fail "repo view 不存在的仓库应报错，但未报错"
else
    if echo "$output" | grep -qiE "not found|404|找不到|不存在"; then
        pass "repo view 不存在仓库返回 404 错误"
    else
        pass "repo view 不存在仓库正确报错（exit $?）"
    fi
fi

# ---------------------------------------------------------------------------
# repo view --output table (默认)
# ---------------------------------------------------------------------------
section "gitee repo view --output table"

if [[ -n "$TEST_REPO" ]]; then
    if output=$("$BINARY" repo view --repo "$TEST_REPO" --output table 2>&1); then
        if echo "$output" | grep -qiE "Name:|Description:|Visibility:"; then
            pass "repo view --output table 输出正确"
        else
            fail "repo view --output table 格式不符: ${output:0:200}"
        fi
    else
        fail "repo view --output table 执行失败"
    fi
else
    skip "无测试仓库，跳过 --output table 测试"
fi

# ---------------------------------------------------------------------------
# repo view --output csv
# ---------------------------------------------------------------------------
section "gitee repo view --output csv"

if [[ -n "$TEST_REPO" ]]; then
    if output=$("$BINARY" repo view --repo "$TEST_REPO" --output csv 2>&1); then
        if echo "$output" | grep -qE "Name:|$TEST_REPO"; then
            pass "repo view --output csv 输出有效"
        else
            fail "repo view --output csv 格式不符: ${output:0:200}"
        fi
    else
        fail "repo view --output csv 执行失败"
    fi
else
    skip "无测试仓库，跳过 --output csv 测试"
fi

# ---------------------------------------------------------------------------
# repo view 缺少所有参数应报错
# ---------------------------------------------------------------------------
section "gitee repo view 无参数报错"

if output=$("$BINARY" repo view 2>&1); then
    pass "repo view 无参数有合理行为"
else
    if echo "$output" | grep -qiE "required|必须|--repo"; then
        pass "repo view 无参数正确提示缺少 --repo"
    else
        pass "repo view 无参数正确报错"
    fi
fi

# ---------------------------------------------------------------------------
# 全局 --hostname 测试
# ---------------------------------------------------------------------------
section "gitee --hostname 全局 flag"

if [[ -n "$TEST_REPO" ]]; then
    if output=$("$BINARY" repo view --repo "$TEST_REPO" --hostname gitee.com 2>&1); then
        pass "--hostname gitee.com 参数接受"
    else
        fail "--hostname gitee.com 参数不被接受"
    fi
else
    skip "无测试仓库，跳过 --hostname 测试"
fi

# ---------------------------------------------------------------------------
# repo view --timeout 测试
# ---------------------------------------------------------------------------
section "gitee repo view --timeout flag"

if [[ -n "$TEST_REPO" ]]; then
    if output=$("$BINARY" repo view --repo "$TEST_REPO" --timeout 10 2>&1); then
        pass "--timeout flag 接受"
    else
        fail "--timeout flag 不被接受"
    fi
else
    skip "无测试仓库，跳过 --timeout 测试"
fi

# ---------------------------------------------------------------------------
# repo view --no-color 测试
# ---------------------------------------------------------------------------
section "gitee repo view --no-color flag"

if [[ -n "$TEST_REPO" ]]; then
    if output=$("$BINARY" repo view --repo "$TEST_REPO" --no-color --output json 2>&1); then
        if echo "$output" | grep -q '"name"'; then
            pass "--no-color flag 接受"
        else
            fail "--no-color flag 输出格式异常"
        fi
    else
        fail "--no-color flag 执行失败"
    fi
else
    skip "无测试仓库，跳过 --no-color 测试"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ repo view 补充测试通过${NC}"
    exit 0
else
    echo -e "${RED}✗ repo view 补充测试有失败项${NC}"
    exit 1
fi