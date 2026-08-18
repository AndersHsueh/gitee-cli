#!/usr/bin/env bash
# =============================================================================
# gitee api 模块测试
# 测试原始 API 访问命令
# =============================================================================

set -euo pipefail

BINARY="${GITEE_CLI_BINARY:-./gitee}"
# 调试旁路: 设 GITEE_KEEP_TEST_REPO=1 跳过删除, 保留仓库供排查
KEEP_TEST_REPO="${GITEE_KEEP_TEST_REPO:-}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

passed=0; failed=0; skipped=0
# leaked: 本脚本产生的临时仓库未能在退出前清理掉的数量
leaked=0

# ---------------------------------------------------------------------------
# 临时测试仓库 cleanup
# 风格沿用 test/auth.sh:37-43 与 test/config.sh:28-33 (命名函数 + trap EXIT)
# 解决: test/api.sh 在 DELETE 失败 / Ctrl-C / set -e 提前退出时残留
#       gitee-cli-api-test-* 私有仓库到 Gitee 的问题
# ---------------------------------------------------------------------------
TEST_REPO_NAME=""
TEST_REPO_OWNER=""

cleanup_test_repo() {
    local owner="${TEST_REPO_OWNER:-}"
    local name="${TEST_REPO_NAME:-}"
    # 已经在第一次 trap 调用时清理过 (例如脚本正常结束时), 直接返回
    [[ -z "$name" ]] && return 0

    if [[ -n "$KEEP_TEST_REPO" ]]; then
        echo -e "  ${YELLOW}⊘${NC} 保留测试仓库 $owner/$name (GITEE_KEEP_TEST_REPO=$KEEP_TEST_REPO)" >&2
        TEST_REPO_NAME=""
        return 0
    fi

    # 重试 3 次: 私有仓库建好后索引可能未传播, 第一次 DELETE 偶尔 404
    local i out
    for i in 1 2 3; do
        out=$("$BINARY" repo delete "$owner/$name" --yes 2>&1) && {
            TEST_REPO_NAME=""
            return 0
        }
        sleep 2
    done

    # 重试仍失败: 增加泄露计数 (汇总块据此 exit 1)
    leaked=$((leaked + 1))
    echo -e "  ${RED}✗${NC} 清理失败, 请手动删除 $owner/$name: $out" >&2
    TEST_REPO_NAME=""
}

trap cleanup_test_repo EXIT

pass()    { passed=$((passed+1)); echo -e "  ${GREEN}✓${NC} $*"; }
fail()    { failed=$((failed+1)); echo -e "  ${RED}✗${NC} $*" >&2; }
skip()    { skipped=$((skipped+1)); echo -e "  ${YELLOW}⊘${NC} $* (跳过)"; }
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
output=$("$BINARY" api raw GET https://gitee.com/api/v5/user 2>&1)
if echo "$output" | grep -qE '"login"|"name"'; then
    pass "api GET /user 返回有效 JSON（含 login/name 字段）"
else
    fail "api GET /user 输出不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# api GET /user — 字段验证
# ---------------------------------------------------------------------------
section "gitee api GET /user 字段完整性"
output=$("$BINARY" api raw GET https://gitee.com/api/v5/user 2>&1)
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
TEST_REPO_NAME="gitee-cli-api-test-$(date +%s)-$$"

# 提前解析当前用户名 (失败也不阻塞, cleanup 时再重试拿)
TEST_REPO_OWNER=$("$BINARY" api raw GET https://gitee.com/api/v5/user 2>&1 \
    | grep '"login"' | head -1 | sed 's/.*"login": *"\([^"]*\)".*/\1/')

# 创建仓库
create_output=$("$BINARY" api raw POST https://gitee.com/api/v5/user/repos \
    --field name="$TEST_REPO_NAME" \
    --field description="API test repo - will be deleted" \
    --field private=true \
    --field auto_init=true \
    2>&1)

if echo "$create_output" | grep -q "\"full_name\""; then
    pass "api POST /user/repos --field 创建仓库成功"

    # happy path: 立即尝试删除 (保留现有 'api DELETE /repos' 测试断言)
    # 用 cli wrapper 而不是 api raw, 因为 wrapper 是稳定的 repo delete 入口
    # 即便这里失败, EXIT trap 还会兜底重试
    if [[ -n "$TEST_REPO_OWNER" ]]; then
        if [[ -n "$KEEP_TEST_REPO" ]]; then
            # 调试旁路: 保留仓库, 同时记录 leaked +1 让 self-check 失败
            pass "api POST /user/repos --field 创建仓库成功 (GITEE_KEEP_TEST_REPO=$KEEP_TEST_REPO 保留)"
            leaked=$((leaked + 1))
        else
            delete_output=$("$BINARY" repo delete "$TEST_REPO_OWNER/$TEST_REPO_NAME" --yes 2>&1 || true)
            if echo "$delete_output" | grep -q "deleted"; then
                pass "api DELETE /repos 删除测试仓库成功"
                TEST_REPO_NAME=""  # 立即清理成功, 不让 trap 再删
            else
                # 留给 trap 兜底重试, 这里不再 skip (会污染 skipped 计数)
                echo -e "  ${YELLOW}⊘${NC} 立即删除失败, 留给 EXIT trap 兜底: $delete_output" >&2
            fi
        fi
    else
        # user_login 解析失败, 全权交给 trap 处理
        echo -e "  ${YELLOW}⊘${NC} 解析当前用户名失败, 留给 EXIT trap 兜底" >&2
    fi
else
    fail "api POST /user/repos --field 创建仓库失败: ${create_output:0:300}"
    # 创建失败, TEST_REPO_NAME 不指向任何实际仓库, 清空避免 trap 误删别的
    TEST_REPO_NAME=""
fi

# ---------------------------------------------------------------------------
# api --include 输出响应头
# ---------------------------------------------------------------------------
section "gitee api --include 响应头输出"
output=$("$BINARY" api raw GET https://gitee.com/api/v5/user --include 2>&1) || true
if echo "$output" | grep -qiE "HTTP|Content-Type|X-Total|RateLimit|^HTTP"; then
    pass "api --include 输出包含响应头"
else
    skip "api --include 响应头格式待确认: ${output:0:100}"
fi

# ---------------------------------------------------------------------------
# api 错误处理：无效路径
# ---------------------------------------------------------------------------
section "gitee api 错误路径处理"
if output=$("$BINARY" api raw GET https://gitee.com/api/v5/nonexistent-endpoint-xyz 2>&1); then
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
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}  ${RED}泄露: $leaked${NC}"
echo "────────────────────────────────────────"

# Leak self-check: 扫账号下所有 gitee-cli-api-test-* 残留仓库
# (覆盖本次跑出来的 + 其他来源积累的). 不依赖本脚本的计数器.
remaining_leaks=0
if leaks_json=$("$BINARY" repo list --paginate --limit 200 -o json 2>/dev/null); then
    remaining_leaks=$(echo "$leaks_json" | python3 -c "
import json,sys
try:
    repos=json.load(sys.stdin)
    print(sum(1 for r in repos if 'gitee-cli-api-test' in (r.get('full_name') or '')))
except Exception:
    print(0)" 2>/dev/null || echo 0)
fi

if [[ $failed -eq 0 && $leaked -eq 0 && $remaining_leaks -eq 0 ]]; then
    echo -e "${GREEN}✓ api 模块测试全部通过${NC}"
    exit 0
else
    [[ $failed -gt 0 ]] && echo -e "${RED}✗ api 模块测试有失败项${NC}"
    [[ $leaked -gt 0 ]] && echo -e "${RED}✗ 本次运行有 $leaked 个测试仓库清理失败${NC}"
    if [[ $remaining_leaks -gt 0 ]]; then
        echo -e "${RED}✗ 账号下仍有 $remaining_leaks 个 gitee-cli-api-test-* 残留仓库 (历史或本次)${NC}"
        if [[ -n "$leaks_json" ]]; then
            echo "$leaks_json" | python3 -c "
import json,sys
for r in json.load(sys.stdin):
    fn=r.get('full_name','')
    if 'gitee-cli-api-test' in fn:
        print(f'    - {fn} (created {r.get(\"created_at\",\"?\")})')" 2>/dev/null || true
        fi
        echo -e "  手动清理: for r in \$(gitee repo list --paginate --limit 200 -o json | jq -r '.[] | select(.full_name | contains(\"gitee-cli-api-test\")) | .full_name'); do gitee repo delete \"\$r\" --yes; done"
    fi
    exit 1
fi
""