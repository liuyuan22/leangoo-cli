#!/usr/bin/env bash
# 安装 leangoo CLI + Agent Skills（GitHub Releases，公开仓库无需 token）
#
#   curl -fsSL https://raw.githubusercontent.com/liuyuan22/leangoo-cli/main/scripts/install.sh | bash
#   curl -fsSL ... | bash -s -- v0.1.0
#
# 或仓库内：bash scripts/install.sh [version]
#
# 环境变量：
#   GITHUB_REPO     默认 liuyuan22/leangoo-cli
#   INSTALL_DIR     默认 ~/.local/bin
#   INSTALL_SKILLS  默认 1
set -euo pipefail

GITHUB_REPO="${GITHUB_REPO:-liuyuan22/leangoo-cli}"
API="https://api.github.com/repos/${GITHUB_REPO}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
INSTALL_SKILLS="${INSTALL_SKILLS:-1}"
VERSION="${1:-}"

echo "==> GitHub: https://github.com/${GITHUB_REPO}"

os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Darwin) goos="Darwin" ;;
  Linux)  goos="Linux" ;;
  MINGW*|MSYS*|CYGWIN*) goos="Windows" ;;
  *) echo "不支持的操作系统: $os" >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) goarch="x86_64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *) echo "不支持的架构: $arch" >&2; exit 1 ;;
esac

if [[ -z "$VERSION" || "$VERSION" == "latest" ]]; then
  echo "==> 查询最新 Release…"
  VERSION="$(curl -fsSL "${API}/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
  if [[ -z "$VERSION" ]]; then
    echo "无法解析最新 tag。可手动指定：bash install.sh v0.1.0" >&2
    exit 1
  fi
fi
echo "==> 版本: $VERSION"

asset_name="leangoo_${goos}_${goarch}.tar.gz"
if [[ "$goos" == "Windows" ]]; then
  asset_name="leangoo_${goos}_${goarch}.zip"
fi
echo "==> 资源: $asset_name"

rel_json="$(curl -fsSL "${API}/releases/tags/${VERSION}")"
asset_url="$(printf '%s' "$rel_json" | python3 -c "
import json,sys
name=sys.argv[1]
d=json.load(sys.stdin)
for a in d.get('assets') or []:
    if a.get('name')==name:
        print(a.get('browser_download_url') or ''); sys.exit(0)
sys.exit(1)
" "$asset_name" 2>/dev/null || true)"

if [[ -z "$asset_url" ]]; then
  echo "在 Release ${VERSION} 中未找到 ${asset_name}" >&2
  printf '%s' "$rel_json" | python3 -c '
import json,sys
d=json.load(sys.stdin)
for a in d.get("assets") or []:
    print(" -", a.get("name"))
' >&2 || true
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
archive="$tmpdir/$asset_name"
echo "==> 下载…"
curl -fL -o "$archive" "$asset_url"

echo "==> 解压到 ${INSTALL_DIR}"
mkdir -p "$INSTALL_DIR"
if [[ "$asset_name" == *.zip ]]; then
  unzip -qo "$archive" -d "$tmpdir/out"
else
  mkdir -p "$tmpdir/out"
  tar -xzf "$archive" -C "$tmpdir/out"
fi
bin_path="$(find "$tmpdir/out" -type f \( -name 'leangoo' -o -name 'leangoo.exe' \) | head -1)"
if [[ -z "$bin_path" ]]; then
  echo "压缩包中未找到 leangoo 二进制" >&2
  exit 1
fi
install -m 755 "$bin_path" "${INSTALL_DIR}/leangoo"
echo "已安装: ${INSTALL_DIR}/leangoo"
"${INSTALL_DIR}/leangoo" version || true

if [[ "$INSTALL_SKILLS" == "1" ]]; then
  echo "==> 安装 Agent Skills…"
  skills_src=""
  # 归档内可能是 leangoo_Darwin_arm64/skills/...
  if [[ -d "$tmpdir/out/skills" ]]; then
    skills_src="$tmpdir/out/skills"
  else
    found="$(find "$tmpdir/out" -type d -name skills | head -1)"
    [[ -n "$found" ]] && skills_src="$found"
  fi
  if [[ -z "$skills_src" ]]; then
    skills_archive="$tmpdir/src.tgz"
    curl -fL -o "$skills_archive" \
      "https://github.com/${GITHUB_REPO}/archive/refs/tags/${VERSION}.tar.gz"
    mkdir -p "$tmpdir/repo"
    tar -xzf "$skills_archive" -C "$tmpdir/repo" --strip-components=1
    skills_src="$tmpdir/repo/skills"
  fi
  for dest in "$HOME/.cursor/skills" "$HOME/.claude/skills"; do
    mkdir -p "$dest"
    for s in leangoo-shared leangoo-sprint leangoo-story; do
      if [[ -d "$skills_src/$s" ]]; then
        rm -rf "$dest/$s"
        cp -R "$skills_src/$s" "$dest/$s"
        echo "  -> $dest/$s"
      fi
    done
  done
fi

case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo
    echo "提示: 请把 ${INSTALL_DIR} 加入 PATH，例如："
    echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
    ;;
esac

echo "==> 完成。下一步: leangoo auth login"
