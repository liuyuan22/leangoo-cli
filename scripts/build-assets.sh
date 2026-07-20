#!/usr/bin/env bash
# 构建多平台 Release 附件到 dist/（CI 与本地共用）
# 用法: bash scripts/build-assets.sh v0.1.0
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  VERSION="$(git describe --tags --exact-match 2>/dev/null || true)"
fi
if [[ -z "$VERSION" ]]; then
  echo "用法: bash scripts/build-assets.sh v0.1.0" >&2
  exit 1
fi
case "$VERSION" in
  v*) ;;
  *) VERSION="v${VERSION}" ;;
esac
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

echo "==> 产物在 ${DIST}"
