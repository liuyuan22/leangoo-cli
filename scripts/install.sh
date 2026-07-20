#!/usr/bin/env bash
# 安装 leangoo CLI + Agent Skills（内网 GitLab Releases）
#
# 用法（推荐：在仓库根目录维护 .env，勿提交）：
#   cp .env.example .env   # 填入 GITLAB_TOKEN
#   bash scripts/install.sh
#   bash scripts/install.sh v0.1.0
#
# 无本地仓库时也可：
#   set -a && source /path/to/leangoo-cli/.env && set +a
#   curl -fsSL --header "PRIVATE-TOKEN: $GITLAB_TOKEN" \
#     "https://gitlab.deepglint.com/api/v4/projects/liuyuan%2Fleangoo-cli/repository/files/scripts%2Finstall.sh/raw?ref=main" \
#     | bash
#
# Token 加载顺序（已有环境变量优先）：
#   1) 当前目录 .env
#   2) 仓库根目录 .env（相对本脚本）
#   3) ~/.leangoo-cli/.env
#
# 环境变量：
#   GITLAB_TOKEN / PRIVATE_TOKEN  访问私有 Release（必填，除非仓库公开）
#   GITLAB_HOST                   默认 gitlab.deepglint.com
#   GITLAB_PROJECT                默认 liuyuan/leangoo-cli
#   INSTALL_DIR                   默认 ~/.local/bin
#   INSTALL_SKILLS                默认 1（安装到 ~/.cursor/skills 与 ~/.claude/skills）

set -euo pipefail

_load_env_file() {
  local f="$1"
  [[ -f "$f" ]] || return 1
  set -a
  # shellcheck disable=SC1090
  source "$f"
  set +a
  return 0
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || true)"
if [[ -z "${GITLAB_TOKEN:-${PRIVATE_TOKEN:-}}" ]]; then
  _load_env_file "${PWD}/.env" \
    || { [[ -n "${SCRIPT_DIR}" ]] && _load_env_file "${SCRIPT_DIR}/../.env"; } \
    || _load_env_file "${HOME}/.leangoo-cli/.env" \
    || true
fi

GITLAB_HOST="${GITLAB_HOST:-gitlab.deepglint.com}"
GITLAB_PROJECT="${GITLAB_PROJECT:-liuyuan/leangoo-cli}"
API="https://${GITLAB_HOST}/api/v4"
PROJECT_ENC="$(printf '%s' "$GITLAB_PROJECT" | sed 's|/|%2F|g')"
TOKEN="${GITLAB_TOKEN:-${PRIVATE_TOKEN:-}}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
INSTALL_SKILLS="${INSTALL_SKILLS:-1}"
VERSION="${1:-}"

auth_header=()
if [[ -n "$TOKEN" ]]; then
  auth_header=(-H "PRIVATE-TOKEN: ${TOKEN}")
fi

echo "==> GitLab: https://${GITLAB_HOST}/${GITLAB_PROJECT}"

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
  rel_json="$(curl -fsSL "${auth_header[@]}" \
    "${API}/projects/${PROJECT_ENC}/releases/permalink/latest")"
  VERSION="$(printf '%s' "$rel_json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
  if [[ -z "$VERSION" ]]; then
    echo "无法解析最新 tag。请检查 GITLAB_TOKEN 或手动传入版本：bash install.sh v0.1.0" >&2
    exit 1
  fi
fi
echo "==> 版本: $VERSION"

asset_name="leangoo_${goos}_${goarch}.tar.gz"
if [[ "$goos" == "Windows" ]]; then
  asset_name="leangoo_${goos}_${goarch}.zip"
fi
echo "==> 资源: $asset_name"

# 从 Release links 里找 direct_asset_url / url
rel_json="$(curl -fsSL "${auth_header[@]}" \
  "${API}/projects/${PROJECT_ENC}/releases/${VERSION}")"
asset_url="$(printf '%s' "$rel_json" | python3 -c "
import json,sys
name=sys.argv[1]
d=json.load(sys.stdin)
for a in d.get('assets',{}).get('links',[]) or []:
    n=a.get('name') or ''
    u=a.get('direct_asset_url') or a.get('url') or ''
    if n==name or n.endswith(name) or name in n:
        print(u); sys.exit(0)
# fallback: sources not useful; try construct download path from links containing name
for a in d.get('assets',{}).get('links',[]) or []:
    u=a.get('direct_asset_url') or a.get('url') or ''
    if name in u:
        print(u); sys.exit(0)
sys.exit(1)
" "$asset_name" 2>/dev/null || true)"

if [[ -z "$asset_url" ]]; then
  echo "在 Release ${VERSION} 中未找到 ${asset_name}" >&2
  echo "可用资源：" >&2
  printf '%s' "$rel_json" | python3 -c "
import json,sys
d=json.load(sys.stdin)
for a in d.get('assets',{}).get('links',[]) or []:
    print(' -', a.get('name'), a.get('direct_asset_url') or a.get('url'))
" >&2 || true
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
archive="$tmpdir/$asset_name"
echo "==> 下载…"
curl -fL "${auth_header[@]}" -o "$archive" "$asset_url"

echo "==> 解压到 ${INSTALL_DIR}"
mkdir -p "$INSTALL_DIR"
if [[ "$asset_name" == *.zip ]]; then
  unzip -qo "$archive" -d "$tmpdir/out"
else
  mkdir -p "$tmpdir/out"
  tar -xzf "$archive" -C "$tmpdir/out"
fi
bin_path="$(find "$tmpdir/out" -type f -name 'leangoo' -o -name 'leangoo.exe' | head -1)"
if [[ -z "$bin_path" ]]; then
  echo "压缩包中未找到 leangoo 二进制" >&2
  exit 1
fi
install -m 755 "$bin_path" "${INSTALL_DIR}/leangoo"
echo "已安装: ${INSTALL_DIR}/leangoo"
"${INSTALL_DIR}/leangoo" version || true

# Skills：优先用归档内 skills/，否则按同 tag 拉仓库文件
if [[ "$INSTALL_SKILLS" == "1" ]]; then
  echo "==> 安装 Agent Skills…"
  skills_src=""
  if [[ -d "$tmpdir/out/skills" ]]; then
    skills_src="$tmpdir/out/skills"
  else
    # 从仓库 raw 拉三个 skill（通过 archive API）
    skills_archive="$tmpdir/skills.tgz"
    curl -fL "${auth_header[@]}" -o "$skills_archive" \
      "${API}/projects/${PROJECT_ENC}/repository/archive.tar.gz?sha=${VERSION}"
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
