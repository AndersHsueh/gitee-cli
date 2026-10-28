#!/usr/bin/env bash
#
# scripts/ci.sh — the single quality gate for gitee-cli.
#
# Runs the same checks in GitHub Actions (.github/workflows/ci.yml), in
# Gitee Go (.workflow/ci.yml), and on a developer machine:
#
#   bash scripts/ci.sh          # full gate
#   bash scripts/ci.sh --fast   # skip -race and govulncheck (quick local loop)
#
# The gate is intentionally limited to checks that are green today and
# cheap to run. If you add a check, make sure it is deterministic and
# does not need network access beyond the Go module proxy.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FAST=0
if [ "${1:-}" = "--fast" ]; then
    FAST=1
fi

step() { printf '\n==> %s\n' "$*"; }
fail() { printf '\nERROR: %s\n' "$*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# 1. Formatting — gofmt is the project's formatting contract (no separate
#    gofumpt/goimports toolchain to install).
# ---------------------------------------------------------------------------
step "gofmt"
unformatted="$(gofmt -l . | sed '/^$/d')"
if [ -n "$unformatted" ]; then
    printf '%s\n' "$unformatted" >&2
    fail "the files above need 'gofmt -w' — run: gofmt -w \$(git ls-files '*.go')"
fi
echo "all Go files are gofmt-clean"

# ---------------------------------------------------------------------------
# 2. Static analysis
# ---------------------------------------------------------------------------
step "go vet"
go vet ./...

if command -v golangci-lint >/dev/null 2>&1; then
    step "golangci-lint"
    golangci-lint run ./...
else
    echo "golangci-lint not installed — skipping (CI installs it explicitly)"
fi

# ---------------------------------------------------------------------------
# 3. Build + tests
# ---------------------------------------------------------------------------
step "go build"
go build ./...

step "go test"
if [ "$FAST" -eq 1 ]; then
    go test ./... -count=1
else
    go test ./... -race -count=1
fi

# ---------------------------------------------------------------------------
# 4. Vulnerability scan (source mode: only reachable vulnerabilities fail)
# ---------------------------------------------------------------------------
if [ "$FAST" -eq 0 ]; then
    step "govulncheck"
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...
fi

printf '\nAll checks passed.\n'
