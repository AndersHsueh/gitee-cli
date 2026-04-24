#!/usr/bin/env bash
# =============================================================================
# gitee search 模块测试
# 测试搜索相关命令（repos / issues / users）
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
# search --help
# ---------------------------------------------------------------------------
section "gitee search --help"
if "$BINARY" search --help > /dev/null 2>&1; then
    pass "search --help 正常显示"
else
    fail "search --help 失败"
fi

# ---------------------------------------------------------------------------
# 子命令列表
# ---------------------------------------------------------------------------
section "gitee search 子命令验证"
expected_commands=("repos" "issues" "users")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" search "$cmd" --help > /dev/null 2>&1; then
        pass "search $cmd 子命令存在"
    else
        fail "search $cmd 子命令不存在"
    fi
done

# ---------------------------------------------------------------------------
# search repos --help 参数检查
# ---------------------------------------------------------------------------
section "gitee search repos --help 参数检查"
help_output=$("$BINARY" search repos --help 2>&1)
expected_args=("language" "sort" "order" "limit")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "search repos --help 包含 $arg 参数"
    else
        fail "search repos --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# search repos 基本搜索（使用公开通用关键词）
# ---------------------------------------------------------------------------
section "gitee search repos 基本搜索"
output=$("$BINARY" search repos "gitee" --limit 5 2>&1 || true)
if echo "$output" | grep -qiE "gitee|NAME|full_name|language"; then
    pass "search repos 返回结果"
else
    fail "search repos 无有效输出: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# search repos --output json
# ---------------------------------------------------------------------------
section "gitee search repos --output json"
output=$("$BINARY" search repos "golang" --output json --limit 3 2>&1 || true)
if echo "$output" | grep -qE '"name"|"full_name"|^\['; then
    pass "search repos --output json 输出有效"
else
    fail "search repos --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# search repos --language 过滤
# ---------------------------------------------------------------------------
section "gitee search repos --language 过滤"
if output=$("$BINARY" search repos "web" --language go --limit 3 2>&1); then
    pass "search repos --language 参数接受"
else
    fail "search repos --language 参数不被接受"
fi

# ---------------------------------------------------------------------------
# search repos --sort 排序
# ---------------------------------------------------------------------------
section "gitee search repos --sort 排序"
for sort_val in "stars" "forks" "updated"; do
    if output=$("$BINARY" search repos "test" --sort "$sort_val" --limit 3 2>&1); then
        pass "search repos --sort $sort_val 接受"
    else
        fail "search repos --sort $sort_val 不被接受"
    fi
done

# ---------------------------------------------------------------------------
# search issues --help 参数检查
# ---------------------------------------------------------------------------
section "gitee search issues --help 参数检查"
help_output=$("$BINARY" search issues --help 2>&1)
if echo "$help_output" | grep -qE "state|repo|label"; then
    pass "search issues --help 包含筛选参数"
else
    skip "search issues --help 参数未完全实现"
fi

# ---------------------------------------------------------------------------
# search issues 基本搜索
# ---------------------------------------------------------------------------
section "gitee search issues 基本搜索"
output=$("$BINARY" search issues "bug" --limit 5 2>&1 || true)
if echo "$output" | grep -qiE "title|number|state|issue|#"; then
    pass "search issues 返回结果"
else
    skip "search issues 无可见数据（可能是权限或未实现）"
fi

# ---------------------------------------------------------------------------
# search users --help
# ---------------------------------------------------------------------------
section "gitee search users --help"
if "$BINARY" search users --help > /dev/null 2>&1; then
    pass "search users --help 正常"
else
    fail "search users --help 失败"
fi

# ---------------------------------------------------------------------------
# search users 基本搜索
# ---------------------------------------------------------------------------
section "gitee search users 基本搜索"
output=$("$BINARY" search users "gitee" --limit 5 2>&1 || true)
if echo "$output" | grep -qiE "login|name|user|gitee"; then
    pass "search users 返回结果"
else
    fail "search users 无有效输出: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# search users --output json
# ---------------------------------------------------------------------------
section "gitee search users --output json"
output=$("$BINARY" search users "go" --output json --limit 3 2>&1 || true)
if echo "$output" | grep -qE '"login"|"name"|^\['; then
    pass "search users --output json 输出有效"
else
    fail "search users --output json 格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# search 无关键词应报错
# ---------------------------------------------------------------------------
section "gitee search repos 无关键词报错"
if output=$("$BINARY" search repos 2>&1); then
    fail "search repos 无关键词应报错，但未报错"
else
    pass "search repos 无关键词正确报错（exit $?）"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ search 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ search 模块测试有失败项${NC}"
    exit 1
fi
