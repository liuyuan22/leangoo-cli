#!/usr/bin/env bash
# 卸载 leangoo CLI 与本工具相关 Skills / 本地会话
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
REMOVE_SESSION="${REMOVE_SESSION:-1}"
REMOVE_SKILLS="${REMOVE_SKILLS:-1}"

echo "==> 卸载 leangoo"

# PATH 中的二进制
if command -v leangoo >/dev/null 2>&1; then
  bin="$(command -v leangoo)"
  echo "删除: $bin"
  rm -f "$bin"
fi
# 默认安装目录
if [[ -x "${INSTALL_DIR}/leangoo" ]]; then
  echo "删除: ${INSTALL_DIR}/leangoo"
  rm -f "${INSTALL_DIR}/leangoo"
fi
# 常见位置
for p in /usr/local/bin/leangoo "$HOME/bin/leangoo"; do
  if [[ -e "$p" ]]; then
    echo "删除: $p"
    rm -f "$p"
  fi
done

if [[ "$REMOVE_SKILLS" == "1" ]]; then
  for dest in "$HOME/.cursor/skills" "$HOME/.claude/skills"; do
    for s in leangoo-shared leangoo-sprint leangoo-story; do
      if [[ -d "$dest/$s" ]]; then
        echo "删除 Skill: $dest/$s"
        rm -rf "$dest/$s"
      fi
    done
  done
fi

if [[ "$REMOVE_SESSION" == "1" && -d "$HOME/.leangoo-cli" ]]; then
  echo "删除会话目录: $HOME/.leangoo-cli"
  rm -rf "$HOME/.leangoo-cli"
fi

echo "==> 卸载完成"
