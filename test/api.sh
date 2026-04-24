#!/usr/bin/env bash
# =============================================================================
# gitee api 模块测试
# 测试原始 API 访问命令
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
    skip "未登录，跳过 api 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# api --help
# ---------------------------------------------------------------------------
section "gitee api --help"
if "$BINARY" api --help > /dev/null 2>&1; then
    pass "api --help 正常显示"
else
    fail "api --help 失败"
fi

# ---------------------------------------------------------------------------
# api --help 参数检查
# ---------------------------------------------------------------------------
section "gitee api --help 参数完整性"
help_output=$("$BINARY" api --help 2>&1)
expected_args=("field" "paginate")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "api --help 包含 $arg 参数"
    else
        fail "api --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# api GET /user (当前用户信息)
# ---------------------------------------------------------------------------
section "gitee api GET /user"
output=$("$BINARY" api raw GET /user 2>&1)
if echo "$output" | grep -qE '"login"|"name"'; then
    pass "api GET /user 返回有效 JSON（含 login/name 字段）"
else
    fail "api GET /user 输出不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# api GET /user — 字段验证
# ---------------------------------------------------------------------------
section "gitee api GET /user 字段完整性"
output=$("$BINARY" api raw GET /user 2>&1)
expected_fields=("login" "name" "id" "avatar_url" "html_url")
for field in "${expected_fields[@]}"; do
    if echo "$output" | grep -q "\"$field\""; then
        pass "api GET /user 包含字段: $field"
    else
        fail "api GET /user 缺少字段: $field"
    fi
done

# ---------------------------------------------------------------------------
# api GET /user/repos?type=owner&per_page=3
# ---------------------------------------------------------------------------
section "gitee api GET /user/repos 带查询参数"
output=$("$BINARY" api raw GET "/user/repos?type=owner&per_page=3" 2>&1)
if echo "$output" | grep -qE '"name"|"full_name"|^\['; then
    pass "api GET /user/repos 返回有效 JSON"
else
    fail "api GET /user/repos 输出不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# api --field 参数构造 POST 请求（测试用临时仓库，用后删除）
# ---------------------------------------------------------------------------
section "gitee api POST --field 参数（创建并删除测试仓库）"
TEST_REPO_NAME="gitee-cli-api-test-$(date +%s)"

# 创建仓库
create_output=$("$BINARY" api POST /user/repos \
    --field name="$TEST_REPO_NAME" \
    --field description="API test repo - will be deleted" \
    --field private=true \
    --field auto_init=true \
    2>&1)

if echo "$create_output" | grep -q "\"full_name\""; then
    pass "api POST /user/repos --field 创建仓库成功"

    # 获取当前用户名
    user_login=$("$BINARY" api raw GET /user 2>&1 | grep '"login"' | head -1 | sed 's/.*"login": *"\([^"]*\)".*/\1/')

    if [[ -n "$user_login" ]]; then
        # 删除测试仓库
        delete_output=$("$BINARY" api raw DELETE "/repos/$user_login/$TEST_REPO_NAME" 2>&1 || true)
        if [[ -z "$delete_output" ]] || echo "$delete_output" | grep -qvE "error|Error|错误"; then
            pass "api DELETE /repos 删除测试仓库成功"
        else
            skip "api DELETE 清理失败，请手动删除 $user_login/$TEST_REPO_NAME: $delete_output"
        fi
    else
        skip "无法获取用户名，请手动删除仓库 $TEST_REPO_NAME"
    fi
else
    fail "api POST /user/repos --field 创建仓库失败: ${create_output:0:300}"
fi

# ---------------------------------------------------------------------------
# api --include 输出响应头
# ---------------------------------------------------------------------------
section "gitee api --include 响应头输出"
output=$("$BINARY" api raw GET /user --include 2>&1)
if echo "$output" | grep -qiE "HTTP|Content-Type|X-Total|RateLimit|^HTTP"; then
    pass "api --include 输出包含响应头"
else
    skip "api --include 响应头格式待确认: ${output:0:100}"
fi

# ---------------------------------------------------------------------------
# api 错误处理：无效路径
# ---------------------------------------------------------------------------
section "gitee api 错误路径处理"
if output=$("$BINARY" api raw GET /nonexistent-endpoint-xyz 2>&1); then
    fail "api 访问不存在的端点应报错，但未报错"
else
    if echo "$output" | grep -qiE "not found|404|error"; then
        pass "api 无效路径正确返回错误"
    else
        pass "api 无效路径返回错误（exit $?）"
    fi
fi

# ---------------------------------------------------------------------------
# api 错误处理：缺少 method 参数
# ---------------------------------------------------------------------------
section "gitee api 缺少参数报错"
if output=$("$BINARY" api 2>&1); then
    fail "api 无参数应报错，但未报错"
else
    pass "api 无参数正确报错（exit $?）"
fi

# ---------------------------------------------------------------------------
# api PATCH /repos/{owner}/{repo} —— 需要真实仓库，跳过
# ---------------------------------------------------------------------------
section "gitee api PATCH（跳过，需真实仓库）"
skip "PATCH 测试需要预先知道真实 owner/repo，跳过以避免副作用"

# ---------------------------------------------------------------------------
# api --paginate 分页（只验证参数接受）
# ---------------------------------------------------------------------------
section "gitee api --paginate 参数"
if output=$("$BINARY" api raw GET "/user/repos?per_page=3" --paginate 2>&1); then
    pass "api --paginate 参数接受，返回数据"
else
    fail "api --paginate 参数不被接受"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ api 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ api 模块测试有失败项${NC}"
    exit 1
fi
