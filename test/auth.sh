#!/usr/bin/env bash
# =============================================================================
# gitee auth 模块测试
# 测试认证相关命令
#
# 依赖: gitee auth login (需交互式授权)
# =============================================================================

set -euo pipefail

BINARY="${GITEE_CLI_BINARY:-./gitee}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

passed=0
failed=0

pass() { ((passed++)); echo -e "  ${GREEN}✓${NC} $*"; }
fail() { ((failed++)); echo -e "  ${RED}✗${NC} $*" >&2; }

section() {
    echo ""
    echo -e "${BLUE}▶${NC} $*"
}

# Save real settings so tests can restore them
SETTINGS_FILE="$HOME/.gitee-cli/settings.json"
SETTINGS_BACKUP="$HOME/.gitee-cli/settings.json.bak"
if [[ -f "$SETTINGS_FILE" ]]; then
    cp "$SETTINGS_FILE" "$SETTINGS_BACKUP"
fi
restore_settings() {
    if [[ -f "$SETTINGS_BACKUP" ]]; then
        cp "$SETTINGS_BACKUP" "$SETTINGS_FILE"
        rm -f "$SETTINGS_BACKUP"
    fi
}
trap restore_settings EXIT

# ---------------------------------------------------------------------------
# 测试 auth status (未登录)
# ---------------------------------------------------------------------------
section "gitee auth status (未登录场景)"

# 清空凭证以模拟未登录
rm -f ~/.gitee-cli/settings.json 2>/dev/null || true

if "$BINARY" auth status 2>&1; then
    # 期望: 如果有 token 则显示用户信息，否则提示登录
    pass "auth status 执行成功"
else
    # 期望: 至少不应该 panic，未登录应给出清晰错误
    result=$?
    if [[ $result -eq 1 ]] && [[ $("$BINARY" auth status 2>&1) == *"not authenticated"* ]] || [[ $("$BINARY" auth status 2>&1) == *"未登录"* ]]; then
        pass "auth status 正确提示未登录"
    else
        fail "auth status 错误退出码或错误信息"
    fi
fi

# ---------------------------------------------------------------------------
# 测试 auth status --json
# ---------------------------------------------------------------------------
section "gitee auth status --json"

output=$("$BINARY" auth status --output json 2>&1 || true)
if echo "$output" | grep -q '"authenticated"'; then
    pass "auth status --output json 输出正确"
else
    # 无论是否登录，JSON 输出格式应一致
    if echo "$output" | grep -q "authenticated"; then
        pass "auth status JSON 输出格式正确"
    else
        fail "auth status --output json 输出格式不符: $output"
    fi
fi

# ---------------------------------------------------------------------------
# 测试 --help
# ---------------------------------------------------------------------------
section "gitee auth --help"

if "$BINARY" auth --help > /dev/null 2>&1; then
    pass "auth --help 正常显示"
else
    fail "auth --help 失败"
fi

# ---------------------------------------------------------------------------
# 测试子命令列表
# ---------------------------------------------------------------------------
section "gitee auth 子命令验证"

expected_commands=("login" "logout" "status" "refresh" "token")
for cmd in "${expected_commands[@]}"; do
    if "$BINARY" auth "$cmd" --help > /dev/null 2>&1; then
        pass "auth $cmd 子命令存在"
    else
        fail "auth $cmd 子命令不存在"
    fi
done

# ---------------------------------------------------------------------------
# 测试 login --help 输出内容
# ---------------------------------------------------------------------------
section "gitee auth login --help 参数检查"

help_output=$("$BINARY" auth login --help 2>&1)
if echo "$help_output" | grep -q "scopes"; then
    pass "auth login --help 显示 scopes 参数"
else
    fail "auth login --help 未显示 scopes 参数"
fi

if echo "$help_output" | grep -q "token"; then
    pass "auth login --help 显示 token 参数"
else
    fail "auth login --help 未显示 token 参数"
fi

# ---------------------------------------------------------------------------
# 测试 logout --yes (无需确认)
# ---------------------------------------------------------------------------
section "gitee auth logout --yes"

# 先创建一个空凭证文件避免目录不存在的问题
mkdir -p ~/.gitee-cli
echo '{}' > ~/.gitee-cli/settings.json

if "$BINARY" auth logout --yes 2>&1; then
    pass "auth logout --yes 执行成功（凭证已被清除）"
else
    fail "auth logout --yes 执行失败"
fi

# ---------------------------------------------------------------------------
# 测试 token 子命令
# ---------------------------------------------------------------------------
section "gitee auth token"

# Restore real settings so token command can succeed
restore_settings
if [[ -f "$SETTINGS_BACKUP" ]]; then
    cp "$SETTINGS_BACKUP" "$SETTINGS_FILE"
fi

if output=$("$BINARY" auth token 2>&1); then
    if [[ -n "$output" ]]; then
        pass "auth token 输出有效"
    else
        fail "auth token 输出为空"
    fi
else
    # When not authenticated, a clear error message is also acceptable
    if echo "$output" | grep -qiE "not authenticated|未登录"; then
        pass "auth token 正确提示未登录"
    else
        fail "auth token 命令执行失败: $output"
    fi
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ auth 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ auth 模块测试有失败项${NC}"
    exit 1
fi