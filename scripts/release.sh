#!/usr/bin/env bash
# 纯 Go 交叉编译 + 上传 GitLab Release（不依赖 goreleaser Docker 镜像）
#
# 用法：
#   bash scripts/release.sh v0.1.0
#
# 需要：GITLAB_TOKEN（api）、go、tar/zip、curl、python3
set -euo pipefail

_load_env_file() {
  local f="$1"
  [[ -f "$f" ]] || return 1
  set -a
  # shellcheck disable=SC1090
  source "$f"
  set +a
}

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

[[ -z "${GITLAB_TOKEN:-}" ]] && _load_env_file "$ROOT/.env" || true
[[ -z "${GITLAB_TOKEN:-}" ]] && _load_env_file "$HOME/.leangoo-cli/.env" || true

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  VERSION="$(git describe --tags --exact-match 2>/dev/null || true)"
fi
if [[ -z "$VERSION" ]]; then
  echo "用法: bash scripts/release.sh v0.1.0" >&2
  exit 1
fi
if [[ -z "${GITLAB_TOKEN:-}" ]]; then
  echo "缺少 GITLAB_TOKEN" >&2
  exit 1
fi

GITLAB_HOST="${GITLAB_HOST:-gitlab.deepglint.com}"
GITLAB_PROJECT="${GITLAB_PROJECT:-liuyuan/leangoo-cli}"
API="https://${GITLAB_HOST}/api/v4"
PROJECT_ENC="$(printf '%s' "$GITLAB_PROJECT" | sed 's|/|%2F|g')"
AUTH=(-H "PRIVATE-TOKEN: ${GITLAB_TOKEN}")
VER_NOPREFIX="${VERSION#v}"

DIST="$ROOT/dist"
rm -rf "$DIST"
mkdir -p "$DIST"

targets=(
  "linux|amd64|Linux_x86_64"
  "linux|arm64|Linux_arm64"
  "darwin|amd64|Darwin_x86_64"
  "darwin|arm64|Darwin_arm64"
  "windows|amd64|Windows_x86_64"
)

echo "==> 构建 ${VERSION}"
for t in "${targets[@]}"; do
  IFS='|' read -r goos goarch label <<<"$t"
  outdir="$DIST/leangoo_${label}"
  mkdir -p "$outdir/skills"
  bin="leangoo"
  [[ "$goos" == "windows" ]] && bin="leangoo.exe"
  echo "  - ${goos}/${goarch}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
    -ldflags "-s -w -X github.com/deepglint/leangoo-cli/internal/cmd.Version=${VER_NOPREFIX}" \
    -o "$outdir/$bin" ./cmd/leangoo
  cp README.md "$outdir/"
  cp -R skills/leangoo-* "$outdir/skills/"
  if [[ "$goos" == "windows" ]]; then
    (cd "$DIST" && zip -qr "leangoo_${label}.zip" "leangoo_${label}")
  else
    (cd "$DIST" && tar -czf "leangoo_${label}.tar.gz" "leangoo_${label}")
  fi
done

(
  cd "$DIST"
  shasum -a 256 leangoo_*.tar.gz leangoo_*.zip | tee checksums.txt
)

echo "==> 创建 / 更新 GitLab Release ${VERSION}"
if ! curl -fsSL "${AUTH[@]}" "${API}/projects/${PROJECT_ENC}/releases/${VERSION}" >/dev/null 2>&1; then
  curl -fsSL "${AUTH[@]}" -X POST "${API}/projects/${PROJECT_ENC}/releases" \
    -H "Content-Type: application/json" \
    --data-binary @- <<EOF
{"name":"${VERSION}","tag_name":"${VERSION}","description":"## leangoo ${VERSION}\n\n安装：\`bash scripts/install.sh ${VERSION}\`\n"}
EOF
  echo "  release created"
else
  echo "  release already exists"
fi

upload() {
  local file="$1"
  local name
  name="$(basename "$file")"
  echo "  upload $name"
  curl -fsSL "${AUTH[@]}" \
    --upload-file "$file" \
    "${API}/projects/${PROJECT_ENC}/packages/generic/leangoo/${VERSION}/${name}" >/dev/null

  local pkg_url="${API}/projects/${PROJECT_ENC}/packages/generic/leangoo/${VERSION}/${name}"
  local existing link_id
  existing="$(curl -fsSL "${AUTH[@]}" \
    "${API}/projects/${PROJECT_ENC}/releases/${VERSION}/assets/links" 2>/dev/null || echo '[]')"
  link_id="$(printf '%s' "$existing" | python3 -c "
import json,sys
name=sys.argv[1]
for a in json.load(sys.stdin):
  if a.get('name')==name:
    print(a['id']); break
" "$name" 2>/dev/null || true)"
  if [[ -n "$link_id" ]]; then
    curl -fsSL "${AUTH[@]}" -X DELETE \
      "${API}/projects/${PROJECT_ENC}/releases/${VERSION}/assets/links/${link_id}" >/dev/null || true
  fi
  curl -fsSL "${AUTH[@]}" -X POST \
    "${API}/projects/${PROJECT_ENC}/releases/${VERSION}/assets/links" \
    -H "Content-Type: application/json" \
    --data-binary @- <<EOF
{"name":"${name}","url":"${pkg_url}","link_type":"package"}
EOF
}

echo "==> 上传附件"
for f in "$DIST"/leangoo_*.tar.gz "$DIST"/leangoo_*.zip "$DIST"/checksums.txt; do
  [[ -f "$f" ]] || continue
  upload "$f"
done

echo "==> 完成: https://${GITLAB_HOST}/${GITLAB_PROJECT}/-/releases/${VERSION}"
