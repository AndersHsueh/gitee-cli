#!/usr/bin/env bash
#
# scripts/ci.sh — the single quality gate for gitee-cli.
#
# GitHub Actions (.github/workflows/ci.yml) and Gitee Go (.workflow/ci.yml)
# both call this script, so "the gate" has exactly one definition.
#
#   bash scripts/ci.sh          # full gate (used by CI)
#   bash scripts/ci.sh --fast   # skip cross-compile + govulncheck (local loop)
#
# golangci-lint is optional: it is used when present on PATH (CI installs it
# before calling this script) and skipped with a notice otherwise.
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
# 1. Formatting — gofmt is the formatting contract (no extra toolchain).
# ---------------------------------------------------------------------------
step "gofmt"
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    go_files="$(git ls-files '*.go')"
else
    go_files="$(find . -name '*.go' -not -path './.git/*')"
fi
unformatted="$(gofmt -l $go_files | sed '/^$/d')"
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

step "golangci-lint"
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run ./...
else
    echo "golangci-lint not installed — skipping (CI installs it before this script)"
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
# 4. Cross-compile the shippable platforms (catches platform-specific code)
# ---------------------------------------------------------------------------
if [ "$FAST" -eq 0 ]; then
    step "cross-compile"
    out="$(mktemp -d)"
    trap 'rm -rf "$out"' EXIT
    GOOS=linux GOARCH=amd64 go build -o "$out/gitee-linux-amd64" ./cmd/gitee
    GOOS=windows GOARCH=amd64 go build -o "$out/gitee-windows-amd64.exe" ./cmd/gitee
    GOOS=darwin GOARCH=arm64 go build -o "$out/gitee-darwin-arm64" ./cmd/gitee
    echo "linux/amd64, windows/amd64, darwin/arm64 build OK"
fi

# ---------------------------------------------------------------------------
# 5. Vulnerability scan — pinned via the go.mod `tool` directive so it always
#    builds with the project toolchain (no @latest toolchain downgrades).
# ---------------------------------------------------------------------------
if [ "$FAST" -eq 0 ]; then
    step "govulncheck"
    go tool govulncheck ./...
fi

printf '\nAll checks passed.\n'
