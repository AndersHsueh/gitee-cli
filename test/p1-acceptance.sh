#!/usr/bin/env bash
#
# test/p1-acceptance.sh — independent P1 acceptance suite for gitee-cli.
#
# Verifies 14 items (A–N) defined by the P1 round's tester task. It never
# modifies product code or docs; it only builds a throwaway binary, runs
# read-only commands against Gitee with the local token (when available),
# and inspects the repository files. Items that need a token are SKIPped
# (never silently passed) when no token can be found.
#
# Usage:
#   bash test/p1-acceptance.sh                 # build + full A–N run
#   bash test/p1-acceptance.sh --no-build      # reuse $P1_BIN / autodetected binary
#   P1_SKIP_CI=1 bash test/p1-acceptance.sh    # skip N (slow) during quick loops
#   P1_REPO=owner/repo ...                     # repo used for read-only API checks
#
# Exit code: 0 = every item PASS or SKIP; 1 = at least one FAIL.
#
# Environment:
#   P1_BIN        pre-built binary (default: build to $TMPDIR)
#   GITEE_BIN     alias for P1_BIN
#   P1_WORKDIR    keep artifacts in this dir instead of a fresh mktemp dir
#   P1_REPO       repository for read-only API checks (default andershsueh/gitee-cli)
#   NO_COLOR      disable colors
#
set -o pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

NO_BUILD=0
for arg in "$@"; do
    case "$arg" in
        --no-build) NO_BUILD=1 ;;
        -h|--help) sed -n '2,30p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) echo "unknown argument: $arg (try --help)" >&2; exit 2 ;;
    esac
done

if [ -n "${P1_BIN:-}" ]; then
    BIN=$P1_BIN
elif [ -n "${GITEE_BIN:-}" ]; then
    BIN=$GITEE_BIN
else
    BIN=""
fi

WORK=${P1_WORKDIR:-}
if [ -z "$WORK" ]; then
    WORK=$(mktemp -d "${TMPDIR:-/tmp}/p1-acceptance.XXXXXX") || exit 2
fi
mkdir -p "$WORK"
if [ -z "$BIN" ]; then
    # Always a fresh build into the throwaway work dir; never reuse a stale
    # binary from /tmp. An explicitly supplied P1_BIN/GITEE_BIN is used as-is
    # (and announced), because that is the caller's deliberate choice.
    BIN="$WORK/gitee-verify"
else
    printf 'using prebuilt binary: %s (P1_BIN/GITEE_BIN explicitly set)\n' "$BIN"
fi

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    C_RED=$'\033[31m'; C_GREEN=$'\033[32m'; C_YELLOW=$'\033[33m'
    C_BOLD=$'\033[1m'; C_RESET=$'\033[0m'
else
    C_RED=""; C_GREEN=""; C_YELLOW=""; C_BOLD=""; C_RESET=""
fi

printf '%s gitee-cli P1 acceptance %s\n' "$C_BOLD" "$C_RESET"
printf 'root : %s\nwork : %s\n' "$ROOT_DIR" "$WORK"

if [ "$NO_BUILD" -eq 0 ]; then
    printf '\n==> building %s\n' "$BIN"
    if ! ( cd "$ROOT_DIR" && go build -o "$BIN" ./cmd/gitee ); then
        printf '%sBUILD FAILED%s\n' "$C_RED" "$C_RESET"
        exit 2
    fi
fi
if [ ! -x "$BIN" ]; then
    printf '%sERROR: no executable binary at %s (build it or pass P1_BIN)%s\n' \
        "$C_RED" "$BIN" "$C_RESET"
    exit 2
fi
printf 'bin  : %s\n' "$BIN"

# ---------------------------------------------------------------------------
# Isolated HOME carrying a read-only copy of the real token config.
# ---------------------------------------------------------------------------
THIS_HOME="$WORK/home"
mkdir -p "$THIS_HOME/.gitee-cli"
HAS_TOKEN=0
if [ -f "$HOME/.gitee-cli/settings.json" ]; then
    cp "$HOME/.gitee-cli/settings.json" "$THIS_HOME/.gitee-cli/settings.json" 2>/dev/null
    chmod 600 "$THIS_HOME/.gitee-cli/settings.json" 2>/dev/null
    if grep -q '"token"' "$THIS_HOME/.gitee-cli/settings.json" 2>/dev/null; then
        HAS_TOKEN=1
    fi
fi
if [ -n "${GITEE_TOKEN:-}" ]; then
    HAS_TOKEN=1
fi
TOKEN_STATE="token found"
[ "$HAS_TOKEN" -eq 1 ] || TOKEN_STATE="NO token (API items will SKIP)"

P1_REPO=${P1_REPO:-andershsueh/gitee-cli}

# Run the binary with the isolated HOME.
g() { HOME="$THIS_HOME" "$BIN" "$@"; }

# Capture stdout/stderr/exit code of a gitee invocation into RC/OUT/ERR.
gcap() {
    RC=0
    OUT=$(HOME="$THIS_HOME" "$BIN" "$@" 2>"$WORK/.stderr") || RC=$?
    ERR=$(cat "$WORK/.stderr")
}

# Same, with extra env assignments: gcapenv "GITEE_OUTPUT=json" repo list ...
gcapenv() {
    local envs=$1; shift
    RC=0
    OUT=$(env HOME="$THIS_HOME" $envs "$BIN" "$@" 2>"$WORK/.stderr") || RC=$?
    ERR=$(cat "$WORK/.stderr")
}

first_line() { printf '%s' "$1" | head -n 1; }

# ---------------------------------------------------------------------------
# Per-item bookkeeping
# ---------------------------------------------------------------------------
FAIL_N=0
PASS_N=0
SKIP_N=0
ITEM=""
ITEM_FAILS=0
ITEM_ERRS=""
ITEM_NOTE=""
ITEM_SKIPPED=0
SUMMARY="$WORK/summary.tsv"
: > "$SUMMARY"

item_begin() {
    ITEM=$1
    ITEM_FAILS=0
    ITEM_ERRS=""
    ITEM_NOTE=""
    ITEM_SKIPPED=0
    printf '\n%s--- %s ---%s\n' "$C_BOLD" "$ITEM" "$C_RESET"
}

item_fail() {
    ITEM_FAILS=$((ITEM_FAILS + 1))
    ITEM_ERRS="$ITEM_ERRS"$'\n'"    $*"
}

item_note() { ITEM_NOTE="$*"; }

item_fail_file() {
    local f=$1 max=${2:-8} n=0 line
    [ -s "$f" ] || return 0
    while IFS= read -r line; do
        [ -n "$line" ] || continue
        item_fail "$line"
        n=$((n + 1))
        [ "$n" -ge "$max" ] && break
    done < "$f"
}

item_finish() {
    if [ "$ITEM_SKIPPED" -eq 1 ]; then
        return
    fi
    if [ "$ITEM_FAILS" -gt 0 ]; then
        FAIL_N=$((FAIL_N + 1))
        printf '%s[FAIL]%s %s — %d problem(s)\n' "$C_RED" "$C_RESET" "$ITEM" "$ITEM_FAILS"
        printf '%s\n' "$ITEM_ERRS" | sed '/^$/d' | head -n 10
        printf '%s\tFAIL\t%s\n' "$ITEM" "$ITEM_NOTE" >> "$SUMMARY"
    else
        PASS_N=$((PASS_N + 1))
        printf '%s[PASS]%s %s %s\n' "$C_GREEN" "$C_RESET" "$ITEM" "$ITEM_NOTE"
        printf '%s\tPASS\t%s\n' "$ITEM" "$ITEM_NOTE" >> "$SUMMARY"
    fi
}

item_skip() {
    ITEM_SKIPPED=1
    SKIP_N=$((SKIP_N + 1))
    printf '%s[SKIP]%s %s — %s\n' "$C_YELLOW" "$C_RESET" "$ITEM" "$*"
    printf '%s\tSKIP\t%s\n' "$ITEM" "$*" >> "$SUMMARY"
}

# Collect command paths from `help reference` (deduped) into a file.
collect_tree() {
    gcap help reference
    printf '%s\n' "$OUT" | awk '
        /^gitee / {
            line = $0
            sub(/^gitee /, "", line)
            sub(/  +.*$/, "", line)
            if (line != "" && !seen[line]++) print line
        }' > "$WORK/tree-paths.txt"
    wc -l < "$WORK/tree-paths.txt" | tr -d ' '
}

# ===========================================================================
# A. Every documented `gitee ...` example is runnable.
# ===========================================================================
item_begin "A. docs examples (README*.md + docs/**/*.md)"
if [ ! -f "$ROOT_DIR/test/verify_docs_examples.py" ]; then
    item_fail "test/verify_docs_examples.py is missing"
else
    RC=0
    OUT=$(python3 "$ROOT_DIR/test/verify_docs_examples.py" --binary "$BIN" \
        --root "$ROOT_DIR" 2>&1) || RC=$?
    printf '%s\n' "$OUT" | grep -E '^(scanned|validated|all documented)' | sed 's/^/    /'
    if [ "$RC" -ne 0 ]; then
        printf '%s\n' "$OUT" | grep -E '^  [^ ].*:[0-9]+:' | head -n 8 > "$WORK/A.err"
        item_fail_file "$WORK/A.err" 8
    fi
    item_note "$(printf '%s\n' "$OUT" | grep -E '^validated' | head -n 1 | sed 's/^validated \([0-9]*\) runnable examples.*/\1 runnable examples/')"
fi
item_finish

# ===========================================================================
# B. Legacy group syntax must fail loudly; bare groups still print help.
# ===========================================================================
item_begin "B. legacy group syntax (repo star/watch/protect)"
check_group_legacy() {
    local verb=$1 hint=$2
    gcap repo "$verb" owner/repo
    if [ "$RC" -eq 0 ]; then
        item_fail "gitee repo $verb owner/repo exited 0 (must fail with a hint)"
        return
    fi
    combined="$OUT$ERR"
    case "$combined" in
        *"$hint"*|*"correct subcommand"*|*"subcommand"*) ;;
        *) item_fail "gitee repo $verb owner/repo failed (rc=$RC) without hinting the correct subcommand: $(first_line "$combined")" ;;
    esac
}
check_group_legacy star "star star"
check_group_legacy watch "watch watch"
check_group_legacy protect "protect protect"
for verb in star watch protect; do
    gcap repo "$verb"
    if [ "$RC" -ne 0 ]; then
        item_fail "gitee repo $verb (no args) exited $RC; should print help and exit 0"
    fi
done
item_note "legacy syntax rejected, bare groups exit 0"
item_finish

# ===========================================================================
# C. Strict positional arguments on list commands.
# ===========================================================================
item_begin "C. strict args (repo/issue/gist list stray)"
if [ "$HAS_TOKEN" -eq 0 ]; then
    # C mixes local arg-parsing checks with read-only API checks ("stray" must
    # be rejected *and* legitimate positionals must still work). Without a
    # token the item is unverifiable end-to-end, so it is SKIP, never PASS.
    item_skip "no Gitee token (GITEE_TOKEN unset, no settings.json); API half unverifiable"
else
    for cmd in "repo list" "issue list" "gist list"; do
        gcap $cmd stray
        if [ "$RC" -eq 0 ]; then
            item_fail "gitee $cmd stray exited 0 (must reject the stray argument)"
            continue
        fi
        if ! printf '%s' "$OUT$ERR" | grep -qiE 'unknown|unexpected|accepts|arg'; then
            item_fail "gitee $cmd stray failed without an argument error: $(first_line "$OUT$ERR")"
        fi
    done
    gcap repo view "$P1_REPO" -o json
    if [ "$RC" -ne 0 ]; then
        item_fail "gitee repo view $P1_REPO (positional repo) must still work, exited $RC: $(first_line "$ERR")"
    fi
    gcap repo list --limit 1
    if [ "$RC" -ne 0 ]; then
        item_fail "gitee repo list --limit 1 exited $RC (over-strict args?): $(first_line "$ERR")"
    fi
fi
item_finish

# ===========================================================================
# D. -R shorthand consistency for every command defining --repo.
# ===========================================================================
item_begin "D. --repo has -R shorthand everywhere"
if [ "$HAS_TOKEN" -eq 0 ]; then
    item_skip "no Gitee token (GITEE_TOKEN unset, no settings.json); -R runtime spot checks need the API"
else
    # One root --help: -R is a *persistent* flag, so every command inherits it.
    gcap --help
    if ! printf '%s' "$OUT" | grep -qE '^[[:space:]]+-R, --repo'; then
        item_fail "gitee --help: global --repo has no -R shorthand"
    fi
    # Local registrations could shadow it; verify them from source in one pass.
    : > "$WORK/d-local.txt"
    # Flag *registrations* only (not form.Set("repo", ...) API payloads).
    grep -rnE '\.[A-Za-z]+VarP?\([^)]*"repo"' --include='*.go' "$ROOT_DIR/cmd" "$ROOT_DIR/internal" 2>/dev/null \
        | grep -v '_test\.go:' | grep -v '/help_topics/' \
        | grep -vE '"repo", *"R"' >> "$WORK/d-local.txt" || true
    if [ -s "$WORK/d-local.txt" ]; then
        item_fail "local --repo flag registration without -R:"
        item_fail_file "$WORK/d-local.txt" 8
    fi
    # Sample the tree (one batch of --help calls): top-level groups + the
    # representative leaves the task asks to spot-check.
    : > "$WORK/d-sample.txt"
    printf '%s\n' "$OUT" | awk '
        /^Available Commands:/ { f = 1; next }
        f && /^[[:space:]]+[A-Za-z]/ { print $1; next }
        f && !/^[[:space:]]/ { f = 0 }
    ' >> "$WORK/d-sample.txt"
    for p in "issue list" "notification list" "repo list" "gist list" "pr list" "search repos"; do
        printf '%s\n' "$p" >> "$WORK/d-sample.txt"
    done
    sort -u "$WORK/d-sample.txt" > "$WORK/d-sample-u.txt"
    : > "$WORK/d-missing.txt"
    while IFS= read -r path; do
        [ -n "$path" ] || continue
        # shellcheck disable=SC2086
        gcap $path --help
        if printf '%s' "$OUT" | grep -q -- '--repo'; then
            if ! printf '%s' "$OUT" | grep -qE '^[[:space:]]+-R, --repo'; then
                printf 'gitee %s\n' "$path" >> "$WORK/d-missing.txt"
            fi
        fi
    done < "$WORK/d-sample-u.txt"
    if [ -s "$WORK/d-missing.txt" ]; then
        item_fail "sampled commands expose --repo without -R:"
        item_fail_file "$WORK/d-missing.txt" 8
    else
        item_note "root -R,--repo + $(wc -l < "$WORK/d-sample-u.txt" | tr -d ' ') sampled helps OK"
    fi
    for cmd in "issue list" "notification list" "repo list" "gist list"; do
        # shellcheck disable=SC2086
        gcap $cmd -R "$P1_REPO" --limit 1
        if [ "$RC" -ne 0 ] && printf '%s' "$ERR" | grep -qiE 'unknown shorthand|unknown flag'; then
            item_fail "gitee $cmd -R $P1_REPO --limit 1: $(first_line "$ERR")"
        fi
    done
fi
item_finish

# ===========================================================================
# E. agent-help coverage matrix matches the live command tree.
# ===========================================================================
item_begin "E. agent-help matrix vs live tree"
if [ ! -f "$WORK/tree-paths.txt" ]; then
    collect_tree > /dev/null
fi
HOME="$THIS_HOME" python3 - "$BIN" "$ROOT_DIR" "$WORK" > "$WORK/E.out" 2>&1 <<'PY'
import os, re, subprocess, sys

binp, root, work = sys.argv[1], sys.argv[2], sys.argv[3]
paths = [l.strip() for l in open(os.path.join(work, "tree-paths.txt")) if l.strip()]

# `help reference` may omit top-level commands (e.g. version); --help has them.
env = dict(os.environ)
env.setdefault("NO_COLOR", "1")
help_out = subprocess.run([binp, "--help"], capture_output=True, text=True,
                          timeout=120, env=env).stdout
section = False
for line in help_out.splitlines():
    if line.startswith("Available Commands:"):
        section = True
        continue
    if section:
        m = re.match(r"^  ([A-Za-z][\w.-]*)\s{2,}", line)
        if m:
            if m.group(1) not in paths:
                paths.append(m.group(1))
        elif not line.strip():
            section = False

cap = {}
rows = []
for p in paths:
    proc = subprocess.run([binp] + p.split() + ["--agent-help"],
                          capture_output=True, text=True, timeout=120, env=env)
    first = (proc.stdout.strip().splitlines() or [""])[0]
    if first.startswith("# "):
        kind = "full"
    elif "operational/config" in proc.stderr:
        kind = "na"
    else:
        kind = "other"
    cap[p] = kind
    rows.append("\t".join([p, kind, first]))
with open(os.path.join(work, "agenthelp.tsv"), "w", encoding="utf-8") as fh:
    fh.write("\n".join(rows) + "\n")

src = open(os.path.join(root, "cmd", "agent_help.go"), encoding="utf-8").read()
matrix = {}
for line in src.splitlines():
    m = re.match(r"^\s*gitee\s+(.*?)\s+--agent-help\s+(✅|🆕|❌|n/a)(?=\s|$)", line)
    if not m:
        continue
    p = m.group(1).strip()
    if any(c in p for c in "<>[]{}"):
        continue
    matrix[p] = m.group(2)

treeset = set(paths)
errs = []
for p in sorted(matrix):
    if p not in treeset:
        errs.append("ghost matrix row (path not in live tree): gitee %s --agent-help" % p)
for p in sorted(matrix):
    status = matrix[p]
    if status in ("✅", "🆕") and p in treeset and cap.get(p) != "full":
        errs.append("matrix claims available but --agent-help is not a full doc: "
                    "gitee %s (got %s)" % (p, cap.get(p)))
for p in paths:
    if cap.get(p) == "full" and p not in matrix:
        errs.append("live --agent-help command missing from coverage matrix: gitee %s" % p)

full = sum(1 for v in cap.values() if v == "full")
print("matrix rows: %d | live tree: %d | full agent-help docs: %d"
      % (len(matrix), len(paths), full))
if errs:
    with open(os.path.join(work, "E.err"), "w", encoding="utf-8") as fh:
        fh.write("\n".join(errs) + "\n")
    print("E problems: %d" % len(errs))
    sys.exit(1)
sys.exit(0)
PY
rc=$?
cat "$WORK/E.out" | sed 's/^/    /'
if [ "$rc" -ne 0 ]; then
    item_fail_file "$WORK/E.err" 10
fi
item_note "$(head -n 1 "$WORK/E.out")"
item_finish

# ===========================================================================
# F. First line of every full agent-help doc is `# gitee <path> — Agent Help`.
# ===========================================================================
item_begin "F. agent-help titles"
if [ ! -s "$WORK/agenthelp.tsv" ]; then
    item_fail "agent-help cache missing (item E did not produce $WORK/agenthelp.tsv)"
else
    python3 - "$WORK" > "$WORK/F.out" 2>&1 <<'PY'
import os, sys

work = sys.argv[1]
errs = []
checked = 0
with open(os.path.join(work, "agenthelp.tsv"), encoding="utf-8") as fh:
    for line in fh:
        line = line.rstrip("\n")
        if not line:
            continue
        p, kind, first = line.split("\t", 2)
        if kind != "full":
            continue
        checked += 1
        want = "# gitee %s — Agent Help" % p
        if first != want:
            errs.append('gitee %s: first line is "%s" (want "%s")'
                        % (p, first[:90], want))
print("checked %d full agent-help docs" % checked)
if errs:
    with open(os.path.join(work, "F.err"), "w", encoding="utf-8") as fh:
        fh.write("\n".join(errs) + "\n")
    print("F problems: %d" % len(errs))
    sys.exit(1)
sys.exit(0)
PY
    rc=$?
    cat "$WORK/F.out" | sed 's/^/    /'
    if [ "$rc" -ne 0 ]; then
        item_fail_file "$WORK/F.err" 10
    fi
    item_note "$(head -n 1 "$WORK/F.out")"
fi
item_finish

# ===========================================================================
# G. Output formats: table/json only; csv/yaml rejected; no stale csv claims.
# ===========================================================================
item_begin "G. -o output formats"
if [ "$HAS_TOKEN" -eq 0 ]; then
    item_skip "no Gitee token (GITEE_TOKEN unset, no settings.json); format behaviour needs the API"
else
    for bad in csv yaml; do
        gcap repo list -o "$bad" --limit 1
        if [ "$RC" -eq 0 ]; then
            item_fail "gitee repo list -o $bad exited 0 (must reject unsupported format)"
        elif ! printf '%s' "$OUT$ERR" | grep -qiE 'invalid|unknown|unsupported|format'; then
            item_fail "gitee repo list -o $bad failed without a format error: $(first_line "$OUT$ERR")"
        fi
    done
    gcap repo list -o json --limit 1
    if [ "$RC" -ne 0 ]; then
        item_fail "gitee repo list -o json exited $RC: $(first_line "$ERR")"
    else
        printf '%s' "$OUT" | python3 -c 'import json,sys; json.load(sys.stdin)' 2>/dev/null \
            || item_fail "gitee repo list -o json did not emit valid JSON"
    fi
    gcap repo list -o table --limit 1
    if [ "$RC" -ne 0 ]; then
        item_fail "gitee repo list -o table exited $RC: $(first_line "$ERR")"
    fi
    gcap --help
    if printf '%s' "$OUT" | grep -qi 'csv'; then
        item_fail "gitee --help still advertises csv: $(printf '%s' "$OUT" | grep -i csv | head -n 1)"
    fi
    gcap help formatting
    if printf '%s' "$OUT" | grep -qi 'csv'; then
        item_fail "gitee help formatting still advertises csv: $(printf '%s' "$OUT" | grep -i csv | head -n 1)"
    fi
fi
item_finish

# ===========================================================================
# H. Environment variables: documented set == implemented set.
# ===========================================================================
item_begin "H. environment variables"
if [ "$HAS_TOKEN" -eq 0 ]; then
    item_skip "no Gitee token (GITEE_TOKEN unset, no settings.json); behaviour assertions need the API"
else
ALLOWED_VARS="GITEE_TOKEN GITEE_HOSTNAME GITEE_API_URL GITEE_OUTPUT GITEE_VERBOSE GITEE_TIMEOUT GITEE_LIMIT GITEE_PAGINATE GITEE_NO_EXTENSIONS NO_COLOR GITEE_ALIASES_FILE GITEE_EXT_DIR BROWSER"
EXEMPT_VARS="HTTPS_PROXY"
gcap help environment
printf '%s\n' "$OUT" | awk '
    /^[A-Z][A-Z0-9_]+[[:space:]][[:space:]]/ && $1 != "VARIABLE" { print $1 }' \
    | sort -u > "$WORK/h-listed.txt"
if [ ! -s "$WORK/h-listed.txt" ]; then
    item_fail "gitee help environment listed no variables"
fi
# documented but not implemented: only *implementation* code counts.
# help_topics/ documents the names (searching it made this check always-true)
# and *_test.go is not implementation evidence.
: > "$WORK/h-phantom.raw"
while IFS= read -r v; do
    [ -n "$v" ] || continue
    case " $ALLOWED_VARS $EXEMPT_VARS " in
        *" $v "*) ;;
        *) echo "$v" >> "$WORK/h-phantom.raw"; continue ;;
    esac
    # HTTPS_PROXY is honoured by net/http's ProxyFromEnvironment, not our code.
    [ "$v" = "HTTPS_PROXY" ] && continue
    if ! grep -rlE "\"$v\"" --include='*.go' "$ROOT_DIR/cmd" "$ROOT_DIR/internal" 2>/dev/null \
            | grep -v '/help_topics/' | grep -v '_test\.go$' | head -n 1 | grep -q .; then
        echo "$v" >> "$WORK/h-phantom.raw"
    fi
done < "$WORK/h-listed.txt"
sort -u "$WORK/h-phantom.raw" > "$WORK/h-phantom.txt"
# allowed but no longer documented
: > "$WORK/h-missing.txt"
for v in $ALLOWED_VARS; do
    if ! grep -qx "$v" "$WORK/h-listed.txt"; then
        echo "$v" >> "$WORK/h-missing.txt"
    fi
done
if [ -s "$WORK/h-phantom.txt" ]; then
    item_fail "help environment lists variables that are not implemented:"
    item_fail_file "$WORK/h-phantom.txt" 8
fi
if [ -s "$WORK/h-missing.txt" ]; then
    item_fail "help environment is missing expected variables:"
    item_fail_file "$WORK/h-missing.txt" 8
fi
    # GITEE_OUTPUT must default the output format.
    gcapenv "GITEE_OUTPUT=json" repo list --limit 1
    if [ "$RC" -ne 0 ]; then
        item_fail "GITEE_OUTPUT=json gitee repo list exited $RC: $(first_line "$ERR")"
    else
        printf '%s' "$OUT" | python3 -c 'import json,sys; json.load(sys.stdin)' 2>/dev/null \
            || item_fail "GITEE_OUTPUT=json was not honoured (output is not JSON)"
    fi
    # GITEE_LIMIT must default --limit.
    gcapenv "GITEE_LIMIT=1" repo list -o json
    if [ "$RC" -ne 0 ]; then
        item_fail "GITEE_LIMIT=1 gitee repo list -o json exited $RC"
    else
        n=$(printf '%s' "$OUT" | python3 -c 'import json,sys
try: print(len(json.load(sys.stdin)))
except Exception: print(-1)' 2>/dev/null)
        if [ "$n" = "-1" ]; then
            item_fail "GITEE_LIMIT=1: output is not JSON"
        elif [ "$n" -gt 1 ]; then
            item_fail "GITEE_LIMIT=1 was not honoured: $n results returned"
        fi
    fi
    # GITEE_VERBOSE must enable HTTP tracing on stderr.
    RC=0
    OUT=$(env HOME="$THIS_HOME" GITEE_VERBOSE=1 "$BIN" repo list --limit 1 2>"$WORK/h-verbose.err") || RC=$?
    if [ "$RC" -ne 0 ]; then
        item_fail "GITEE_VERBOSE=1 gitee repo list exited $RC"
    elif ! grep -qiE 'http|GET ' "$WORK/h-verbose.err"; then
        item_fail "GITEE_VERBOSE=1 was not honoured (no HTTP trace on stderr)"
    fi
    # GITEE_TOKEN must override the stored token (invalid env token => auth fails).
    RC=0
    OUT=$(env HOME="$THIS_HOME" GITEE_TOKEN=p1-invalid-token "$BIN" auth status 2>"$WORK/h-token.err") || RC=$?
    if [ "$RC" -eq 0 ]; then
        item_fail "GITEE_TOKEN did not override the stored token (auth status succeeded with an invalid token)"
    fi
    # Remaining vars: env-derived flag defaults are observable in --help.
    for pair in \
        "GITEE_TIMEOUT=42|--timeout int|42" \
        "GITEE_HOSTNAME=enterprise.example.com|--hostname string|enterprise.example.com" \
        "GITEE_API_URL=http://127.0.0.1:9/api/v5|--api-url string|127.0.0.1:9" \
        "GITEE_PAGINATE=1|-P, --paginate|default true" \
        "GITEE_NO_EXTENSIONS=1|--no-extensions|default true"; do
        envs=${pair%%|*}
        rest=${pair#*|}
        flag=${rest%%|*}
        want=${rest#*|}
        RC=0
        OUT=$(env HOME="$THIS_HOME" "$envs" "$BIN" repo list --help 2>&1) || RC=$?
        line=$(printf '%s\n' "$OUT" | grep -E -- "$flag" | head -n 1)
        case "$line" in
            *"$want"*) ;;
            *) item_fail "$envs not reflected in $flag default (got: ${line:-<absent>})" ;;
        esac
    done
    item_note "listed=$(wc -l < "$WORK/h-listed.txt" | tr -d ' ') vars; $TOKEN_STATE"
fi
item_finish

# ===========================================================================
# I. Search: honest empty results, friendly 404s, no stale README claims.
# ===========================================================================
item_begin "I. search honesty"
if [ "$HAS_TOKEN" -eq 0 ]; then
    item_skip "no Gitee token (GITEE_TOKEN unset, no settings.json); empty-result/404 behaviour needs the API"
else
    gcap search repos "p1-no-such-query-zzz-98765" -o json
    if [ "$RC" -ne 0 ]; then
        item_fail "gitee search repos <no-match> exited $RC: $(first_line "$ERR")"
    elif ! printf '%s' "$ERR" | grep -qiE 'no result|empty|无结果|not found|repo list --owner'; then
        item_fail "gitee search repos with no matches printed no stderr hint"
    fi
    gcap search code "func main" --repo "$P1_REPO"
    if ! printf '%s' "$OUT$ERR" | grep -qiE '404|not available|not supported|不支持|unsupported'; then
        item_fail "gitee search code did not give a friendly 404/not-supported message: $(first_line "$OUT$ERR")"
    fi
    gcap search commits "init" --repo "$P1_REPO"
    if ! printf '%s' "$OUT$ERR" | grep -qiE '404|not available|not supported|不支持|unsupported'; then
        item_fail "gitee search commits did not give a friendly 404/not-supported message: $(first_line "$OUT$ERR")"
    fi
    : > "$WORK/i-claims.txt"
    for f in "$ROOT_DIR"/README*.md; do
        [ -f "$f" ] || continue
        grep -nEi 'search.*\b(code|commits)\b|\b(code|commits)\b.*search' "$f" 2>/dev/null \
            | grep -viE 'not available|not supported|unsupported|unavailable|deprecated|removed|不支持|不可用|非対応|一部制限|404|no results|见下注|see (the )?note|see below' \
            | sed "s|^|$(basename "$f"):|" >> "$WORK/i-claims.txt"
    done
    if [ -s "$WORK/i-claims.txt" ]; then
        item_fail "README still claims code/commits search as available:"
        item_fail_file "$WORK/i-claims.txt" 6
    fi
fi
item_finish

# ===========================================================================
# J. auth docs describe PAT verification, not an OAuth2 device flow.
# ===========================================================================
item_begin "J. auth docs (PAT, no OAuth2 device flow)"
: > "$WORK/j-device.txt"
J_FILES="$ROOT_DIR/README_en.md $ROOT_DIR/README_ja.md $ROOT_DIR/.gitee/agent.md
$ROOT_DIR/docs/en/FSD.md $ROOT_DIR/docs/en/SDD.md
$ROOT_DIR/docs/ja/FSD.md $ROOT_DIR/docs/ja/SDD.md
$ROOT_DIR/docs/zh/FSD.md $ROOT_DIR/docs/zh/SDD.md"

# Device-flow wording is only a problem when it presents OAuth2/device flow
# as the current mechanism. Negations ("does NOT run a device flow", "no
# OAuth2 device flow"), status banners, and adjacent-context disclaimers are
# ignored. Handled in python for reliable multi-language word handling.
J_CHECKLIST="$WORK/j-files.txt"
: > "$J_CHECKLIST"
for f in $J_FILES; do
    [ -f "$f" ] && printf '%s\n' "$f" >> "$J_CHECKLIST"
done
# Generated onboard template (what `gitee agent onboard` writes for agents).
rm -rf "$WORK/onboard"; mkdir -p "$WORK/onboard"
if ( cd "$WORK/onboard" && HOME="$THIS_HOME" "$BIN" agent onboard >"$WORK/j-onboard.log" 2>&1 ); then
    if [ -f "$WORK/onboard/.gitee/agent.md" ]; then
        printf '%s\n' "$WORK/onboard/.gitee/agent.md" >> "$J_CHECKLIST"
        grep -nE -- '--delete-branch|--with-token|--no-interaction|--lang([^a-z]|$)|--include' \
            "$WORK/onboard/.gitee/agent.md" 2>/dev/null \
            | sed 's|^|onboard-template-fake-flag:|' >> "$WORK/j-device.txt"
    else
        item_fail "gitee agent onboard did not create .gitee/agent.md"
    fi
else
    item_fail "gitee agent onboard failed: $(first_line "$(cat "$WORK/j-onboard.log" 2>/dev/null)")"
fi
# The source template itself is also user-visible documentation.
if [ -f "$ROOT_DIR/cmd/agent/onboard.go" ]; then
    printf '%s\n' "$ROOT_DIR/cmd/agent/onboard.go" >> "$J_CHECKLIST"
fi
python3 - "$WORK/j-files.txt" "$WORK/j-device.txt" <<'PY'
import os, re, sys

checklist, out = sys.argv[1], sys.argv[2]
device_re = re.compile(
    r"(?i)oauth ?2?(\.0)?[^\n]{0,60}(device|设备)|device[ -]?(flow|authorization)"
    r"|设备流|设备授权|浏览器授权|browser[ -]auth"
)
neg_words = re.compile(r"(?i)\b(no|not|never|without|removed|deprecated|outdated|none)\b")
neg_phrases = re.compile(
    r"(?i)no longer|instead of|not implemented|out of date|no browser|does not|do not"
    r"|rather than|previously|historically|was documented|used to|していました|以前は|旧版"
    r"|不再|没有|并非|未实现|未実装|不一致|过期|古く|削除|廃止|非対応|无|無"
)
findings = []
for path in [l.strip() for l in open(checklist) if l.strip()]:
    if not os.path.isfile(path):
        continue
    if "onboard/.gitee/agent.md" in path:
        label = "onboard-template"
    elif path.endswith("/.gitee/agent.md"):
        label = ".gitee/agent.md"
    else:
        label = os.path.basename(path)
    with open(path, encoding="utf-8", errors="replace") as fh:
        lines = fh.read().splitlines()
    banner_end = 12 if any("⚠️" in l for l in lines[:15]) else 0
    for i, line in enumerate(lines):
        if i < banner_end or not device_re.search(line):
            continue
        context = " ".join(lines[max(0, i - 1):i + 2])
        if neg_words.search(context) or neg_phrases.search(context):
            continue
        findings.append("%s:%d:%s" % (label, i + 1, line.strip()))
with open(out, "w", encoding="utf-8") as fh:
    fh.write("\n".join(findings) + ("\n" if findings else ""))
PY
if [ -s "$WORK/j-device.txt" ]; then
    item_fail "OAuth2 device-flow wording still presented as current behaviour:"
    item_fail_file "$WORK/j-device.txt" 6
fi
gcap auth login --help
if printf '%s' "$OUT" | grep -qiE 'oauth|device flow|browser'; then
    item_fail "auth login --help still mentions OAuth2/device flow/browser authorization"
fi
if ! printf '%s' "$OUT" | grep -qiE 'token'; then
    item_fail "auth login --help does not describe the PAT/token flow"
fi
if ! printf '%s' "$OUT" | grep -qiE 'verif|validat|校验|验证'; then
    item_fail "auth login --help does not say the token is verified"
fi
item_finish

# ===========================================================================
# K. Multilingual README consistency + reachable relative links.
# ===========================================================================
item_begin "K. multilingual README"
for f in README_en.md README_ja.md; do
    if [ ! -f "$ROOT_DIR/$f" ]; then
        item_fail "$f is missing"
        continue
    fi
    if grep -qE -- '--no-interaction' "$ROOT_DIR/$f"; then
        item_fail "$f still documents --no-interaction"
    fi
    if grep -qE -- '--lang([^a-z]|$)' "$ROOT_DIR/$f"; then
        item_fail "$f still documents --lang"
    fi
    if ! grep -q '\.tar\.gz' "$ROOT_DIR/$f"; then
        item_fail "$f does not mention .tar.gz release artifacts"
    fi
    if ! grep -q '\.zip' "$ROOT_DIR/$f"; then
        item_fail "$f does not mention .zip release artifacts"
    fi
    if grep -qE '(gitee\.com|github\.com)/gitee/gitee-cli' "$ROOT_DIR/$f"; then
        item_fail "$f references the non-existent gitee/gitee-cli repository"
    fi
done
if [ -e "$ROOT_DIR/README_jp.md" ]; then
    item_fail "README_jp.md still exists (duplicate of README_ja.md)"
fi
python3 - "$ROOT_DIR" > "$WORK/K.out" 2>&1 <<'PY'
import os, re, sys

root = sys.argv[1]
files = []
for name in sorted(os.listdir(root)):
    if name.startswith("README") and name.endswith(".md"):
        files.append(os.path.join(root, name))
for dirpath, _dirs, names in os.walk(os.path.join(root, "docs")):
    for name in sorted(names):
        if name.endswith(".md"):
            files.append(os.path.join(dirpath, name))

link_re = re.compile(r"!?\[[^\]]*\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
ref_re = re.compile(r"^\[[^\]]+\]:\s*(\S+)")
errs = []
count = 0
for path in files:
    in_fence = False
    with open(path, encoding="utf-8", errors="replace") as fh:
        for lineno, line in enumerate(fh, 1):
            if re.match(r"^\s*(```|~~~)", line):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            targets = [m.group(1) for m in link_re.finditer(line)]
            m = ref_re.match(line)
            if m:
                targets.append(m.group(1))
            for target in targets:
                target = target.strip("<>")
                if not target or re.match(r"^(https?:|mailto:|tel:|#|//)", target):
                    continue
                target = target.split("#")[0].split("?")[0]
                if not target:
                    continue
                count += 1
                if target.startswith("/"):
                    continue  # site-absolute link, not a repo file
                resolved = os.path.normpath(os.path.join(os.path.dirname(path), target))
                if not os.path.exists(resolved):
                    errs.append("%s:%d: broken relative link %s"
                                % (os.path.relpath(path, root), lineno, target))
print("checked %d relative links in %d files" % (count, len(files)))
if errs:
    print("K problems: %d" % len(errs))
    for e in errs:
        print(e)
    sys.exit(1)
PY
rc=$?
cat "$WORK/K.out" | sed 's/^/    /'
if [ "$rc" -ne 0 ]; then
    printf '%s\n' "$(cat "$WORK/K.out")" | grep 'broken relative link' | head -n 8 > "$WORK/K.err"
    item_fail_file "$WORK/K.err" 8
fi
item_finish

# ===========================================================================
# L. Status banners on the three design docs in all three languages.
# ===========================================================================
item_begin "L. docs status banners"
: > "$WORK/l-missing.txt"
for lang in en ja zh; do
    for doc in FSD SDD developer-guide; do
        f="$ROOT_DIR/docs/$lang/$doc.md"
        rel="docs/$lang/$doc.md"
        if [ ! -f "$f" ]; then
            echo "$rel: missing" >> "$WORK/l-missing.txt"
            continue
        fi
        head -n 15 "$f" > "$WORK/l-head.txt"
        if ! grep -q '⚠️' "$WORK/l-head.txt"; then
            echo "$rel: no ⚠️ status banner in the first 15 lines" >> "$WORK/l-missing.txt"
            continue
        fi
        if ! grep -qiE 'not implemented|not consistent|no longer|outdated|out of date|obsolete|stale|design draft|design document|未実装|未实现|不一致|过期|历史|设计文档|設計ドラフト' "$WORK/l-head.txt"; then
            echo "$rel: banner does not say the document may be outdated" >> "$WORK/l-missing.txt"
        fi
    done
done
if [ -s "$WORK/l-missing.txt" ]; then
    item_fail "design docs without a proper status banner:"
    item_fail_file "$WORK/l-missing.txt" 9
fi
item_finish

# ===========================================================================
# M. `gitee help reference`: no duplicate rows, includes version.
# ===========================================================================
item_begin "M. help reference"
gcap help reference
printf '%s\n' "$OUT" | grep '^gitee ' > "$WORK/m-lines.txt"
total=$(wc -l < "$WORK/m-lines.txt" | tr -d ' ')
uniq_n=$(sort -u "$WORK/m-lines.txt" | wc -l | tr -d ' ')
if [ "$total" -eq 0 ]; then
    item_fail "help reference produced no command rows"
elif [ "$total" -ne "$uniq_n" ]; then
    sort "$WORK/m-lines.txt" | uniq -d > "$WORK/m-dups.txt"
    item_fail "help reference has $((total - uniq_n)) duplicate row(s):"
    item_fail_file "$WORK/m-dups.txt" 8
fi
if ! printf '%s\n' "$OUT" | grep -qE '^gitee version([[:space:]]|$)'; then
    item_fail "help reference does not list the version command"
fi
item_note "$uniq_n unique command rows"
item_finish

# ===========================================================================
# N. Full CI gate + go test.
# ===========================================================================
item_begin "N. CI gate (scripts/ci.sh)"
if [ "${P1_SKIP_CI:-0}" = "1" ]; then
    item_skip "P1_SKIP_CI=1"
else
    # golangci-lint is optional: use it when reachable via PATH or the local
    # tool dir; a missing linter is reported as SKIP lint, never a FAIL.
    LINT_NOTE=""
    if command -v golangci-lint >/dev/null 2>&1; then
        :
    elif [ -x /tmp/glbin/golangci-lint ]; then
        PATH="/tmp/glbin:$PATH"
    else
        printf '    SKIP lint (golangci-lint not installed; ci.sh notes the skip)\n'
        LINT_NOTE="lint skipped (not installed)"
    fi
    RC=0
    ( cd "$ROOT_DIR" && bash scripts/ci.sh ) > "$WORK/ci.log" 2>&1 || RC=$?
    if [ "$RC" -ne 0 ]; then
        item_fail "bash scripts/ci.sh exited $RC; last lines:"
        tail -n 12 "$WORK/ci.log" > "$WORK/ci-tail.txt"
        item_fail_file "$WORK/ci-tail.txt" 8
    else
        # ci.sh already runs `go test ./... -race -count=1`; no duplicate run.
        grep -E 'all Go files are gofmt-clean|linux/amd64.*build OK|All checks passed' \
            "$WORK/ci.log" | sed 's/^/    /'
        item_note "ci.sh green (go test -race included)${LINT_NOTE:+; $LINT_NOTE}"
    fi
fi
item_finish

# ===========================================================================
# INFO: canonical numbers for the release notes / README.
# ===========================================================================
if [ "${P1_SKIP_NUMBERS:-0}" != "1" ]; then
    printf '\n%s--- INFO: canonical numbers ---%s\n' "$C_BOLD" "$C_RESET"
    NUMBERS_OUT="$WORK/numbers.txt"
    if [ -f "$ROOT_DIR/test/report_numbers.sh" ]; then
        # Reuse item E's --agent-help cache instead of re-probing every command.
        P1_BIN="$BIN" P1_AGENTHELP_TSV="$WORK/agenthelp.tsv" \
            bash "$ROOT_DIR/test/report_numbers.sh" > "$NUMBERS_OUT" 2>&1 || true
        cat "$NUMBERS_OUT"
        # Structural counts are stable for this round; a mismatch means the
        # command tree changed and release notes numbers need re-checking.
        # Test counts (packages/files/funcs) are deliberately NOT hardcoded:
        # they move whenever tests are added.
        for pair in \
            "reference_unique_rows:177" \
            "command_paths:177" \
            "leaf_commands:149" \
            "matrix_rows:157" \
            "agent_help_full_docs:137"; do
            key=${pair%%:*}
            want=${pair#*:}
            got=$(awk -v k="$key" '$1 == k { print $3; exit }' "$NUMBERS_OUT")
            if [ -n "$got" ] && [ "$got" != "$want" ]; then
                printf 'WARN canonical drift: %s expected %s, got %s\n' "$key" "$want" "$got"
            fi
        done
    else
        printf '(test/report_numbers.sh not found — live counts unavailable)\n'
    fi
fi

# ===========================================================================
# Summary
# ===========================================================================
printf '\n%s=== summary ===%s\n' "$C_BOLD" "$C_RESET"
if [ -s "$SUMMARY" ]; then
    while IFS=$'\t' read -r letter status note; do
        color=$C_RESET
        [ "$status" = "FAIL" ] && color=$C_RED
        [ "$status" = "PASS" ] && color=$C_GREEN
        [ "$status" = "SKIP" ] && color=$C_YELLOW
        printf '%s%-4s %-4s%s %s\n' "$color" "$letter" "$status" "$C_RESET" "$note"
    done < "$SUMMARY"
fi
printf '\nPASS=%d FAIL=%d SKIP=%d  (artifacts: %s)\n' "$PASS_N" "$FAIL_N" "$SKIP_N" "$WORK"
printf 'reproduce: bash test/p1-acceptance.sh   (or --no-build with P1_BIN=...)\n'
if [ "$FAIL_N" -gt 0 ]; then
    exit 1
fi
exit 0
