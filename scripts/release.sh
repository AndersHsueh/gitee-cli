#!/usr/bin/env bash
#
# scripts/release.sh — build + package + checksum, and (optionally) publish
# a gitee-cli release to Gitee.
#
# Usage:
#   bash scripts/release.sh                     # build/package the current tag
#   VERSION=0.13.1 bash scripts/release.sh      # explicit version
#   OUT_DIR=/tmp/release bash scripts/release.sh
#   GITEE_TOKEN=xxx bash scripts/release.sh --publish   # also create release + upload assets
#
# Output in $OUT_DIR (default: dist/):
#   gitee-<goos>-<goarch>[.exe]              raw binaries
#   gitee-<goos>-<goarch>[.exe].tar.gz/.zip  archives incl. LICENSE
#   SHA256SUMS.txt
#
# Publishing is the only networked step and is opt-in (--publish); it is
# implemented in scripts/publish-release.sh so packaging stays offline-safe.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

OUT_DIR="${OUT_DIR:-dist}"
PUBLISH=0
for arg in "$@"; do
  case "$arg" in
    --publish) PUBLISH=1 ;;
    *) echo "ERROR: unknown argument: $arg" >&2; exit 1 ;;
  esac
done

# Prefer the exact tag we are standing on; fall back to describe for local builds.
if [ -z "${VERSION:-}" ]; then
  VERSION="$(git describe --tags --exact-match 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || true)"
  VERSION="${VERSION#v}"
fi

if [ -z "${VERSION:-}" ]; then
  echo "ERROR: cannot determine version. Set VERSION=0.13.1 or run from a tagged commit." >&2
  exit 1
fi

if [ "$PUBLISH" -eq 1 ] && [ -z "${GITEE_TOKEN:-}" ]; then
  echo "ERROR: --publish needs GITEE_TOKEN (a Gitee personal access token with repo write access)." >&2
  exit 1
fi

echo "=========================================="
echo " gitee-cli release ${VERSION}"
echo "=========================================="

# 1. Clean previous artifacts so re-runs cannot package stale binaries.
mkdir -p "$OUT_DIR"
rm -f "$OUT_DIR"/gitee-* "$OUT_DIR"/SHA256SUMS.txt "$OUT_DIR"/LICENSE

# 2. Build all platform binaries (VERSION is forwarded so ldflags stamp it).
echo
echo "[1/3] Building binaries..."
VERSION="$VERSION" OUT_DIR="$OUT_DIR" bash scripts/build-release.sh

# 3. Package archives. LICENSE is copied in (clean name) so every archive is
#    self-contained; it is removed again before checksums.
echo
echo "[2/3] Packaging archives..."
cd "$OUT_DIR"
cp "$ROOT_DIR/LICENSE" ./LICENSE

shopt -s nullglob
binaries=( gitee-* )
shopt -u nullglob

packaged=0
for bin in "${binaries[@]}"; do
  case "$bin" in
    *.tar.gz|*.zip) continue ;;
  esac
  case "$bin" in
    *windows*)
      zip -q "${bin}.zip" "$bin" LICENSE
      rm -f "$bin"
      echo "  ✓ ${bin}.zip"
      ;;
    *)
      tar -czf "${bin}.tar.gz" "$bin" LICENSE
      rm -f "$bin"
      echo "  ✓ ${bin}.tar.gz"
      ;;
  esac
  packaged=$((packaged + 1))
done
rm -f ./LICENSE
cd "$ROOT_DIR"

if [ "$packaged" -eq 0 ]; then
  echo "ERROR: no binaries were packaged — build step produced nothing in ${OUT_DIR}/." >&2
  exit 1
fi

# 4. Checksums (portable: sha256sum on Linux, shasum on macOS).
echo
echo "[3/3] Generating checksums..."
cd "$OUT_DIR"
shopt -s nullglob
archives=( *.tar.gz *.zip )
shopt -u nullglob
if [ "${#archives[@]}" -eq 0 ]; then
  echo "ERROR: no archives found for checksums." >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${archives[@]}" > SHA256SUMS.txt
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 "${archives[@]}" > SHA256SUMS.txt
else
  echo "ERROR: neither sha256sum nor shasum found." >&2
  exit 1
fi
cd "$ROOT_DIR"

echo
echo "=========================================="
echo " Artifacts ready in ${OUT_DIR}/"
echo "=========================================="
ls -lh "$OUT_DIR"/

# 5. Optional publish.
if [ "$PUBLISH" -eq 1 ]; then
  echo
  VERSION="$VERSION" OUT_DIR="$OUT_DIR" bash scripts/publish-release.sh
else
  echo
  echo "Next: publish with"
  echo "  GITEE_TOKEN=<token> VERSION=${VERSION} bash scripts/release.sh --publish"
fi
