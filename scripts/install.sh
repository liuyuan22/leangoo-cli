#!/usr/bin/env bash
# 安装 leangoo CLI + Agent Skills（GitHub Releases，公开仓库无需 token）
#
# 推荐（jsDelivr，避免 raw CDN 缓存旧脚本）：
#   curl -fsSL https://cdn.jsdelivr.net/gh/liuyuan22/leangoo-cli@main/scripts/install.sh | bash
#   curl -fsSL https://cdn.jsdelivr.net/gh/liuyuan22/leangoo-cli@main/scripts/install.sh | bash -s -- v1.2.0
#
# 备用（GitHub raw，main 分支偶发缓存滞后）：
#   curl -fsSL https://raw.githubusercontent.com/liuyuan22/leangoo-cli/main/scripts/install.sh | bash
#
# 或仓库内：bash scripts/install.sh [version]
#
# 环境变量：
#   GITHUB_REPO     默认 liuyuan22/leangoo-cli
#   INSTALL_DIR     默认 ~/.local/bin
#   INSTALL_SKILLS  默认 1
#   GITHUB_TOKEN / GH_TOKEN  可选；仅在走 API 回退时使用
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

asset_name="leangoo_${goos}_${goarch}.tar.gz"
if [[ "$goos" == "Windows" ]]; then
  asset_name="leangoo_${goos}_${goarch}.zip"
fi
echo "==> 资源: $asset_name"

# 从 github.com/releases/latest 的 Location 解析 tag（不走 api.github.com，避免 403 限流）
resolve_latest_tag() {
  local headers loc
  # 不要 -L：需要读 302 Location；-f 会对 3xx 误判，故不用 -f
  headers="$(curl -sSI --max-redirs 0 "https://github.com/${GITHUB_REPO}/releases/latest" 2>/dev/null || true)"
  loc="$(printf '%s' "$headers" | tr -d '\r' | awk 'tolower($1)=="location:"{print $2; exit}')"
  if [[ -z "$loc" ]]; then
    return 1
  fi
  printf '%s' "$loc" | sed -n 's|.*/releases/tag/\([^/?#]*\).*|\1|p' | head -1
}

api_auth_args=()
if [[ -n "${GITHUB_TOKEN:-}" ]]; then
  api_auth_args=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
elif [[ -n "${GH_TOKEN:-}" ]]; then
  api_auth_args=(-H "Authorization: Bearer ${GH_TOKEN}")
fi

if [[ -z "$VERSION" || "$VERSION" == "latest" ]]; then
  echo "==> 查询最新 Release…"
  VERSION="$(resolve_latest_tag || true)"
  if [[ -z "$VERSION" ]]; then
    # 可选：带 token 时回退 API
    VERSION="$(curl -fsSL "${api_auth_args[@]}" -H "Accept: application/vnd.github+json" -H "User-Agent: leangoo-install" \
      "${API}/releases/latest" 2>/dev/null \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1 || true)"
  fi
  if [[ -z "$VERSION" ]]; then
    echo "无法解析最新 tag（GitHub API 可能被限流）。可手动指定：bash install.sh v1.2.0" >&2
    echo "或设置 GITHUB_TOKEN / GH_TOKEN 后再试。" >&2
    exit 1
  fi
  asset_url="https://github.com/${GITHUB_REPO}/releases/latest/download/${asset_name}"
else
  case "$VERSION" in
    v*) ;;
    *) VERSION="v${VERSION}" ;;
  esac
  asset_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${asset_name}"
fi
echo "==> 版本: $VERSION"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
archive="$tmpdir/$asset_name"
echo "==> 下载…"
if ! curl -fL -o "$archive" -A "leangoo-install" "$asset_url"; then
  echo "直链下载失败: $asset_url" >&2
  echo "尝试通过 GitHub API 解析 asset…" >&2
  rel_json="$(curl -fsSL "${api_auth_args[@]}" -H "Accept: application/vnd.github+json" -H "User-Agent: leangoo-install" \
    "${API}/releases/tags/${VERSION}" 2>/dev/null || true)"
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
    exit 1
  fi
  curl -fL -o "$archive" -A "leangoo-install" "$asset_url"
fi

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
    curl -fL -o "$skills_archive" -A "leangoo-install" \
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
echo "    重装/分享安装命令（推荐 jsDelivr）："
echo "    curl -fsSL https://cdn.jsdelivr.net/gh/${GITHUB_REPO}@main/scripts/install.sh | bash"
