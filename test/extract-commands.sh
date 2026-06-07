#!/usr/bin/env bash
# =============================================================================
# extract-commands.sh
# 提取 gh 和 gitee-cli 各自的子命令清单,生成 JSON 给 feature-comparison 文档使用
#
# 用法:
#   ./extract-commands.sh                    # 提取到 stdout
#   ./extract-commands.sh --json             # 输出 JSON 格式
#   ./extract-commands.sh --md               # 输出 markdown 表格
#   ./extract-commands.sh --raw-gh > gh.txt  # 只抽 gh
#
# 前置条件:
#   - gh 已安装
#   - ./gitee 已构建(go build -o gitee ./cmd/gitee)
# =============================================================================

set -euo pipefail

OUTPUT_FORMAT="text"
ONLY=""
BINARY_GH="${GH_BIN:-gh}"
BINARY_GITEE="${GITEE_BIN:-./gitee}"

# 解析参数
while [[ $# -gt 0 ]]; do
    case "$1" in
        --json) OUTPUT_FORMAT="json"; shift ;;
        --md) OUTPUT_FORMAT="md"; shift ;;
        --raw-gh) ONLY="gh"; shift ;;
        --raw-gitee) ONLY="gitee"; shift ;;
        -h|--help)
            sed -n '2,15p' "$0"
            exit 0
            ;;
        *) echo "Unknown flag: $1" >&2; exit 1 ;;
    esac
done

# gh 一级命令(从 gh --help 抓)
GH_TOP_LEVEL=(
    auth browse codespace gist issue org pr project release repo
    cache run workflow alias api attestation completion config extension
    gpg-key label preview ruleset search secret ssh-key status variable
)

# gitee-cli 一级命令(从 ./gitee --help 抓)
GITEE_TOP_LEVEL=(
    api auth completion config gist gpg-key issue label org
    pr release repo search ssh-key webhook
)

extract_subcommands() {
    local bin="$1"
    local cmd="$2"
    local help_output
    help_output=$("$bin" "$cmd" --help 2>/dev/null)
    # 如果 help 没有 "AVAILABLE COMMANDS" 段,说明这个 cmd 本身就是 leaf
    if ! echo "$help_output" | grep -qiE "(AVAILABLE COMMANDS|COMMANDS:)"; then
        return 0
    fi
    # 提取 Available Commands 段(到下一个全大写 SECTION 或 FLAGS 为止)
    echo "$help_output" \
        | awk '
            # 段头(支持全大写和首字母大写两种风格)
            /^[ \t]*(AVAILABLE COMMANDS|COMMANDS:|General Commands|Targeted Commands|Additional Commands|Available Commands|GitHub Actions Commands|Alias Commands|Help Topics)/ { in_section=1; next }
            # 段尾: 全大写 SECTION 头 或  FLAGS 段
            in_section && /^FLAGS$/ { in_section=0 }
            in_section && /^[A-Z][A-Z ]+:/ { in_section=0 }
            in_section && /^[A-Z][A-Z ]+$/ { in_section=0 }
            in_section && /^[ \t]*$/ { next }
            # 段内子命令:行首 2-4 空格 + verb(小写/连字符) + (冒号或多个空格) + 大写描述
            in_section && /^[ \t]{2,4}[a-z][a-z-]*[ \t:]+[A-Z]/ {
                gsub(/^[ \t]+/, "")
                # 去掉可能的尾随冒号
                gsub(/:$/, "")
                split($0, parts, /[ \t]+/)
                verb = parts[1]
                gsub(/:$/, "", verb)
                if (verb ~ /^[a-z][a-z-]*$/) {
                    print verb
                }
            }
        ' | sort -u
}

output_text() {
    if [[ "$ONLY" != "gitee" ]]; then
        echo "=== gh 一级 + 子命令 ==="
        gh --help 2>/dev/null | head -1
        for c in "${GH_TOP_LEVEL[@]}"; do
            subs=$(extract_subcommands "$BINARY_GH" "$c")
            if [[ -n "$subs" ]]; then
                echo "gh $c:"
                echo "$subs" | sed 's/^/  /'
            fi
        done
    fi
    if [[ "$ONLY" != "gh" ]]; then
        echo ""
        echo "=== gitee-cli 一级 + 子命令 ==="
        "$BINARY_GITEE" --help 2>/dev/null | head -1
        for c in "${GITEE_TOP_LEVEL[@]}"; do
            subs=$(extract_subcommands "$BINARY_GITEE" "$c")
            if [[ -n "$subs" ]]; then
                echo "gitee $c:"
                echo "$subs" | sed 's/^/  /'
            fi
        done
    fi
}

output_json() {
    echo "{"
    if [[ "$ONLY" != "gitee" ]]; then
        echo '  "gh": {'
        first=1
        for c in "${GH_TOP_LEVEL[@]}"; do
            subs=$(extract_subcommands "$BINARY_GH" "$c")
            [[ -z "$subs" ]] && continue
            [[ $first -eq 0 ]] && echo ","
            first=0
            printf '    "%s": [' "$c"
            sep=""
            while IFS= read -r verb; do
                printf '%s"%s"' "$sep" "$verb"
                sep=", "
            done <<< "$subs"
            printf ']'
        done
        echo ""
        echo -n "  }"
    fi
    if [[ "$ONLY" != "gh" ]]; then
        [[ "$ONLY" != "gitee" ]] && echo ","
        echo '  "gitee": {'
        first=1
        for c in "${GITEE_TOP_LEVEL[@]}"; do
            subs=$(extract_subcommands "$BINARY_GITEE" "$c")
            [[ -z "$subs" ]] && continue
            [[ $first -eq 0 ]] && echo ","
            first=0
            printf '    "%s": [' "$c"
            sep=""
            while IFS= read -r verb; do
                printf '%s"%s"' "$sep" "$verb"
                sep=", "
            done <<< "$subs"
            printf ']'
        done
        echo ""
        echo -n "  }"
    fi
    echo ""
    echo "}"
}

output_md() {
    if [[ "$ONLY" != "gitee" ]]; then
        echo "## gh 子命令清单"
        echo ""
        for c in "${GH_TOP_LEVEL[@]}"; do
            subs=$(extract_subcommands "$BINARY_GH" "$c")
            [[ -z "$subs" ]] && continue
            echo "- **gh $c**: $(echo "$subs" | tr '\n' ',' | sed 's/,$//' | sed 's/,/, /g')"
        done
        echo ""
    fi
    if [[ "$ONLY" != "gh" ]]; then
        echo "## gitee-cli 子命令清单"
        echo ""
        for c in "${GITEE_TOP_LEVEL[@]}"; do
            subs=$(extract_subcommands "$BINARY_GITEE" "$c")
            [[ -z "$subs" ]] && continue
            echo "- **gitee $c**: $(echo "$subs" | tr '\n' ',' | sed 's/,$//' | sed 's/,/, /g')"
        done
    fi
}

case "$OUTPUT_FORMAT" in
    json) output_json ;;
    md) output_md ;;
    *) output_text ;;
esac
