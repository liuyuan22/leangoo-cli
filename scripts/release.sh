#!/usr/bin/env bash
# 本地一行发版：构建多平台包 → 打 tag → 推送 → 创建 GitHub Release 并上传附件
#
#   bash scripts/release.sh v0.1.0
#
# 需要：go、tar、zip、curl、python3
# 认证（任选其一）：
#   - 已安装并登录 gh（推荐：brew install gh && gh auth login）
#   - 或 .env / 环境变量里的 GITHUB_TOKEN（contents:write）
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

[[ -z "${GITHUB_TOKEN:-}" ]] && _load_env_file "$ROOT/.env" || true
[[ -z "${GITHUB_TOKEN:-}" ]] && _load_env_file "$HOME/.leangoo-cli/.env" || true

GITHUB_REPO="${GITHUB_REPO:-liuyuan22/leangoo-cli}"
VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "用法: bash scripts/release.sh v0.1.0" >&2
  exit 1
fi
case "$VERSION" in
  v*) ;;
  *) VERSION="v${VERSION}" ;;
esac
VER_NOPREFIX="${VERSION#v}"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "工作区有未提交改动，请先 commit 再发版。" >&2
  git status --short >&2
  exit 1
fi

use_gh=0
if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
  use_gh=1
elif [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "未检测到 gh 登录，也缺少 GITHUB_TOKEN。" >&2
  echo "请任选其一：" >&2
  echo "  brew install gh && gh auth login" >&2
  echo "  或在 .env 写入 GITHUB_TOKEN=ghp_xxx（contents:write）" >&2
  exit 1
fi

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

echo "==> git tag ${VERSION}"
if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "  tag 已存在，跳过创建"
else
  git tag -a "$VERSION" -m "Release ${VERSION}"
fi

echo "==> 推送 main + ${VERSION}"
git push origin HEAD:main
git push origin "$VERSION"

notes="$(cat <<EOF
## leangoo ${VERSION}

### 安装

\`\`\`bash
curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/scripts/install.sh | bash
# 或指定版本
curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/scripts/install.sh | bash -s -- ${VERSION}
\`\`\`
EOF
)"

assets=()
for f in "$DIST"/leangoo_*.tar.gz "$DIST"/leangoo_*.zip "$DIST"/checksums.txt; do
  [[ -f "$f" ]] && assets+=("$f")
done

echo "==> 创建 GitHub Release"
if [[ "$use_gh" -eq 1 ]]; then
  if gh release view "$VERSION" --repo "$GITHUB_REPO" >/dev/null 2>&1; then
    echo "  Release 已存在，上传/覆盖附件…"
    gh release upload "$VERSION" "${assets[@]}" --repo "$GITHUB_REPO" --clobber
  else
    gh release create "$VERSION" "${assets[@]}" \
      --repo "$GITHUB_REPO" \
      --title "$VERSION" \
      --notes "$notes"
  fi
else
  api="https://api.github.com/repos/${GITHUB_REPO}"
  auth=(-H "Authorization: Bearer ${GITHUB_TOKEN}" -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28")
  rel_json="$(curl -fsSL "${auth[@]}" "${api}/releases/tags/${VERSION}" 2>/dev/null || true)"
  if [[ -z "$rel_json" || "$rel_json" == *"Not Found"* ]]; then
    rel_json="$(curl -fsSL "${auth[@]}" -X POST "${api}/releases" \
      -H "Content-Type: application/json" \
      -d "$(python3 -c "import json,sys; print(json.dumps({'tag_name':sys.argv[1],'name':sys.argv[1],'body':sys.argv[2]}))" "$VERSION" "$notes")")"
  fi
  upload_url="$(printf '%s' "$rel_json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["upload_url"].split("{")[0])')"
  release_id="$(printf '%s' "$rel_json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
  # 删除同名旧附件后重传
  existing="$(curl -fsSL "${auth[@]}" "${api}/releases/${release_id}/assets")"
  for f in "${assets[@]}"; do
    name="$(basename "$f")"
    old_id="$(printf '%s' "$existing" | python3 -c "
import json,sys
name=sys.argv[1]
for a in json.load(sys.stdin):
  if a.get('name')==name:
    print(a['id']); break
" "$name" 2>/dev/null || true)"
    if [[ -n "$old_id" ]]; then
      curl -fsSL "${auth[@]}" -X DELETE "${api}/releases/assets/${old_id}" >/dev/null
    fi
    echo "  upload $name"
    curl -fsSL "${auth[@]}" \
      -H "Content-Type: application/octet-stream" \
      --data-binary @"$f" \
      "${upload_url}?name=$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))' "$name")" >/dev/null
  done
fi

echo "==> 完成: https://github.com/${GITHUB_REPO}/releases/tag/${VERSION}"
