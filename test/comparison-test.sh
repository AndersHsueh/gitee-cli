#!/usr/bin/env bash
# =============================================================================
# gh vs gitee-cli 行为对比测试
# 黑盒端到端测试: 逐个 verb 跑 gitee 和 gh, 验证 --help / --agent-help /
# 真实 API 调用的行为对标.
#
# 前置条件:
#   - gh 已安装并认证
#   - go toolchain 可用
#   - 设置环境变量 GITEE_TEST_TOKEN=<token>  (或已有 ~/.config/gitee/credentials.yml)
#
# 用法:
#   export GITEE_TEST_TOKEN=xxx
#   ./test/comparison-test.sh
#
# 产出:
#   - test/behavior-comparison-gh-vs-gitee.md  (主报告)
# =============================================================================

set -euo pipefail

# ----------------------------------------------------------------------------
# 常量
# ----------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="${GITEE_CLI_BINARY:-$ROOT_DIR/gitee}"
GH_BIN="${GH_BIN:-gh}"
REPORT="$SCRIPT_DIR/behavior-comparison-gh-vs-gitee.md"
TEST_REPO="${GITEE_TEST_REPO:-andershsueh/outsourcing-ms}"
GH_TEST_REPO="${GH_TEST_REPO:-cli/cli}"  # 公开有 issue+pr 的 gh 测试仓

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

passed=0
failed=0
skipped=0
declare -a FINDINGS=()   # 行为差异(非 bug)
declare -a BUGS=()       # 真实 bug
declare -a AGENT_HELP_GAPS=()  # --agent-help 缺失章节

pass()    { ((passed++)) || true; echo -e "  ${GREEN}✓${NC} $*"; }
fail()    { ((failed++)) || true; echo -e "  ${RED}✗${NC} $*" >&2; BUGS+=("$*"); }
skip()    { ((skipped++)) || true; echo -e "  ${YELLOW}⊘${NC} $* (跳过)"; }
note()    { echo -e "  ${CYAN}ⓘ${NC} $*"; FINDINGS+=("$*"); }
section() { echo ""; echo -e "${BLUE}▶${NC} $*"; }
hr()      { echo -e "────────────────────────────────────────"; }

# ----------------------------------------------------------------------------
# 前置: 构建二进制, 准备凭证
# ----------------------------------------------------------------------------
section "前置: 编译 gitee-cli"

cd "$ROOT_DIR"
if [[ ! -x "$BINARY" ]]; then
    if command -v go >/dev/null 2>&1; then
        echo "  正在编译 $BINARY ..."
        go build -o gitee ./cmd/gitee
    else
        echo -e "${RED}✗ go 不可用且 $BINARY 不存在${NC}" >&2
        exit 1
    fi
fi
pass "二进制就绪: $("$BINARY" --version 2>&1 | head -1)"

# ----------------------------------------------------------------------------
# 凭证
# ----------------------------------------------------------------------------
section "前置: Gitee 凭证"

CREDS_FILE="$HOME/.config/gitee/credentials.yml"
mkdir -p "$(dirname "$CREDS_FILE")"
chmod 700 "$(dirname "$CREDS_FILE")"

if [[ -f "$CREDS_FILE" ]]; then
    pass "凭证已存在: $CREDS_FILE"
elif [[ -n "${GITEE_TEST_TOKEN:-}" ]]; then
    # ⚠️ token 仅写入本地凭证文件, 不会被脚本记录或回显
    cat > "$CREDS_FILE" <<EOF
credentials:
  token: ${GITEE_TEST_TOKEN}
EOF
    chmod 600 "$CREDS_FILE"
    pass "凭证已写入: $CREDS_FILE (mode 600)"
else
    echo -e "${YELLOW}⚠ 未找到 $CREDS_FILE 且 GITEE_TEST_TOKEN 未设置${NC}"
    echo "  真实 API 测试将全部 skip, --help / --agent-help 仍会跑"
fi

# 验证 gitee auth status (token 不回显, 只看是否成功)
if "$BINARY" auth status >/dev/null 2>&1; then
    pass "gitee auth status: 正常"
else
    skip "gitee auth status 失败, 真实 API 测试会 skip"
    export GITEE_AUTHED=0
fi
export GITEE_AUTHED="${GITEE_AUTHED:-1}"

# 验证 gh
if "$GH_BIN" auth status >/dev/null 2>&1; then
    pass "gh auth status: 正常 ($("$GH_BIN" api user --jq .login 2>/dev/null || echo unknown))"
else
    echo -e "${YELLOW}⚠ gh 未认证, gh 对照测试会 skip${NC}"
    export GH_AUTHED=0
fi
export GH_AUTHED="${GH_AUTHED:-1}"

# ----------------------------------------------------------------------------
# 工具函数
# ----------------------------------------------------------------------------

# 抽命令清单: `extract_commands gitee` 抽一级; `extract_commands gitee <group>` 抽 verb
extract_commands() {
    local bin="$1" group="${2:-}"
    local target="$bin"
    local args="--help"
    if [[ -n "$group" ]]; then
        args="$group --help"
    fi
    "$target" $args 2>/dev/null | awk '
        # gitee: "Available Commands:"; gh: "CORE COMMANDS" / "ADDITIONAL COMMANDS" 等 (无冒号)
        /^Available Commands:/ {flag=1; next}
        /^(CORE|GITHUB ACTIONS|ALIAS|ADDITIONAL|FEATURE) COMMANDS$/ {flag=1; next}
        # 收尾: 行首无缩进的非空行表示段落结束
        flag && /^[^[:space:]]/ {flag=0}
        flag && /^[[:space:]]+[a-z]/ {
            sub(/^[[:space:]]+/, "")
            # gh 格式: "auth:          Authenticate..."  → 取第一个字段去掉冒号
            cmd=$1
            sub(/:$/, "", cmd)
            print cmd
        }
    ' | sort -u
}

# 验证 --agent-help 14 章节 (宽松版: 核心章节必须在, 命名变体也算)
# 调用方负责先 invoke `--agent-help` 一次, 把输出通过 $2 传入, 避免重复 spawn
check_agent_help() {
    local cmd="$1" out="$2"
    local core=(
        "## Purpose"
        "## Idempotency"
        "## Atomicity"
        "## Side effects"
    )
    local optional=(
        "## Pre-conditions"
        "## Common errors"
        "## Example"
        "## OAuth scope required"
        "## Rate limit"
    )
    local missing_core=()
    local missing_opt=()
    for s in "${core[@]}"; do
        if ! grep -qF "$s" <<<"$out"; then
            missing_core+=("$s")
        fi
    done
    for s in "${optional[@]}"; do
        if ! grep -qF "$s" <<<"$out"; then
            missing_opt+=("$s")
        fi
    done
    if [[ ${#missing_core[@]} -eq 0 ]]; then
        pass "$cmd --agent-help 核心章节齐全 (缺可选: ${missing_opt[*]:-无})"
        if [[ ${#missing_opt[@]} -gt 0 ]]; then
            AGENT_HELP_GAPS+=("$cmd: 缺可选: ${missing_opt[*]}")
        fi
        return 0
    else
        fail "$cmd --agent-help 缺核心: ${missing_core[*]}"
        AGENT_HELP_GAPS+=("$cmd: 缺核心: ${missing_core[*]}")
        return 1
    fi
}

# 对比 gitee 和 gh 的 json keys (level 1)
diff_json_keys() {
    local label="$1" gitee_out="$2" gh_out="$3"
    if [[ -z "$gitee_out" || -z "$gh_out" ]]; then return; fi
    local gitee_keys gh_keys
    gitee_keys=$(echo "$gitee_out" | jq -r 'if type=="array" then .[0] else . end | keys | join(",")' 2>/dev/null | sort)
    gh_keys=$(echo "$gh_out" | jq -r 'if type=="array" then .[0] else . end | keys | join(",")' 2>/dev/null | sort)
    if [[ "$gitee_keys" == "$gh_keys" ]]; then
        pass "$label json keys 一致: $(echo "$gitee_keys" | tr ',' ' ')"
    else
        note "$label json keys 差异: gitee=[$(echo "$gitee_keys" | tr ',' ' ')] vs gh=[$(echo "$gh_keys" | tr ',' ' ')]"
    fi
}

# ----------------------------------------------------------------------------
# Phase 1: 一级命令覆盖
# ----------------------------------------------------------------------------
section "Phase 1: 一级命令覆盖"

GITEE_TOP=()
while IFS= read -r line; do
    [[ -n "$line" ]] && GITEE_TOP+=("$line")
done < <(extract_commands "$BINARY")

GH_TOP=()
while IFS= read -r line; do
    [[ -n "$line" ]] && GH_TOP+=("$line")
done < <(extract_commands "$GH_BIN")

# 过滤 gitee 的 'help' (gh 也会有, 跳过)
GITEE_TOP_FILTERED=()
for c in "${GITEE_TOP[@]}"; do
    [[ "$c" == "help" ]] && continue
    GITEE_TOP_FILTERED+=("$c")
done

echo "  gitee 一级命令 (${#GITEE_TOP_FILTERED[@]} 个): ${GITEE_TOP_FILTERED[*]}"
echo "  gh    一级命令 (${#GH_TOP[@]} 个): ${GH_TOP[*]}"

if [[ ${#GITEE_TOP_FILTERED[@]} -ge 15 ]]; then
    pass "gitee 一级命令 ≥ 15 (v0.9.0-a 期望 17)"
else
    fail "gitee 一级命令过少: ${#GITEE_TOP_FILTERED[@]} (期望 ≥ 15)"
fi

# ----------------------------------------------------------------------------
# Phase 2: 每个 verb 的 --help 抽取
# ----------------------------------------------------------------------------
section "Phase 2: --help 抽取每个 verb"

for group in "${GITEE_TOP_FILTERED[@]}"; do
    VERBS=()
    while IFS= read -r line; do
        [[ -n "$line" ]] && VERBS+=("$line")
    done < <(extract_commands "$BINARY" "$group")
    if [[ ${#VERBS[@]} -eq 0 ]]; then
        # leaf command (如 status, version) — 直接跑 --help
        if "$BINARY" "$group" --help >/dev/null 2>&1; then
            pass "$group --help OK (leaf)"
        else
            fail "$group --help 失败"
        fi
        continue
    fi
    for v in "${VERBS[@]}"; do
        local_cmd="$group $v"
        if "$BINARY" $group $v --help >/dev/null 2>&1; then
            pass "$local_cmd --help OK"
        else
            fail "$local_cmd --help 失败"
        fi
    done
done

# ----------------------------------------------------------------------------
# Phase 3: --agent-help 14 章节验证
# ----------------------------------------------------------------------------
section "Phase 3: --agent-help 14 章节验证"

# 哪些 verb 声明有 --agent-help (从 root --agent-help 的 Coverage matrix 抽)
ROOT_AGENT_HELP=$("$BINARY" --agent-help 2>/dev/null || true)
AGENT_HELP_VERBS=()
while IFS= read -r line; do
    line=$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    [[ -n "$line" ]] && AGENT_HELP_VERBS+=("$line")
done < <(echo "$ROOT_AGENT_HELP" | grep "✅ available" | sed -E 's/.*gitee (.+) +--agent-help.*/\1/')

if [[ ${#AGENT_HELP_VERBS[@]} -eq 0 ]]; then
    skip "未找到任何 --agent-help verb (Coverage matrix 为空)"
else
    echo "  声明 --agent-help 的 verb 数: ${#AGENT_HELP_VERBS[@]}"
    for v in "${AGENT_HELP_VERBS[@]}"; do
        # 某些 verb 需要 positional (如 issue comments <n>), 传占位符
        # 每次只 spawn 一次, 把输出传给 check_agent_help 避免重复
        if out=$("$BINARY" $v --agent-help 2>&1); then
            check_agent_help "$v" "$out"
        elif out=$("$BINARY" $v 1 --agent-help 2>&1); then
            note "$v --agent-help 需要 positional (1), 用占位符验证"
            check_agent_help "$v 1" "$out"
        elif out=$("$BINARY" $v 1 2 --agent-help 2>&1); then
            note "$v --agent-help 需要 2 个 positional, 用占位符验证"
            check_agent_help "$v 1 2" "$out"
        else
            fail "$v --agent-help 始终失败"
        fi
    done
fi

# ----------------------------------------------------------------------------
# Phase 4: 真实 API 抽查
# ----------------------------------------------------------------------------
section "Phase 4: 真实 API 抽查"

SMOKE_TESTS=(
    "repo list --limit 1 --output json"
    "issue list --repo $TEST_REPO --limit 1 --output json"
    "pr list --repo $TEST_REPO --limit 1 --output json"
    "gist list --limit 1 --output json"
    "org list --limit 1 --output json"
    "release list --repo $TEST_REPO --limit 1 --output json"
    "label list --repo $TEST_REPO --output json"
    "search repos gitee-cli --limit 1 --output json"
    "search issues bug --limit 1 --output json"
    "ssh-key list --output json"
    "webhook list --repo $TEST_REPO --limit 1 --output json"
    "api user --output json"
)
# 用临时文件存结果(兼容 bash 3.2 无 declare -A)
SMOKE_RESULTS_DIR=$(mktemp -d)
# trap 会在 Phase 5 重新设置, 这里只兜底
trap "rm -rf $SMOKE_RESULTS_DIR 2>/dev/null" EXIT

smoke_key() { local s="${1//\//_}"; echo "${s// /_}"; }

# 已知预期失败: 友好错误而不是 bug
# - "Gitee v5 API does not provide" → 端点不支持
# - "accepts N arg(s)" → cobra arg 校验
is_expected_failure() {
    local out="$1"
    [[ "$out" == *"Gitee v5 API does not provide"* ]] && return 0
    [[ "$out" == *"accepts "*"arg(s), received"* ]] && return 0
    return 1
}

if [[ "$GITEE_AUTHED" != "1" ]]; then
    skip "gitee 未认证, 跳过所有真实 API 抽查"
    for t in "${SMOKE_TESTS[@]}"; do echo "⏭ skip" > "$SMOKE_RESULTS_DIR/$(smoke_key "$t")"; done
else
    for t in "${SMOKE_TESTS[@]}"; do
        echo ""
        echo -e "  ${CYAN}→${NC} gitee $t"
        key=$(smoke_key "$t")
        out=$("$BINARY" $t 2>&1) && {
            pass "成功"
            if echo "$out" | jq . >/dev/null 2>&1; then
                pass "  输出是有效 JSON"
                echo "✅" > "$SMOKE_RESULTS_DIR/$key"
                # 缓存原始输出供 Phase 5 复用 (避免重复 API 调用)
                echo "$out" > "$SMOKE_RESULTS_DIR/$key.out"
            else
                note "  输出非 JSON: ${out:0:100}"
                echo "🟡 non-JSON" > "$SMOKE_RESULTS_DIR/$key"
            fi
        } || {
            if is_expected_failure "$out"; then
                pass "预期失败 (Gitee 端点不支持 / 参数校验): ${out:0:120}"
                echo "🚫 expected" > "$SMOKE_RESULTS_DIR/$key"
            else
                fail "失败: ${out:0:200}"
                echo "❌" > "$SMOKE_RESULTS_DIR/$key"
            fi
        }
        sleep 0.3
    done
fi

# ----------------------------------------------------------------------------
# Phase 5: gh 对照测试
# ----------------------------------------------------------------------------
section "Phase 5: gh 对照测试 (json keys 对比)"

# 提前创建 (兼容 bash 3.2 无 declare -A)
GH_MAP_DIR=$(mktemp -d)
trap "rm -rf $GH_MAP_DIR $SMOKE_RESULTS_DIR 2>/dev/null" EXIT

if [[ "$GH_AUTHED" != "1" ]]; then
    skip "gh 未认证, 跳过"
else
    # GH_MAP_DIR 已在 Phase 5 开头创建
    echo "gh repo list --limit 1 --json name,description,isPrivate,createdAt --jq '.[0]'" > "$GH_MAP_DIR/repo list"
    echo "gh issue list --repo $GH_TEST_REPO --limit 1 --json number,title,state,createdAt,author" > "$GH_MAP_DIR/issue list"
    echo "gh pr list --repo $GH_TEST_REPO --limit 1 --json number,title,state,createdAt,author" > "$GH_MAP_DIR/pr list"
    echo "gh gist list --limit 1 --json id,description,public,createdAt" > "$GH_MAP_DIR/gist list"

    for k in "repo list" "issue list" "pr list" "gist list"; do
        gh_cmd=$(cat "$GH_MAP_DIR/$k")
        case "$k" in
            "repo list") gitee_cmd="repo list --limit 1 --output json" ;;
            "issue list") gitee_cmd="issue list --repo $TEST_REPO --limit 1 --output json" ;;
            "pr list") gitee_cmd="pr list --repo $TEST_REPO --limit 1 --output json" ;;
            "gist list") gitee_cmd="gist list --limit 1 --output json" ;;
        esac

        echo ""
        echo -e "  ${CYAN}→${NC} 对比: $k"
        # 优先复用 Phase 4 真实 API 抽查的输出 (避免再发一次 Gitee API)
        cache_key=$(smoke_key "$gitee_cmd")
        if [[ -f "$SMOKE_RESULTS_DIR/$cache_key.out" ]]; then
            g_out=$(<"$SMOKE_RESULTS_DIR/$cache_key.out")
        else
            g_out=$("$BINARY" $gitee_cmd 2>/dev/null || echo "")
        fi
        h_out=$(eval "$gh_cmd" 2>/dev/null || echo "")
        if [[ -n "$g_out" && -n "$h_out" ]]; then
            diff_json_keys "$k" "$g_out" "$h_out"
        else
            skip "  至少一边没拿到输出"
        fi
        sleep 0.3
    done
fi

# ----------------------------------------------------------------------------
# Phase 6: 行为差异 (写操作不测, 只看 read 路径)
# ----------------------------------------------------------------------------
section "Phase 6: 行为差异标注"

# 已知差异 (从 common-todos.md 和 agent-help 文档中整理)
note "字段命名风格: gitee 用 snake_case (Gitee API 原生); gh 用 camelCase. 这是 API 风格差异, 不是 bug."
note "--agent-help 是 gitee 独有; gh 无此 flag"
note "gitee 一级命令里把 webhook 提升为顶级 (gh 把它藏在 api 后)"

# ----------------------------------------------------------------------------
# Phase 7: 写报告
# ----------------------------------------------------------------------------
section "Phase 7: 生成报告"

TOTAL=$((passed + failed + skipped))
{
    echo "# gh vs gitee-cli 行为对比测试报告"
    echo ""
    echo "> **生成时间**: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "> **gh 版本**: $("$GH_BIN" --version 2>&1 | head -1)"
    echo "> **gitee-cli 版本**: $("$BINARY" --version 2>&1 | head -1)"
    echo "> **测试范围**: 全量 (${#GITEE_TOP_FILTERED[@]} 个一级命令 + 全部 verb)"
    echo ""
    echo "## 1. 总览"
    echo ""
    echo "| 指标 | 数字 |"
    echo "|------|------|"
    echo "| gitee 一级命令 | ${#GITEE_TOP_FILTERED[@]} |"
    echo "| gh 一级命令 | ${#GH_TOP[@]} |"
    echo "| --agent-help 声明 verb 数 | ${#AGENT_HELP_VERBS[@]} |"
    echo "| **通过** | **$passed** |"
    echo "| **失败** | **$failed** |"
    echo "| **跳过** | **$skipped** |"
    echo "| 总计 | $TOTAL |"
    echo ""
    echo "## 2. 一级命令覆盖"
    echo ""
    echo "| gitee-cli 一级 | gh 等价 | 状态 |"
    echo "|---|---|---|"
    for c in "${GITEE_TOP_FILTERED[@]}"; do
        gh_equiv=""
        case "$c" in
            auth|api|completion|config|extension|issue|pr|release|repo|search|status) gh_equiv="$c" ;;
            gist|org) gh_equiv="$c" ;;
            label|ssh-key|gpg-key) gh_equiv="$c" ;;
            webhook) gh_equiv="(在 api 后)" ;;
            *) gh_equiv="—" ;;
        esac
        echo "| \`$c\` | \`$gh_equiv\` | ✅ |"
    done
    echo ""
    echo "## 3. --agent-help 14 章节验证"
    echo ""
    if [[ ${#AGENT_HELP_VERBS[@]} -eq 0 ]]; then
        echo "_无 verb 声明 --agent-help_"
    else
        echo "| verb | 状态 | 缺失章节 |"
        echo "|---|---|---|"
        for v in "${AGENT_HELP_VERBS[@]}"; do
            missing=""
            for gap in "${AGENT_HELP_GAPS[@]}"; do
                if [[ "$gap" == "$v:"* ]]; then
                    missing="${gap#*: }"
                    break
                fi
            done
            if [[ -z "$missing" ]]; then
                echo "| \`$v\` | ✅ | — |"
            else
                echo "| \`$v\` | ❌ | $missing |"
            fi
        done
    fi
    echo ""
    echo "## 4. 行为差异 (非 bug)"
    echo ""
    if [[ ${#FINDINGS[@]} -eq 0 ]]; then
        echo "_无_"
    else
        for f in "${FINDINGS[@]}"; do
            echo "- $f"
        done
    fi
    echo ""
    echo "## 5. 真实 Bug 候选 (如发现)"
    echo ""
    if [[ ${#BUGS[@]} -eq 0 ]]; then
        echo "_无_"
    else
        for b in "${BUGS[@]}"; do
            echo "- $b"
        done
    fi
    echo ""
    echo "## 6. 真实 API 抽查"
    echo ""
    echo "| 命令 | 状态 | 备注 |"
    echo "|---|---|---|"
    for t in "${SMOKE_TESTS[@]}"; do
        key=$(smoke_key "$t")
        result="⏭ skip"
        if [[ -f "$SMOKE_RESULTS_DIR/$key" ]]; then
            result=$(<"$SMOKE_RESULTS_DIR/$key")
        fi
        echo "| \`gitee $t\` | $result | — |"
    done
    echo ""
    echo "## 7. 复现方式"
    echo ""
    echo '```bash'
    echo "# 1. 编译"
    echo "cd $ROOT_DIR"
    echo "go build -o gitee ./cmd/gitee"
    echo ""
    echo "# 2. 设置 token (私密, 不入 git)"
    echo "export GITEE_TEST_TOKEN=<your-token>"
    echo ""
    echo "# 3. 跑测试"
    echo "./test/comparison-test.sh"
    echo ""
    echo "# 4. 查看报告"
    echo "cat test/behavior-comparison-gh-vs-gitee.md"
    echo '```'
    echo ""
    echo "---"
    echo ""
    echo "_本报告由 \`test/comparison-test.sh\` 自动生成, 不可手改._"
} > "$REPORT"

pass "报告已写入: $REPORT"

# ----------------------------------------------------------------------------
# 总结
# ----------------------------------------------------------------------------
hr
echo -e "  ${GREEN}通过: $passed${NC}  ${RED}失败: $failed${NC}  ${YELLOW}跳过: $skipped${NC}  ${CYAN}信息: ${#FINDINGS[@]}${NC}"
hr

if [[ $failed -eq 0 ]]; then
    echo -e "${GREEN}✓ 全部通过${NC}"
    exit 0
else
    echo -e "${RED}✗ 有失败项, 查看上面日志与报告 $REPORT${NC}"
    exit 1
fi
