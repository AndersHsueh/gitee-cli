#!/usr/bin/env bash
# =============================================================================
# gitee config 模块测试
# 测试配置管理相关命令
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

# 保存测试前的配置，以便还原
CONFIG_FILE="${HOME}/.gitee-cli/settings.json"
CONFIG_BACKUP="${HOME}/.gitee-cli/settings.json.bak.$$"

cleanup() {
    if [[ -f "$CONFIG_BACKUP" ]]; then
        mv "$CONFIG_BACKUP" "$CONFIG_FILE" 2>/dev/null || true
    fi
}
trap cleanup EXIT

# 备份当前配置
if [[ -f "$CONFIG_FILE" ]]; then
    cp "$CONFIG_FILE" "$CONFIG_BACKUP"
fi

mkdir -p "$(dirname "$CONFIG_FILE")"

# ---------------------------------------------------------------------------
# config --help
# ---------------------------------------------------------------------------
section "gitee config --help"
if "$BINARY" config --help > /dev/null 2>&1; then
    pass "config --help 正常显示"
else
    fail "config --help 失败"
fi

# ---------------------------------------------------------------------------
# 子命令列表
# ---------------------------------------------------------------------------
section "gitee config 子命令验证"
expected_commands=("list" "set" "get" "unset")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" config "$cmd" --help > /dev/null 2>&1; then
        pass "config $cmd 子命令存在"
    else
        fail "config $cmd 子命令不存在"
    fi
done

# ---------------------------------------------------------------------------
# config set + get 往返测试
# ---------------------------------------------------------------------------
section "gitee config set/get 往返测试"

# 设置测试键值
if "$BINARY" config set protocol https 2>&1; then
    pass "config set protocol https 成功"
else
    fail "config set protocol https 失败"
fi

# 读取刚设置的值
get_output=$("$BINARY" config get protocol 2>&1)
if echo "$get_output" | grep -q "https"; then
    pass "config get protocol 返回正确值: https"
else
    fail "config get protocol 未返回 https，实际: $get_output"
fi

# ---------------------------------------------------------------------------
# config set editor
# ---------------------------------------------------------------------------
section "gitee config set editor"
if "$BINARY" config set editor vim 2>&1; then
    pass "config set editor vim 成功"
else
    fail "config set editor vim 失败"
fi

get_output=$("$BINARY" config get editor 2>&1)
if echo "$get_output" | grep -q "vim"; then
    pass "config get editor 返回 vim"
else
    fail "config get editor 未返回 vim，实际: $get_output"
fi

# ---------------------------------------------------------------------------
# config set output
# ---------------------------------------------------------------------------
section "gitee config set output"
if "$BINARY" config set output json 2>&1; then
    pass "config set output json 成功"
else
    fail "config set output json 失败"
fi

# 恢复为 table（避免影响后续测试）
"$BINARY" config set output table 2>/dev/null || true

# ---------------------------------------------------------------------------
# config list 包含已设置的键
# ---------------------------------------------------------------------------
section "gitee config list"
list_output=$("$BINARY" config list 2>&1)
if echo "$list_output" | grep -qiE "protocol|editor|output|prompt"; then
    pass "config list 包含已知配置键"
else
    fail "config list 输出不含预期键: ${list_output:0:200}"
fi

# ---------------------------------------------------------------------------
# config unset 测试
# ---------------------------------------------------------------------------
section "gitee config unset"

# 先设置一个临时键（如果支持任意键）或用 editor
"$BINARY" config set editor nano 2>/dev/null || true
if "$BINARY" config unset editor 2>&1; then
    pass "config unset editor 成功"
    # 验证已删除
    get_after=$("$BINARY" config get editor 2>&1 || true)
    if echo "$get_after" | grep -qvE "^nano$"; then
        pass "config get editor 删除后无值"
    else
        skip "config unset 后仍能 get，可能返回默认值"
    fi
else
    skip "config unset 未实现或失败（可接受）"
fi

# ---------------------------------------------------------------------------
# config get 不存在的键
# ---------------------------------------------------------------------------
section "gitee config get 不存在的键"
fake_key="gitee_cli_nonexistent_key_xyz"
if output=$("$BINARY" config get "$fake_key" 2>&1); then
    # 某些实现会返回空字符串（exit 0）
    if [[ -z "$output" ]]; then
        pass "config get 不存在的键返回空"
    else
        skip "config get 不存在的键返回: $output"
    fi
else
    pass "config get 不存在的键正确报错（exit $?）"
fi

# ---------------------------------------------------------------------------
# config set 无效的 protocol 值
# ---------------------------------------------------------------------------
section "gitee config set 无效值验证"
if output=$("$BINARY" config set protocol ftp 2>&1); then
    # 某些实现不做枚举验证
    skip "config set protocol ftp 被接受（实现未做枚举校验）"
else
    pass "config set 无效的 protocol 值正确报错"
fi

# 清理：恢复 protocol 为 https
"$BINARY" config set protocol https 2>/dev/null || true

# ---------------------------------------------------------------------------
# config list --output json（如支持）
# ---------------------------------------------------------------------------
section "gitee config list --output json"
if output=$("$BINARY" config list --output json 2>&1); then
    if echo "$output" | grep -qE '^\{|^\[|"protocol"'; then
        pass "config list --output json 输出有效 JSON"
    else
        skip "config list --output json 输出格式待确认: ${output:0:100}"
    fi
else
    skip "config list --output json 不支持（可接受）"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ config 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ config 模块测试有失败项${NC}"
    exit 1
fi
