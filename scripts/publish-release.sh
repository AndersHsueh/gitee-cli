#!/usr/bin/env bash
#
# scripts/publish-release.sh — create a Gitee release for the current tag and
# upload every artifact produced by scripts/release.sh.
#
# Usage:
#   GITEE_TOKEN=xxx VERSION=0.13.1 bash scripts/publish-release.sh
#   GITEE_TOKEN=xxx VERSION=0.13.1 OUT_DIR=dist bash scripts/publish-release.sh
#
# Idempotent enough for a re-run: if a release for the tag already exists we
# reuse it and re-upload assets (Gitee replaces same-named attachments).
#
# Requires: curl, python3 (JSON parsing only). Token needs repo write access.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

: "${GITEE_TOKEN:?GITEE_TOKEN is required (Gitee personal access token with repo write)}"
OUT_DIR="${OUT_DIR:-dist}"
TAG="v${VERSION:?VERSION is required (e.g. VERSION=0.13.1)}"

if [ ! -d "$OUT_DIR" ]; then
  echo "ERROR: ${OUT_DIR}/ not found — run scripts/release.sh first." >&2
  exit 1
fi

# Derive owner/repo from the origin remote (ssh or https form).
REMOTE="$(git remote get-url origin)"
case "$REMOTE" in
  git@gitee.com:*/*.git)  SLUG="${REMOTE#git@gitee.com:}"; SLUG="${SLUG%.git}" ;;
  https://gitee.com/*/*.git) SLUG="${REMOTE#https://gitee.com/}"; SLUG="${SLUG%.git}" ;;
  https://gitee.com/*/*)  SLUG="${REMOTE#https://gitee.com/}" ;;
  *) echo "ERROR: cannot derive owner/repo from origin remote: ${REMOTE}" >&2; exit 1 ;;
esac
OWNER="${SLUG%%/*}"
REPO="${SLUG#*/}"
API="https://gitee.com/api/v5/repos/${OWNER}/${REPO}"

echo "==> Publishing ${TAG} to ${OWNER}/${REPO}"

# Release notes: use release-notes/<tag>.md when present, else a short default.
NOTES_FILE="release-notes/${TAG}.md"
if [ -f "$NOTES_FILE" ]; then
  BODY="$(cat "$NOTES_FILE")"
else
  BODY="$(printf 'Release %s\n\nSee git log for the full changelog.' "$TAG")"
  echo "    (no ${NOTES_FILE} — using a minimal release body)"
fi

TARGET="$(git rev-parse HEAD)"

# Create the release. If it already exists (422/400), look it up instead.
echo "==> Creating release ${TAG}"
CREATE_JSON="$(python3 - "$TAG" "$TARGET" "$BODY" <<'PY'
import json, sys
tag, target, body = sys.argv[1], sys.argv[2], sys.argv[3]
print(json.dumps({
    "tag_name": tag,
    "target_commitish": target,
    "name": tag,
    "body": body,
    "prerelease": False,
}))
PY
)"

RESP="$(curl -sS -X POST "${API}/releases" \
  -H "Content-Type: application/json" \
  -H "Authorization: token ${GITEE_TOKEN}" \
  -d "$CREATE_JSON")"

RELEASE_ID="$(printf '%s' "$RESP" | python3 -c 'import json,sys
try:
    d=json.load(sys.stdin)
except Exception:
    d={}
print(d.get("id","") if isinstance(d,dict) else "")')"

if [ -z "$RELEASE_ID" ]; then
  echo "    create did not return an id; looking up existing release for ${TAG}"
  RESP="$(curl -sS -H "Authorization: token ${GITEE_TOKEN}" "${API}/releases/tags/${TAG}")"
  RELEASE_ID="$(printf '%s' "$RESP" | python3 -c 'import json,sys
try:
    d=json.load(sys.stdin)
except Exception:
    d={}
print(d.get("id","") if isinstance(d,dict) else "")')"
fi

if [ -z "$RELEASE_ID" ]; then
  echo "ERROR: could not create or find a release for ${TAG}." >&2
  echo "response: ${RESP}" >&2
  exit 1
fi
echo "    release id: ${RELEASE_ID}"

# Upload every archive + checksums file.
echo "==> Uploading assets"
shopt -s nullglob
assets=( "$OUT_DIR"/*.tar.gz "$OUT_DIR"/*.zip "$OUT_DIR"/SHA256SUMS.txt )
shopt -u nullglob
if [ "${#assets[@]}" -eq 0 ]; then
  echo "ERROR: no assets found in ${OUT_DIR}/." >&2
  exit 1
fi

for asset in "${assets[@]}"; do
  printf '    %-40s ' "$(basename "$asset")"
  UP="$(curl -sS -X POST "${API}/releases/${RELEASE_ID}/attach_files" \
    -H "Authorization: token ${GITEE_TOKEN}" \
    -F "file=@${asset}")"
  UP_NAME="$(printf '%s' "$UP" | python3 -c 'import json,sys
try:
    d=json.load(sys.stdin)
except Exception:
    d={}
print(d.get("name","") if isinstance(d,dict) else "")')"
  if [ -n "$UP_NAME" ]; then
    echo "✓ ${UP_NAME}"
  else
    echo "FAILED"
    echo "response: ${UP}" >&2
    exit 1
  fi
done

echo
echo "==> Release ${TAG} published: https://gitee.com/${OWNER}/${REPO}/releases/tag/${TAG}"
