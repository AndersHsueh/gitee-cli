#!/usr/bin/env bash
#
# scripts/build-release.sh — cross-compile gitee-cli for 5 platforms.
#
# Output: $OUT_DIR/gitee-<goos>-<goarch>[.exe]  (one binary per platform)
#
# Usage:
#   bash scripts/build-release.sh                  # auto-detect version from git
#   VERSION=v0.4.1 bash scripts/build-release.sh   # explicit version
#   OUT_DIR=/tmp/build bash scripts/build-release.sh  # custom output dir
#
# This script only BUILDS the binaries. It does not package them into
# tar.gz/zip or generate checksums. For the full release workflow
# (build + package + checksums), use scripts/release.sh.
#
# Maintainer note: when adding a new platform, also update:
#   - PLATFORMS list below
#   - README.md "📦 下载" section
#   - release-guide.md "Supported platforms" table

set -euo pipefail

# ----- Configuration ---------------------------------------------------------

# Default version: use `git describe` (works for both tagged and untagged commits).
# Strip the leading 'v' so `-ldflags -X ...=v0.4.1` and `=0.4.1` both work.
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')}"
if [ -z "$VERSION" ]; then
  echo "ERROR: cannot determine version. Set VERSION env var or run inside a git repo with a tag." >&2
  exit 1
fi

OUT_DIR="${OUT_DIR:-dist}"
BIN_NAME="gitee"
LDFLAGS="-X gitee-cli/cmd.Version=${VERSION}"

# Supported platforms. Add a new entry here when you want to ship a new combo.
# Format: "GOOS/GOARCH"
PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

# ----- Sanity checks ---------------------------------------------------------

# Must run from repo root (we use ./cmd/gitee as the package path).
if [ ! -d "./cmd/gitee" ]; then
  echo "ERROR: ./cmd/gitee not found. Run this script from the repo root." >&2
  exit 1
fi

command -v go >/dev/null 2>&1 || { echo "ERROR: go is not in PATH." >&2; exit 1; }

# ----- Build -----------------------------------------------------------------

mkdir -p "$OUT_DIR"

echo "==> Building gitee-cli ${VERSION} for ${#PLATFORMS[@]} platforms"
echo "    Output: ${OUT_DIR}/"
echo

for platform in "${PLATFORMS[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  ext=""
  [ "$GOOS" = "windows" ] && ext=".exe"
  out="${OUT_DIR}/${BIN_NAME}-${GOOS}-${GOARCH}${ext}"

  printf "  %-30s " "${GOOS}/${GOARCH}"
  GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags "$LDFLAGS" -o "$out" ./cmd/gitee
  printf "✓ %s\n" "$out"
done

echo
echo "==> Done. ${#PLATFORMS[@]} binaries in ${OUT_DIR}/"
ls -lh "$OUT_DIR/"
