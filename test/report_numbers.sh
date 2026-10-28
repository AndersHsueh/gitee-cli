#!/usr/bin/env bash
#
# test/report_numbers.sh — canonical counts used in release notes and the
# READMEs (so docs never quote stale numbers). Read-only; always builds a
# fresh throwaway binary from the current source tree unless a binary is
# explicitly passed (a prebuilt binary is announced with a WARNING because
# its counts describe that binary, not the tree).
#
# Usage:
#   bash test/report_numbers.sh [BINARY]
#   P1_BIN=/path/to/gitee bash test/report_numbers.sh
#
# Env:
#   P1_AGENTHELP_TSV  optional path<TAB>kind<TAB>first-line cache (item E);
#                     when set, --agent-help is not re-probed for the count.
#
# Output (stable "label: value" format):
#   reference_unique_rows  — unique `gitee <cmd>` rows in `help reference`
#   reference_has_version  — yes/no
#   command_paths          — total unique command paths in the live tree
#   leaf_commands          — paths with no subcommands
#   agent_help_full_docs   — commands whose --agent-help emits a full markdown doc
#   test_packages          — `go list ./...` package count
#   test_files             — *_test.go file count
#   test_funcs             — ^func Test... count
#
set -o pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# Binary selection: numbers must describe the CURRENT source tree, so by
# default we always build a fresh throwaway binary. Reusing an arbitrary
# pre-existing binary (e.g. a stale /tmp/gitee-verify) silently produced
# wrong counts; it is now only allowed when explicitly requested.
BIN=${1:-${P1_BIN:-${GITEE_BIN:-}}}
TMPBIN_DIR=""
if [ -n "$BIN" ]; then
    echo "WARNING: using prebuilt binary: $BIN (counts describe that binary, not the current tree)" >&2
else
    TMPBIN_DIR=$(mktemp -d "${TMPDIR:-/tmp}/gitee-numbers.XXXXXX")
    ( cd "$ROOT_DIR" && go build -o "$TMPBIN_DIR/gitee" ./cmd/gitee ) || {
        echo "ERROR: go build ./cmd/gitee failed" >&2
        exit 2
    }
    BIN="$TMPBIN_DIR/gitee"
fi
[ -x "$BIN" ] || { echo "ERROR: not an executable gitee binary: $BIN" >&2; exit 2; }

TMPDIR_NUMS=$(mktemp -d "${TMPDIR:-/tmp}/gitee-numbers.XXXXXX")
trap 'rm -rf "$TMPDIR_NUMS" ${TMPBIN_DIR:+"$TMPBIN_DIR"}' EXIT

printf 'version               : %s\n' "$("$BIN" version 2>/dev/null | head -n 1)"
printf 'binary                : %s\n' "$BIN"

"$BIN" help reference 2>/dev/null | grep '^gitee ' | sed '/^$/d' \
    | sort -u > "$TMPDIR_NUMS/ref.txt"
REF_ROWS=$(wc -l < "$TMPDIR_NUMS/ref.txt" | tr -d ' ')
printf 'reference_unique_rows : %s\n' "$REF_ROWS"
if grep -qE '^gitee version([[:space:]]|$)' "$TMPDIR_NUMS/ref.txt"; then
    printf 'reference_has_version : yes\n'
else
    printf 'reference_has_version : NO (version command missing from reference)\n'
fi

awk '/^gitee / {
        line = $0
        sub(/^gitee /, "", line)
        sub(/  +.*$/, "", line)
        if (line != "") print line
    }' "$TMPDIR_NUMS/ref.txt" | sort -u > "$TMPDIR_NUMS/paths.txt"
printf 'command_paths         : %s\n' "$(wc -l < "$TMPDIR_NUMS/paths.txt" | tr -d ' ')"

LEAF=$(awk '
    { line[NR] = $0 }
    END {
        leaves = 0
        for (i = 1; i <= NR; i++) {
            is_leaf = 1
            for (j = 1; j <= NR; j++) {
                if (i != j && index(line[j], line[i] " ") == 1) { is_leaf = 0; break }
            }
            if (is_leaf) leaves++
        }
        print leaves
    }' "$TMPDIR_NUMS/paths.txt")
printf 'leaf_commands         : %s\n' "$LEAF"

# When p1-acceptance.sh runs this script it exports P1_AGENTHELP_TSV with the
# item-E cache (path<TAB>kind<TAB>first-line), so we do not re-probe ~177
# commands a second time.
if [ -n "${P1_AGENTHELP_TSV:-}" ] && [ -s "${P1_AGENTHELP_TSV:-}" ]; then
    FULL=$(awk -F'\t' '$2 == "full"' "$P1_AGENTHELP_TSV" | wc -l | tr -d ' ')
    printf 'agent_help_full_docs  : %s  (from item E cache)\n' "$FULL"
else
    FULL=0
    while IFS= read -r p; do
        [ -n "$p" ] || continue
        # shellcheck disable=SC2086
        first=$("$BIN" $p --agent-help 2>/dev/null | head -n 1)
        case "$first" in
            '# '*) FULL=$((FULL + 1)) ;;
        esac
    done < "$TMPDIR_NUMS/paths.txt"
    printf 'agent_help_full_docs  : %s\n' "$FULL"
fi

MATRIX=$(grep -cE '^[[:space:]]*gitee .*--agent-help[[:space:]]+(✅|🆕|❌|n/a)' \
    "$ROOT_DIR/cmd/agent_help.go" 2>/dev/null)
printf 'matrix_rows           : %s\n' "$MATRIX"

PKG=$( cd "$ROOT_DIR" && go list ./... 2>/dev/null | wc -l | tr -d ' ')
FILES=$( cd "$ROOT_DIR" && find . -name '*_test.go' -not -path './.git/*' | wc -l | tr -d ' ')
FUNCS=$( cd "$ROOT_DIR" && grep -rh '^func Test' --include='*_test.go' . 2>/dev/null | wc -l | tr -d ' ')
printf 'test_packages         : %s\n' "$PKG"
printf 'test_files            : %s\n' "$FILES"
printf 'test_funcs            : %s\n' "$FUNCS"
