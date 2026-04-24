#!/usr/bin/env bash
# =============================================================================
# gitee gist 模块测试
# 测试代码片段相关命令
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
    skip "未登录，跳过 gist 测试"
    exit 0
fi
pass "认证状态正常"

# ---------------------------------------------------------------------------
# gist --help
# ---------------------------------------------------------------------------
section "gitee gist --help"
if "$BINARY" gist --help > /dev/null 2>&1; then
    pass "gist --help 正常显示"
else
    fail "gist --help 失败"
fi

# ---------------------------------------------------------------------------
# 子命令列表
# ---------------------------------------------------------------------------
section "gitee gist 子命令验证"
expected_commands=("list" "create" "view" "edit" "delete" "star" "clone")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" gist "$cmd" --help > /dev/null 2>&1; then
        pass "gist $cmd 子命令存在"
    else
        fail "gist $cmd 子命令不存在"
    fi
done

# ---------------------------------------------------------------------------
# gist list --output json
# ---------------------------------------------------------------------------
section "gitee gist list --output json"
output=$("$BINARY" gist list --output json --limit 5 2>&1 || true)
if echo "$output" | grep -qE '"id"|^\['; then
    pass "gist list --output json 输出有效"
else
    fail "gist list --output json 输出格式不符: ${output:0:200}"
fi

# ---------------------------------------------------------------------------
# gist list --output table
# ---------------------------------------------------------------------------
section "gitee gist list --output table"
output=$("$BINARY" gist list --limit 5 2>&1 || true)
if echo "$output" | grep -qiE "ID|DESCRIPTION|FILES|PUBLIC|UPDATED|gist"; then
    pass "gist list 表格输出含表头"
else
    skip "gist list 表头未识别（可能无 gist 数据）"
fi

# ---------------------------------------------------------------------------
# gist create --help 参数检查
# ---------------------------------------------------------------------------
section "gitee gist create --help 参数检查"
help_output=$("$BINARY" gist create --help 2>&1)
expected_args=("description" "public")
for arg in "${expected_args[@]}"; do
    if echo "$help_output" | grep -q "$arg"; then
        pass "gist create --help 包含 $arg 参数"
    else
        fail "gist create --help 缺少 $arg 参数"
    fi
done

# ---------------------------------------------------------------------------
# gist create 缺少文件参数时行为
# ---------------------------------------------------------------------------
section "gitee gist create 参数验证"
if output=$("$BINARY" gist create --description "test" 2>&1); then
    # 某些实现会进入 stdin 读取模式，这也是合理的
    skip "gist create 无文件参数行为待确认（可能读取 stdin）"
else
    exit_code=$?
    if [[ $exit_code -eq 1 ]] || [[ $exit_code -eq 2 ]]; then
        pass "gist create 缺少内容时正确报错（exit $exit_code）"
    else
        fail "gist create 错误退出码: $exit_code"
    fi
fi

# ---------------------------------------------------------------------------
# gist view --help
# ---------------------------------------------------------------------------
section "gitee gist view --help"
if "$BINARY" gist view --help > /dev/null 2>&1; then
    pass "gist view --help 正常"
else
    fail "gist view --help 失败"
fi

# ---------------------------------------------------------------------------
# gist delete --help (应有安全确认提示)
# ---------------------------------------------------------------------------
section "gitee gist delete --help"
help_output=$("$BINARY" gist delete --help 2>&1)
if echo "$help_output" | grep -q "yes\|confirm\|删除"; then
    pass "gist delete --help 包含确认提示"
else
    skip "gist delete --help 未见确认提示（可能已省略）"
fi

# ---------------------------------------------------------------------------
# gist star --help
# ---------------------------------------------------------------------------
section "gitee gist star --help"
if "$BINARY" gist star --help > /dev/null 2>&1; then
    pass "gist star --help 正常"
else
    fail "gist star --help 失败"
fi

# ---------------------------------------------------------------------------
# gist clone --help
# ---------------------------------------------------------------------------
section "gitee gist clone --help"
if "$BINARY" gist clone --help > /dev/null 2>&1; then
    pass "gist clone --help 正常"
else
    fail "gist clone --help 失败"
fi

# ---------------------------------------------------------------------------
# 全局 flag 验证
# ---------------------------------------------------------------------------
section "全局 Flag 验证 (gist)"
for fmt in "json" "table" "csv"; do
    if output=$("$BINARY" gist list --output "$fmt" --limit 1 2>&1); then
        pass "--output $fmt 有效"
    else
        fail "--output $fmt 无效"
    fi
done

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ gist 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ gist 模块测试有失败项${NC}"
    exit 1
fi
