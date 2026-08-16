#!/usr/bin/env bash
# =============================================================================
# gitee-cli 测试入口脚本
# 运行所有模块的功能测试脚本
#
# 用法:
#   ./run.sh               # 运行所有测试
#   ./run.sh auth          # 仅运行 auth 模块测试
#   ./run.sh repo issue    # 运行多个模块测试
#
# 前置条件:
#   - gitee 二进制文件已编译 (go build -o gitee ./cmd/gitee)
#   - 测试用的 Gitee 账号已登录 (gitee auth login)
# =============================================================================

set -euo pipefail

BINARY="${GITEE_CLI_BINARY:-./gitee}"
TEST_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$TEST_DIR")"

# 如果相对路径找不到，尝试绝对路径
if [[ ! -f "$BINARY" ]]; then
    if [[ -f "$ROOT_DIR/gitee" ]]; then
        BINARY="$ROOT_DIR/gitee"
    fi
fi

export BINARY
export GITEE_CLI_BINARY="$BINARY"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 测试模块列表 (25 个)
MODULES=(
    "auth" "repo" "issue" "pr" "gist" "org" "search"
    "api" "config" "completion" "repo_view" "pagination"
    "gist-unstar" "repo-branches" "pr-comments"
    "label" "milestone" "notification" "release" "ssh-key"
    "user" "webhook" "alias" "extension" "status"
)

log_info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_pass()  { echo -e "${GREEN}[PASS]${NC}  $*"; }
log_fail()  { echo -e "${RED}[FAIL]${NC}  $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }

# 检查二进制文件
check_binary() {
    if [[ ! -f "$BINARY" ]]; then
        log_warn "未找到 gitee 二进制文件: $BINARY"
        log_info "正在编译..."
        cd "$ROOT_DIR"
        if command -v go >/dev/null 2>&1; then
            go build -o gitee ./cmd/gitee
            log_info "编译完成"
        else
            log_warn "go 不可用，跳过编译"
            if [[ -f "$ROOT_DIR/gitee" ]]; then
                BINARY="$ROOT_DIR/gitee"
                export BINARY
                export GITEE_CLI_BINARY="$BINARY"
            else
                log_fail "找不到 gitee 二进制文件，且 go 不可用"
                exit 1
            fi
        fi
    fi
    if [[ ! -x "$BINARY" ]]; then
        chmod +x "$BINARY"
    fi
    log_info "使用二进制: $BINARY"
    "$BINARY" --version
}

# 运行单个测试脚本
run_module() {
    local module="$1"
    local script="$TEST_DIR/${module}.sh"

    if [[ ! -f "$script" ]]; then
        log_warn "测试脚本不存在: $script，跳过"
        return 0
    fi

    echo ""
    log_info "═══════════════════════════════════════════"
    log_info "  运行 ${module} 模块测试"
    log_info "═══════════════════════════════════════════"

    if GITEE_CLI_BINARY="$BINARY" bash "$script"; then
        log_pass "${module} 模块测试通过 ✓"
    else
        local exit_code=$?
        log_fail "${module} 模块测试失败 ✗ (exit $exit_code)"
        FAILED_MODULES+=("$module")
    fi
}

# 显示帮助
show_help() {
    echo "gitee-cli 测试入口"
    echo ""
    echo "用法: $0 [模块...]"
    echo ""
    echo "可用模块: ${MODULES[*]}"
    echo "不传参数则运行全部模块"
    echo ""
    echo "环境变量:"
    echo "  GITEE_CLI_BINARY  指定二进制文件路径 (默认: ./gitee)"
    echo ""
    echo "示例:"
    echo "  $0                 # 运行所有测试"
    echo "  $0 auth repo       # 仅测试 auth 和 repo"
    echo "  GITEE_CLI_BINARY=/path/to/gitee $0 issue"
}

# 主函数
main() {
    echo "╔══════════════════════════════════════════╗"
    echo "║       gitee-cli 功能测试套件              ║"
    echo "╚══════════════════════════════════════════╝"

    check_binary

    # 解析参数
    if [[ $# -eq 0 ]]; then
        TARGET_MODULES=("${MODULES[@]}")
    else
        for arg in "$@"; do
            case "$arg" in
                -h|--help) show_help; exit 0 ;;
                *) TARGET_MODULES+=("$arg") ;;
            esac
        done
    fi

    # 检查指定模块是否存在
    for m in "${TARGET_MODULES[@]}"; do
        if [[ ! " ${MODULES[*]} " =~ " ${m} " ]]; then
            log_fail "未知模块: $m"
            echo "可用模块: ${MODULES[*]}"
            exit 1
        fi
    done

    # 运行测试
    FAILED_MODULES=()
    for module in "${TARGET_MODULES[@]}"; do
        run_module "$module"
    done

    # 汇总结果
    echo ""
    echo "═══════════════════════════════════════════"
    if [[ ${#FAILED_MODULES[@]} -eq 0 ]]; then
        log_pass "全部测试通过！"
        exit 0
    else
        log_fail "以下模块测试失败: ${FAILED_MODULES[*]}"
        exit 1
    fi
}

main "$@"
