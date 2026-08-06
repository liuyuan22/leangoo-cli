# leangoo-cli

Leangoo（领歌）在线版命令行工具。通过网页接口登录并读写企业 / 项目 / Sprint / Story。

> 非官方接口，页面改版可能失效。仅支持 `lg.team` 在线版。  
> 仓库：https://github.com/liuyuan22/leangoo-cli

## 安装

```bash
curl -fsSL https://raw.githubusercontent.com/liuyuan22/leangoo-cli/main/scripts/install.sh | bash
# 指定版本
curl -fsSL https://raw.githubusercontent.com/liuyuan22/leangoo-cli/main/scripts/install.sh | bash -s -- v0.1.0
```

默认安装到 `~/.local/bin/leangoo`，并把 Skills 拷到 `~/.cursor/skills` 与 `~/.claude/skills`。

### 更新

```bash
leangoo update              # 升到最新 Release（覆盖当前二进制，默认同步 Skills）
leangoo update --check      # 只检查
leangoo update --version v1.1.0
```

启动其它命令时会自动做轻量版本检查（约每天一次，缓存 `~/.leangoo-cli/version_check.json`）；有新版本时在 stderr 提示运行 `leangoo update`。关闭检查：`--no-update-check` 或 `LEANGOO_NO_UPDATE_CHECK=1`。

旧版本若还没有 `update` 命令，先再跑一次上面的安装脚本，之后即可用 `leangoo update`。

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

# 写操作示例（删除/移动等需 --yes）
leangoo story create --sprint '<看板URL>' --title '新卡片' --list todo
leangoo story edit <task_id> --sprint '<看板URL>' --title '改标题'
leangoo story comment <task_id> --sprint '<看板URL>' --text '进展'
leangoo story set-dates <task_id> --sprint '<看板URL>' --due '2026-08-10 18:00'
# 一键拆前后端（标签/点数/关联/排序）
leangoo story split '<卡片URL>' --sprint '<看板URL>' --fe-estimate 0.5 --be-estimate 0.5 --yes
leangoo sprint set-status '<看板URL>' --status ongoing
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
| `sprint list/get/set-status/set-period/rename/star/activity/export-*` | Sprint 读/写 |
| `list add/rename/delete/set-done/copy/reorder` | 列 |
| `lane add/delete/copy/move` | 泳道 |
| `recycle list/restore-*` | 回收站 |
| `story list/get/users/tags` | Story 读 |
| `story create/edit/delete/copy/move/shortcut` | Story 生命周期 |
| `story comment/comments/set-dates/set-done/set-hours/set-type` | 评论与字段 |
| `story member-*/tag-*/board-tags/checklist*` | 成员/标签/检查项 |
| `story attachments/attach-*/connect*/relate/batch-*` | 附件/关联/批量 |

## 数据说明

- Sprint 结构来自看板页 `board_data`
- Story 列表来自 `getLaneTasks`；描述 / 历史为 `getTaskDesc` / `getTaskActivity`
- 写接口走 `/kanban` form 与 `NEW_LEANGOO_WEB_URL` JSON 两套后端
