# leangoo-cli

Leangoo（领歌）在线版命令行工具。通过网页接口登录并读取企业 / 项目 / Sprint / Story。

> 非官方接口，页面改版可能失效。仅支持 `lg.team` 在线版。  
> 仓库：https://github.com/liuyuan22/leangoo-cli

## 安装

```bash
curl -fsSL https://raw.githubusercontent.com/liuyuan22/leangoo-cli/main/scripts/install.sh | bash
# 指定版本
curl -fsSL https://raw.githubusercontent.com/liuyuan22/leangoo-cli/main/scripts/install.sh | bash -s -- v0.1.0
```

默认安装到 `~/.local/bin/leangoo`，并把 Skills 拷到 `~/.cursor/skills` 与 `~/.claude/skills`。

### 卸载

```bash
bash scripts/uninstall.sh
```

### 从源码安装

```bash
go build -o bin/leangoo ./cmd/leangoo
cp bin/leangoo ~/.local/bin/leangoo
cp -R skills/leangoo-* ~/.cursor/skills/
```

## 快速开始

```bash
leangoo auth login          # 交互：手机号 → 密码/验证码
leangoo auth status
leangoo auth logout
```

会话：`~/.leangoo-cli/session.json`（Cookie，无明文密码）。

## 企业 / 项目 / Sprint / Story

```bash
leangoo ent list && leangoo ent use 15599
leangoo project list
leangoo sprint list --project <project_id>
leangoo sprint get '<看板URL或board_uuid>'

leangoo story list --sprint '<看板URL>'
leangoo story list --sprint '<看板URL>' --user me --tag 工时
leangoo story get 'https://www.lg.team/kanban/board/go/<board_uuid>/<task_uuid>'
```

## 发版（维护者）

本地打 tag 并推送，**GitHub Actions 自动打包发版**：

```bash
bash scripts/release.sh v0.1.0
```

等价于：

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

无需 token / 公司镜像。跑完后在 Releases 下载：  
https://github.com/liuyuan22/leangoo-cli/releases

## Agent Skills

```text
skills/
├── leangoo-shared/
├── leangoo-sprint/
└── leangoo-story/
```

也可：`npx skills add https://github.com/liuyuan22/leangoo-cli.git -g -y`

## 命令一览

| 命令 | 说明 |
|------|------|
| `auth login/status/logout/send-code` | 登录相关 |
| `ent list` / `ent use` | 企业 |
| `project list` | 项目 |
| `sprint list` / `sprint get` | Sprint |
| `story list/get/users/tags` | Story |

## 数据说明

- Sprint 结构来自看板页 `board_data`
- Story 列表来自 `getLaneTasks`；描述 / 历史为 `getTaskDesc` / `getTaskActivity`
