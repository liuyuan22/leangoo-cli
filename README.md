# leangoo-cli

Leangoo（领歌）在线版命令行工具。通过网页接口登录并读取企业 / 项目 / Sprint / Story。

> 非官方接口，页面改版可能失效。仅支持 `lg.team` 在线版。  
> 仓库：https://gitlab.deepglint.com/liuyuan/leangoo-cli

## 安装（推荐）

私有仓库需先准备有 `read_api` / `api` 权限的 Token（**不要写进仓库**）：

```bash
export GITLAB_TOKEN=你的_glpat_或项目token

# 安装最新版 CLI + Skills
curl -fsSL --header "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  "https://gitlab.deepglint.com/api/v4/projects/liuyuan%2Fleangoo-cli/repository/files/scripts%2Finstall.sh/raw?ref=main" \
  | bash

# 或指定版本
curl -fsSL --header "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  "https://gitlab.deepglint.com/api/v4/projects/liuyuan%2Fleangoo-cli/repository/files/scripts%2Finstall.sh/raw?ref=v0.1.0" \
  | bash -s -- v0.1.0
```

默认安装到 `~/.local/bin/leangoo`，并把 Skills 拷到：

- `~/.cursor/skills/leangoo-*`
- `~/.claude/skills/leangoo-*`

### 卸载

```bash
curl -fsSL --header "PRIVATE-TOKEN: $GITLAB_TOKEN" \
  "https://gitlab.deepglint.com/api/v4/projects/liuyuan%2Fleangoo-cli/repository/files/scripts%2Funinstall.sh/raw?ref=main" \
  | bash
```

或本地：

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

1. 在 GitLab 项目 **Settings → CI/CD → Variables** 添加：
   - Key: `GITLAB_TOKEN`
   - Value: 有 `api` 权限的 Token（**Masked + Protected**）
2. 推送代码到 `main` 后打 tag：

```bash
git remote add origin git@gitlab.deepglint.com:liuyuan/leangoo-cli.git   # 若尚未添加
git add -A && git commit -m "..." && git push -u origin main

git tag v0.1.0
git push origin v0.1.0
```

3. CI 跑 GoReleaser，产物挂到  
   https://gitlab.deepglint.com/liuyuan/leangoo-cli/-/releases

## Agent Skills

```text
skills/
├── leangoo-shared/
├── leangoo-sprint/
└── leangoo-story/
```

也可：`npx skills add git@gitlab.deepglint.com:liuyuan/leangoo-cli.git -g -y`

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
