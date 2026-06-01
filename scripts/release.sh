#!/usr/bin/env bash
#
# scripts/release.sh — full release workflow: build + package + checksums.
#
# This is the one-shot script you run when cutting a new release.
# It calls build-release.sh to compile, then wraps each binary in a
# platform-appropriate archive (tar.gz for Linux/macOS, zip for Windows),
# and finally generates a SHA256SUMS.txt file.
#
# Usage:
#   bash scripts/release.sh                          # version from git describe
#   VERSION=v0.4.2 bash scripts/release.sh          # explicit version
#   OUT_DIR=/tmp/release bash scripts/release.sh    # custom output
#
# After this script finishes:
#   1. Go to https://gitee.com/<owner>/<repo>/releases
#   2. Click "新建发布" / Create Release for the matching tag
#   3. Drag every file in $OUT_DIR/ into the upload area
#   4. Paste release notes (CHANGELOG-style)
#   5. Publish
#
# For the complete agent-targeted workflow, see release-guide.md
# (which is gitignored — local-only documentation for the agent fleet).

set -euo pipefail

OUT_DIR="${OUT_DIR:-dist}"
BIN_NAME="gitee"

echo "=========================================="
echo " gitee-cli release workflow"
echo "=========================================="
echo

# Step 1: build all platform binaries
echo "[1/3] Building binaries..."
bash scripts/build-release.sh
echo

# Step 2: package each binary
echo "[2/3] Packaging archives..."
cd "$OUT_DIR"

# Stale archives from a previous run would be re-packaged (gitee-linux-amd64.tar.gz
# -> gitee-linux-amd64.tar.gz.tar.gz). Skip anything that already has an archive
# extension, so re-running the script is idempotent.
shopt -s nullglob
binaries=( ${BIN_NAME}-* )
shopt -u nullglob

if [ ${#binaries[@]} -eq 0 ]; then
  echo "  (no binaries to package — dist/ was already packaged?)" >&2
  exit 1
fi

# Include LICENSE in every archive so the binary is self-contained.
# Copy LICENSE into dist/ with a clean name (avoids ".." in the archive
# path, which breaks some tar implementations and is a small security smell).
LICENSE_ARGS=()
if [ -f ../LICENSE ]; then
  cp ../LICENSE ./LICENSE
  LICENSE_ARGS=(./LICENSE)
fi

for bin in "${binaries[@]}"; do
  case "$bin" in
    *.tar.gz|*.zip)
      # Skip — already an archive from a prior run.
      continue
      ;;
    *windows*)
      # zip: just the .exe
      zip -q "${bin}.zip" "$bin" "${LICENSE_ARGS[@]}"
      rm "$bin"
      echo "  ✓ ${bin}.zip"
      ;;
    *)
      # tar.gz: binary + LICENSE
      tar -czf "${bin}.tar.gz" "$bin" "${LICENSE_ARGS[@]}"
      rm "$bin"
      echo "  ✓ ${bin}.tar.gz"
      ;;
  esac
done

# Clean up the LICENSE copy (it's not part of the release, just a packaging helper)
[ -f ./LICENSE ] && rm ./LICENSE

cd - >/dev/null
echo

# Step 3: checksums
echo "[3/3] Generating checksums..."
cd "$OUT_DIR"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum *.tar.gz *.zip 2>/dev/null > SHA256SUMS.txt
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 *.tar.gz *.zip 2>/dev/null > SHA256SUMS.txt
else
  echo "ERROR: neither sha256sum nor shasum found." >&2
  exit 1
fi
cd - >/dev/null

echo
echo "=========================================="
echo " Release artifacts ready in ${OUT_DIR}/"
echo "=========================================="
ls -lh "$OUT_DIR/"
echo
echo "Next: upload these to https://gitee.com/<owner>/<repo>/releases"
echo "      See release-guide.md for the full workflow."
