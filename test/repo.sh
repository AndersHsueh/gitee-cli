#!/usr/bin/env bash
# =============================================================================
# gitee repo 模块测试
# 测试仓库管理相关命令
# =============================================================================

set -euo pipefail

BINARY="${GITEE_CLI_BINARY:-./gitee}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

passed=0
failed=0
skipped=0

pass() { ((passed++)); echo -e "  ${GREEN}✓${NC} $*"; }
fail() { ((failed++)); echo -e "  ${RED}✗${NC} $*" >&2; }
skip() { ((skipped++)); echo -e "  ${YELLOW}⊘${NC} $* (跳过)"; }

section() {
    echo ""
    echo -e "${BLUE}▶${NC} $*"
}

# ---------------------------------------------------------------------------
# 前置检查: 确认已登录
# ---------------------------------------------------------------------------
section "前置检查: 认证状态"

if ! "$BINARY" auth status > /dev/null 2>&1; then
    skip "未登录，跳过所有 repo 测试"
    echo "  运行 'gitee auth login' 完成认证后再执行此测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# repo --help
# ---------------------------------------------------------------------------
section "gitee repo --help"

if "$BINARY" repo --help > /dev/null 2>&1; then
    pass "repo --help 正常显示"
else
    fail "repo --help 失败"
fi

# ---------------------------------------------------------------------------
# repo 子命令列表
# ---------------------------------------------------------------------------
section "gitee repo 子命令验证"

expected_commands=("list" "create" "clone" "fork" "view" "edit" "delete" "browse")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" repo "$cmd" --help > /dev/null 2>&1; then
        pass "repo $cmd 子命令存在"
    else
        fail "repo $cmd 子命令不存在"
    fi
done

# ---------------------------------------------------------------------------
# repo list --output json
# ---------------------------------------------------------------------------
section "gitee repo list --output json"

output=$("$BINARY" repo list --output json --limit 5 2>&1 || true)
if echo "$output" | grep -q '"repositories"\|"name"\|"full_name"\|"id"'; then
    pass "repo list --output json 输出正确"
else
    # 即使列表为空，也应有正确的 JSON 结构
    if echo "$output" | grep -q '^\[' || echo "$output" | grep -q '{'; then
        pass "repo list --output json 返回有效 JSON（可能为空列表）"
    else
        fail "repo list --output json 输出格式不符: ${output:0:200}"
    fi
fi

# ---------------------------------------------------------------------------
# repo list --output table (默认)
# ---------------------------------------------------------------------------
section "gitee repo list --output table"

output=$("$BINARY" repo list --limit 5 --output table 2>&1 || true)
if echo "$output" | grep -q "NAME\|仓库\|NAME\|full_name"; then
    pass "repo list --output table 输出包含表头"
else
    fail "repo list --output table 输出缺少表头"
fi

# ---------------------------------------------------------------------------
# repo list --visibility
# ---------------------------------------------------------------------------
section "gitee repo list --visibility"

for visibility in "public" "private" "internal"; do
    if output=$("$BINARY" repo list --visibility "$visibility" --limit 5 2>&1); then
        pass "repo list --visibility $visibility 执行成功"
    else
        fail "repo list --visibility $visibility 执行失败"
    fi
done

# ---------------------------------------------------------------------------
# repo list --page --per-page
# ---------------------------------------------------------------------------
section "gitee repo list 分页参数"

if output=$("$BINARY" repo list --page 1 --per-page 10 2>&1); then
    pass "repo list --page --per-page 参数接受"
else
    fail "repo list --page --per-page 参数不被接受"
fi

# ---------------------------------------------------------------------------
# repo list --owner
# ---------------------------------------------------------------------------
section "gitee repo list --owner"

if output=$("$BINARY" repo list --owner gitee --limit 5 2>&1); then
    pass "repo list --owner 执行成功"
else
    fail "repo list --owner 执行失败"
fi

# ---------------------------------------------------------------------------
# repo create --help 参数验证
# ---------------------------------------------------------------------------
section "gitee repo create --help"

help_output=$("$BINARY" repo create --help 2>&1)
expected_args=("name" "description" "private" "auto-init" "gitignore" "license" "org")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "repo create --help 包含 $arg 参数"
    else
        fail "repo create --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# repo create Dry-run 验证 (用已有仓库名触发错误)
# ---------------------------------------------------------------------------
section "gitee repo create 参数验证"

# 缺少必填参数 --name 时应报错
if output=$("$BINARY" repo create 2>&1); then
    fail "repo create 缺少 --name 应报错，但未报错"
else
    pass "repo create 缺少 --name 正确报错"
fi

# ---------------------------------------------------------------------------
# repo view --help
# ---------------------------------------------------------------------------
section "gitee repo view --help"

if "$BINARY" repo view --help > /dev/null 2>&1; then
    pass "repo view --help 正常"
else
    fail "repo view --help 失败"
fi

# ---------------------------------------------------------------------------
# repo browse --help
# ---------------------------------------------------------------------------
section "gitee repo browse --help"

if "$BINARY" repo browse --help > /dev/null 2>&1; then
    pass "repo browse --help 正常"
else
    fail "repo browse --help 失败"
fi

# ---------------------------------------------------------------------------
# repo edit --help
# ---------------------------------------------------------------------------
section "gitee repo edit --help"

if "$BINARY" repo edit --help > /dev/null 2>&1; then
    pass "repo edit --help 正常"
else
    fail "repo edit --help 失败"
fi

# ---------------------------------------------------------------------------
# repo delete --help (应强调安全性)
# ---------------------------------------------------------------------------
section "gitee repo delete --help"

help_output=$("$BINARY" repo delete --help 2>&1)
if echo "$help_output" | grep -q "yes\|confirm\|删除"; then
    pass "repo delete --help 包含安全确认提示"
else
    fail "repo delete --help 缺少安全确认提示"
fi

# ---------------------------------------------------------------------------
# repo fork --help
# ---------------------------------------------------------------------------
section "gitee repo fork --help"

if "$BINARY" repo fork --help > /dev/null 2>&1; then
    pass "repo fork --help 正常"
else
    fail "repo fork --help 失败"
fi

# ---------------------------------------------------------------------------
# 全局 flag 验证
# ---------------------------------------------------------------------------
section "全局 Flag 验证"

# --api-url
if output=$("$BINARY" repo list --api-url "https://api.gitee.com/api/v5" --limit 1 2>&1); then
    pass "--api-url flag 接受"
else
    fail "--api-url flag 不被接受"
fi

# --output
for fmt in "json" "table" "csv"; do
    if output=$("$BINARY" repo list --output "$fmt" --limit 1 2>&1); then
        pass "--output $fmt 有效"
    else
        fail "--output $fmt 无效"
    fi
done

# --verbose
if "$BINARY" repo list --verbose --limit 1 2>&1 | grep -qi "http\|request\|debug"; then
    pass "--verbose 输出调试信息"
else
    skip "--verbose 未见明显调试输出（可能正常）"
fi

# --no-color
if "$BINARY" repo list --no-color --limit 1 > /dev/null 2>&1; then
    pass "--no-color 参数接受"
else
    fail "--no-color 参数不被接受"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ repo 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ repo 模块测试有失败项${NC}"
    exit 1
fi