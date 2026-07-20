#!/usr/bin/env bash
# 本地一行触发发版：打 tag 并推到 GitHub，由 Actions 自动打包上传 Release
#
#   bash scripts/release.sh v0.1.0
#
# 等价于：
#   git tag -a v0.1.0 -m "Release v0.1.0"
#   git push origin v0.1.0
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "用法: bash scripts/release.sh v0.1.0" >&2
  exit 1
fi
case "$VERSION" in
  v*) ;;
  *) VERSION="v${VERSION}" ;;
esac

if [[ -n "$(git status --porcelain)" ]]; then
  echo "工作区有未提交改动，请先 commit 再发版。" >&2
  git status --short >&2
  exit 1
fi

# 确保当前 commit 已在远端
branch="$(git rev-parse --abbrev-ref HEAD)"
echo "==> 推送分支 ${branch}"
git push -u origin "HEAD:${branch}"

echo "==> tag ${VERSION}"
if git rev-parse "$VERSION" >/dev/null 2>&1; then
  if [[ "$(git rev-parse "${VERSION}^{commit}")" == "$(git rev-parse HEAD)" ]]; then
    echo "  tag 已指向当前 commit"
  else
    echo "  tag 指向旧 commit，移动到 HEAD"
    git tag -d "$VERSION"
    git tag -a "$VERSION" -m "Release ${VERSION}"
  fi
else
  git tag -a "$VERSION" -m "Release ${VERSION}"
fi

echo "==> 推送 tag（触发 GitHub Actions Release）"
git push origin "refs/tags/${VERSION}" --force

repo="$(git remote get-url origin | sed -E 's#^(git@github\.com:|https://github\.com/)##; s#\.git$##')"
echo "==> 已触发: https://github.com/${repo}/actions"
echo "    Release: https://github.com/${repo}/releases/tag/${VERSION}"
echo "    （Actions 跑完后附件才会出现在 Release 页）"
