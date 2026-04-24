#!/usr/bin/env bash
# =============================================================================
# gitee completion 模块测试
# 测试 Shell 自动补全脚本生成
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
# completion --help
# ---------------------------------------------------------------------------
section "gitee completion --help"
if "$BINARY" completion --help > /dev/null 2>&1; then
    pass "completion --help 正常显示"
else
    fail "completion --help 失败"
fi

# ---------------------------------------------------------------------------
# 子命令列表
# ---------------------------------------------------------------------------
section "gitee completion 子命令验证"
expected_shells=("bash" "zsh" "fish" "powershell")
for shell in "${expected_shells[@]}"; do
    if "$BINARY" completion "$shell" --help > /dev/null 2>&1; then
        pass "completion $shell 子命令存在"
    else
        fail "completion $shell 子命令不存在"
    fi
done

# ---------------------------------------------------------------------------
# bash 补全脚本内容验证
# ---------------------------------------------------------------------------
section "gitee completion bash 脚本内容"
output=$("$BINARY" completion bash 2>&1)
if [[ -n "$output" ]]; then
    if echo "$output" | grep -qE "__gitee_|_gitee|complete.*gitee|compgen"; then
        pass "completion bash 输出包含 Bash 补全函数"
    elif echo "$output" | grep -qE "^#|bash|complete"; then
        pass "completion bash 输出包含补全脚本特征"
    else
        fail "completion bash 输出无法识别为 Bash 补全脚本: ${output:0:100}"
    fi
else
    fail "completion bash 输出为空"
fi

# ---------------------------------------------------------------------------
# zsh 补全脚本内容验证
# ---------------------------------------------------------------------------
section "gitee completion zsh 脚本内容"
output=$("$BINARY" completion zsh 2>&1)
if [[ -n "$output" ]]; then
    if echo "$output" | grep -qE "#compdef|_gitee|compdef"; then
        pass "completion zsh 输出包含 Zsh 补全特征（#compdef / _gitee）"
    elif echo "$output" | grep -qE "^#|zsh|compadd"; then
        pass "completion zsh 输出包含补全脚本特征"
    else
        fail "completion zsh 输出无法识别为 Zsh 补全脚本: ${output:0:100}"
    fi
else
    fail "completion zsh 输出为空"
fi

# ---------------------------------------------------------------------------
# fish 补全脚本内容验证
# ---------------------------------------------------------------------------
section "gitee completion fish 脚本内容"
output=$("$BINARY" completion fish 2>&1)
if [[ -n "$output" ]]; then
    if echo "$output" | grep -qE "complete.*-c gitee|complete -c"; then
        pass "completion fish 输出包含 Fish complete 语法"
    elif echo "$output" | grep -qE "fish|complete"; then
        pass "completion fish 输出包含 Fish 补全脚本特征"
    else
        fail "completion fish 输出无法识别为 Fish 补全脚本: ${output:0:100}"
    fi
else
    fail "completion fish 输出为空"
fi

# ---------------------------------------------------------------------------
# powershell 补全脚本内容验证
# ---------------------------------------------------------------------------
section "gitee completion powershell 脚本内容"
output=$("$BINARY" completion powershell 2>&1)
if [[ -n "$output" ]]; then
    if echo "$output" | grep -qE "Register-ArgumentCompleter|CompletionResult|pwsh|PowerShell"; then
        pass "completion powershell 输出包含 PowerShell 补全特征"
    elif echo "$output" | grep -qE "function|param|\\\$"; then
        pass "completion powershell 输出包含 PowerShell 语法特征"
    else
        fail "completion powershell 输出无法识别: ${output:0:100}"
    fi
else
    fail "completion powershell 输出为空"
fi

# ---------------------------------------------------------------------------
# 补全脚本长度合理性检查
# ---------------------------------------------------------------------------
section "补全脚本长度检查（应 > 100 字节）"
for shell in "bash" "zsh" "fish" "powershell"; do
    output=$("$BINARY" completion "$shell" 2>&1)
    length=${#output}
    if [[ $length -gt 100 ]]; then
        pass "completion $shell 脚本长度合理: ${length} 字节"
    else
        fail "completion $shell 脚本过短（${length} 字节），可能未正确生成"
    fi
done

# ---------------------------------------------------------------------------
# bash 补全脚本包含核心命令关键词
# ---------------------------------------------------------------------------
section "bash 补全脚本包含核心命令"
bash_output=$("$BINARY" completion bash 2>&1)
core_commands=("auth" "repo" "issue" "pr" "gist" "org" "search" "api" "config" "completion")
for cmd in "${core_commands[@]}"; do
    if echo "$bash_output" | grep -q "$cmd"; then
        pass "bash 补全脚本包含命令: ${cmd}"
    else
        skip "bash 补全脚本未显式包含: ${cmd} (可能通过动态反射生成)"
    fi
done

# ---------------------------------------------------------------------------
# 安装测试（验证脚本可写入文件且可被 source）
# ---------------------------------------------------------------------------
section "bash 补全脚本可写入文件"
TMP_FILE=$(mktemp /tmp/gitee-completion-test-XXXX.bash)
trap 'rm -f "$TMP_FILE"' EXIT

if "$BINARY" completion bash > "$TMP_FILE" 2>&1; then
    if [[ -s "$TMP_FILE" ]]; then
        pass "completion bash 脚本成功写入文件: $TMP_FILE"
        # 验证是否可被 bash source（语法检查）
        if bash -n "$TMP_FILE" 2>/dev/null; then
            pass "completion bash 脚本语法正确（bash -n 检查通过）"
        else
            fail "completion bash 脚本语法错误（bash -n 检查失败）"
        fi
    else
        fail "completion bash 写入文件后为空"
    fi
else
    fail "completion bash 重定向写入失败"
fi

# ---------------------------------------------------------------------------
# zsh 补全脚本语法检查（需要 zsh）
# ---------------------------------------------------------------------------
section "zsh 补全脚本语法检查"
if command -v zsh &>/dev/null; then
    ZSH_TMP=$(mktemp /tmp/gitee-completion-test-XXXX.zsh)
    "$BINARY" completion zsh > "$ZSH_TMP" 2>/dev/null || true
    if zsh -n "$ZSH_TMP" 2>/dev/null; then
        pass "completion zsh 脚本语法正确（zsh -n 检查通过）"
    else
        fail "completion zsh 脚本语法错误"
    fi
    rm -f "$ZSH_TMP"
else
    skip "zsh 未安装，跳过语法检查"
fi

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------
echo ""
echo "────────────────────────────────────────"
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}"
echo "────────────────────────────────────────"

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ completion 模块测试全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ completion 模块测试有失败项${NC}"
    exit 1
fi
