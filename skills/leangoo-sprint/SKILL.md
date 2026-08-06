---
name: leangoo-sprint
description: >-
  用 leangoo CLI 管理企业/项目/Sprint（看板）：列表与结构、状态/周期/重命名/可见性/收藏、
  看板动态与导出、列与泳道增删改复制排序移动、回收站列表与恢复。
  支持 lg.team 看板链接（/kanban/board/go/{board_uuid}）。
  用户提到领歌项目、Sprint、看板、泳道、列、回收站、看板状态、导出时使用。
  卡片操作见 leangoo-story；登录见 leangoo-shared。
---

# leangoo Sprint / 项目 / 企业 / 列泳道 / 回收站

| 用户说法 | CLI |
|----------|-----|
| 企业 | `ent`（团队版 id=`-1`） |
| 项目 | `project` |
| Sprint / 看板 | `sprint` |
| 列 | `list` |
| 泳道 | `lane` |
| 回收站 | `recycle` |

## 链接

```bash
leangoo sprint get 'https://www.lg.team/kanban/board/go/<board_uuid>#/board_view'
```

带 `/{task_uuid}` 时改用 `leangoo-story` 的 `story get '<完整URL>'`。

## 前置

```bash
leangoo auth status   # 未登录 → leangoo-shared
```

## 企业 / 项目 / Sprint 读

```bash
leangoo ent list
leangoo ent use 15599                 # 或名称；团队版：leangoo ent use -- -1
leangoo project list
leangoo sprint list --project <project_id>
leangoo sprint get <board_uuid|url>   # structure: lane_*/list_*
```

## Sprint 写

```bash
leangoo sprint set-status <board|url> --status unset|pending|ongoing|finished
leangoo sprint set-period <board|url> --begin YYYY-MM-DD --end YYYY-MM-DD
leangoo sprint rename <board|url> --name 'Sprint 2'
leangoo sprint set-visibility <board|url> --visibility <int>   # 页面 data-limit
leangoo sprint star <board|url>
leangoo sprint star <board|url> --unstar
leangoo sprint activity <board|url> [--last-id <id>]
leangoo sprint export-excel <board|url>    # 返回需登录 Cookie 打开的 URL
leangoo sprint export-word <board|url> [--with-attachment]
```

## 列 `list`

```bash
leangoo list add --sprint <board> --name 'review'
leangoo list rename <list_id|名> --sprint <board> --name '待办'
leangoo list set-done <list_id|名> --sprint <board> --done=true|false
leangoo list reorder <list_id|名> --sprint <board> --order <n>
leangoo list copy <list_id|名> --sprint <board> [--to-sprint <board>] --yes
leangoo list delete <list_id|名> --sprint <board> --yes
```

## 泳道 `lane`

```bash
leangoo lane add --sprint <board> --name '泳道A'
leangoo lane copy <lane_id|名> --sprint <board> [--to-sprint <board>] --yes
leangoo lane move <lane_id|名> --sprint <board> --to-sprint <board> --yes
leangoo lane delete <lane_id|名> --sprint <board> --yes
```

## 回收站 `recycle`

```bash
leangoo recycle list --sprint <board> [--last-id <id>]
leangoo recycle restore-task <task_id> --sprint <board> --list <列> --yes
leangoo recycle restore-list <list_id> --sprint <board> --yes
leangoo recycle restore-lane <lane_id> --sprint <board> --yes
```

回收站列表来自删除类看板动态；恢复卡片必须指定目标列。

## Agent 工作流

1. 有看板链接 → `sprint get` / `story list --sprint '<URL>'`
2. 否则：`ent list` → `ent use` → `project list` → `sprint list`
3. 卡片 → `leangoo-story`
4. 删/复制/移动/恢复必须 `--yes`
