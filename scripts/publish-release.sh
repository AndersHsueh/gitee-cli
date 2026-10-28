#!/usr/bin/env bash
#
# scripts/publish-release.sh — create the Gitee release for a git tag and
# upload every artifact produced by scripts/release.sh.
#
# The network calls go through the gitee CLI itself (release create / view /
# upload) rather than hand-rolled HTTP, so publishing uses the same auth
# resolution, retries and endpoint builders as the shipped binary.
#
# Usage:
#   GITEE_TOKEN=xxx TAG=v0.13.1 bash scripts/publish-release.sh
#   GITEE_TOKEN=xxx VERSION=0.13.1 OUT_DIR=dist bash scripts/publish-release.sh
#
# Idempotent: an existing release is reused, and assets already attached
# (matched by name) are skipped, so a failed run can be re-run cheaply.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

: "${GITEE_TOKEN:?GITEE_TOKEN is required (Gitee personal access token with repo write)}"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/dist}"

if [ -z "${TAG:-}" ]; then
  : "${VERSION:?set TAG=vX.Y.Z or VERSION=X.Y.Z}"
  TAG="v${VERSION#v}"
fi

if [ ! -d "$OUT_DIR" ]; then
  echo "ERROR: ${OUT_DIR}/ not found — run scripts/release.sh first." >&2
  exit 1
fi
if [ ! -f "$OUT_DIR/SHA256SUMS.txt" ]; then
  echo "ERROR: ${OUT_DIR}/SHA256SUMS.txt missing — run scripts/release.sh first." >&2
  exit 1
fi
if ! git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null; then
  echo "ERROR: git tag ${TAG} does not exist locally — tag the release commit first." >&2
  exit 1
fi
TARGET="$(git rev-list -n1 "$TAG")"

# The release commands take --repo; derive owner/repo from the origin remote
# (ssh and https forms).
REMOTE="$(git remote get-url origin 2>/dev/null || true)"
if [ -z "$REMOTE" ]; then
  echo "ERROR: no 'origin' remote — cannot determine the target repository." >&2
  exit 1
fi
case "$REMOTE" in
  git@gitee.com:*)         SLUG="${REMOTE#git@gitee.com:}" ;;
  ssh://git@gitee.com/*)   SLUG="${REMOTE#ssh://git@gitee.com/}" ;;
  https://gitee.com/*)     SLUG="${REMOTE#https://gitee.com/}" ;;
  *)
    echo "ERROR: origin is not a gitee.com remote: ${REMOTE}" >&2
    exit 1
    ;;
esac
SLUG="${SLUG%.git}"
case "$SLUG" in
  */*) ;;
  *)
    echo "ERROR: cannot derive owner/repo from origin remote: ${REMOTE}" >&2
    exit 1
    ;;
esac

NOTES_FILE="release-notes/${TAG}.md"
if [ -f "$NOTES_FILE" ]; then
  BODY="$(cat "$NOTES_FILE")"
else
  BODY="Release ${TAG}"
  echo "==> No ${NOTES_FILE}, using a minimal release body"
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# Throwaway host binary so publishing exercises the real CLI code path.
echo "==> Building publishing binary"
go build -o "$WORK/gitee" ./cmd/gitee
BIN="$WORK/gitee"

echo "==> Ensuring release ${TAG} on ${SLUG} (target ${TARGET})"
if "$BIN" release create --repo "$SLUG" --tag "$TAG" --name "$TAG" --target "$TARGET" --body "$BODY" >/dev/null 2>"$WORK/create.err"; then
  echo "    created"
else
  echo "    create did not succeed — assuming the release already exists"
  sed 's/^/    /' "$WORK/create.err" | tail -2
fi

# Gitee indexing can lag right after a create, so retry the read.
echo "==> Resolving release"
ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if "$BIN" release view --repo "$SLUG" --tag "$TAG" -o json > "$WORK/release.json" 2>"$WORK/view.err"; then
    ready=1
    break
  fi
  sleep 2
done
if [ "$ready" -ne 1 ]; then
  echo "ERROR: release ${TAG} is not readable after 20s." >&2
  if [ -s "$WORK/create.err" ]; then
    echo "create error:" >&2
    sed 's/^/    /' "$WORK/create.err" >&2
  fi
  if [ -s "$WORK/view.err" ]; then
    echo "view error:" >&2
    sed 's/^/    /' "$WORK/view.err" >&2
  fi
  exit 1
fi

# One parse gives us both the already-attached asset names and the URL.
python3 - "$WORK/release.json" "$WORK/assets.txt" "$WORK/url.txt" <<'PY'
import json, sys
src, assets_path, url_path = sys.argv[1], sys.argv[2], sys.argv[3]
with open(src, encoding="utf-8") as fh:
    release = json.load(fh)
with open(assets_path, "w", encoding="utf-8") as fh:
    for asset in release.get("assets") or []:
        fh.write(asset.get("name", "") + "\n")
with open(url_path, "w", encoding="utf-8") as fh:
    fh.write(release.get("html_url", ""))
PY
existing="$(cat "$WORK/assets.txt")"

shopt -s nullglob
assets=( "$OUT_DIR"/*.tar.gz "$OUT_DIR"/*.zip "$OUT_DIR"/SHA256SUMS.txt )
shopt -u nullglob
if [ "${#assets[@]}" -eq 0 ]; then
  echo "ERROR: no assets found in ${OUT_DIR}/." >&2
  exit 1
fi

echo "==> Uploading assets"
for asset in "${assets[@]}"; do
  name="$(basename "$asset")"
  if printf '%s\n' "$existing" | grep -qxF "$name"; then
    echo "    ${name} — already attached, skipping"
    continue
  fi
  printf '    %-42s ' "$name"
  uploaded=0
  for _ in 1 2 3 4 5; do
    if "$BIN" release upload --repo "$SLUG" "$TAG" --file "$asset" >/dev/null 2>"$WORK/upload.err"; then
      uploaded=1
      break
    fi
    sleep 2
  done
  if [ "$uploaded" -eq 1 ]; then
    echo "✓"
  else
    echo "FAILED"
    sed 's/^/    /' "$WORK/upload.err" >&2
    exit 1
  fi
done

echo
echo "==> Release ${TAG} published"
echo "    $(cat "$WORK/url.txt")"
