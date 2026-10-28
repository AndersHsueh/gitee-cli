#!/usr/bin/env bash
#
# scripts/release.sh — build + package + checksum, and (optionally) publish
# a gitee-cli release.
#
# Usage:
#   bash scripts/release.sh                      # build/package the current tag
#   VERSION=0.13.1 bash scripts/release.sh       # explicit version
#   OUT_DIR=/tmp/release bash scripts/release.sh
#   GITEE_TOKEN=xxx bash scripts/release.sh --publish   # also create release + upload assets
#
# Output in $OUT_DIR (default: dist/):
#   gitee-<goos>-<goarch>[.exe].tar.gz/.zip  archives incl. LICENSE
#   SHA256SUMS.txt
#
# Builds and packaging happen in a temp staging dir; $OUT_DIR is only touched
# after everything succeeded, so a failed build never leaves a half-release
# (or deletes the previous one).
#
# Publishing is the only networked step and is opt-in (--publish); it is
# implemented in scripts/publish-release.sh (which dogfoods the gitee CLI).
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

# Refuse an output directory that would make the cleanup below delete repo files.
OUT_ABS="$(mkdir -p "$OUT_DIR" && cd "$OUT_DIR" && pwd)"
if [ "$OUT_ABS" = "$ROOT_DIR" ]; then
  echo "ERROR: OUT_DIR must not be the repository root." >&2
  exit 1
fi
OUT_DIR="$OUT_ABS"

# Prefer the exact tag we are standing on; fall back to describe for local builds.
if [ -z "${VERSION:-}" ]; then
  VERSION="$(git describe --tags --exact-match 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || true)"
  VERSION="${VERSION#v}"
fi
if [ -z "${VERSION:-}" ]; then
  echo "ERROR: cannot determine version. Set VERSION=0.13.1 or run from a tagged commit." >&2
  exit 1
fi
TAG="v${VERSION#v}"

# Publishing a non-tag (e.g. `0.13.1-3-gabc-dirty`) would create a release on a
# branch name. Require the tag to exist before building anything.
if [ "$PUBLISH" -eq 1 ]; then
  if ! git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null; then
    echo "ERROR: --publish requires a tagged commit; ${TAG} does not exist locally." >&2
    echo "       Tag the release commit first: git tag -a ${TAG} -m ${TAG}" >&2
    exit 1
  fi
  if [ -z "${GITEE_TOKEN:-}" ]; then
    echo "ERROR: --publish needs GITEE_TOKEN (a Gitee personal access token with repo write access)." >&2
    exit 1
  fi
fi

echo "=========================================="
echo " gitee-cli release ${VERSION} (${TAG})"
echo "=========================================="

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
BUILD_DIR="$STAGE/build"
mkdir -p "$BUILD_DIR"

# 1. Build all platform binaries into the staging dir (VERSION is forwarded so
#    ldflags stamp it).
echo
echo "[1/3] Building binaries..."
VERSION="$VERSION" OUT_DIR="$BUILD_DIR" bash scripts/build-release.sh

# Smoke-check the host binary before packaging: a wrong VERSION or an ldflags
# regression should fail now, not after the upload.
host="$(go env GOOS)-$(go env GOARCH)"
if [ -x "$BUILD_DIR/gitee-${host}" ]; then
  got="$("$BUILD_DIR/gitee-${host}" version 2>&1)"
  case "$got" in
    *" $VERSION") echo "    smoke check: ${got}" ;;
    *) echo "ERROR: gitee-${host} reports '${got}', expected version ${VERSION}." >&2; exit 1 ;;
  esac
fi

# 2. Package archives. LICENSE is staged separately so no output directory is
#    ever polluted and every archive is self-contained.
echo
echo "[2/3] Packaging archives..."
cp "$ROOT_DIR/LICENSE" "$STAGE/LICENSE"
cp "$STAGE/LICENSE" "$BUILD_DIR/LICENSE"

cd "$BUILD_DIR"
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
      zip -q -j "${bin}.zip" "$bin" "$STAGE/LICENSE"
      rm -f "$bin"
      echo "  ✓ ${bin}.zip"
      ;;
    *)
      tar -czf "${bin}.tar.gz" "$bin" -C "$STAGE" LICENSE
      rm -f "$bin"
      echo "  ✓ ${bin}.tar.gz"
      ;;
  esac
  packaged=$((packaged + 1))
done
rm -f "$BUILD_DIR/LICENSE"
cd "$ROOT_DIR"

if [ "$packaged" -eq 0 ]; then
  echo "ERROR: no binaries were packaged — build step produced nothing." >&2
  exit 1
fi

# 3. Checksums (portable: sha256sum on Linux, shasum on macOS).
echo
echo "[3/3] Generating checksums..."
cd "$BUILD_DIR"
shopt -s nullglob
archives=( *.tar.gz *.zip )
shopt -u nullglob
if [ "${#archives[@]}" -eq 0 ]; then
  echo "ERROR: no archives found for checksums." >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${archives[@]}" > SHA256SUMS.txt
else
  shasum -a 256 "${archives[@]}" > SHA256SUMS.txt
fi
cd "$ROOT_DIR"

# 4. Swap the freshly built artifacts into OUT_DIR (previous run is replaced
#    only now that the new one is complete).
rm -f "$OUT_DIR"/gitee-* "$OUT_DIR"/SHA256SUMS.txt
mv "$BUILD_DIR"/* "$OUT_DIR"/

echo
echo "=========================================="
echo " Artifacts ready in ${OUT_DIR}/"
echo "=========================================="
ls -lh "$OUT_DIR"/

# 5. Optional publish.
if [ "$PUBLISH" -eq 1 ]; then
  echo
  VERSION="$VERSION" TAG="$TAG" OUT_DIR="$OUT_DIR" bash scripts/publish-release.sh
else
  echo
  echo "Next: publish with"
  echo "  GITEE_TOKEN=<token> VERSION=${VERSION} bash scripts/release.sh --publish"
fi
