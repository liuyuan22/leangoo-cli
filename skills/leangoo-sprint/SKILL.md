---
name: leangoo-sprint
description: >-
  用 leangoo CLI 列出/切换企业、列出项目、列出 Sprint（看板）、查看泳道与列结构。
  支持用户粘贴的 lg.team 看板链接（/kanban/board/go/{board_uuid}）。
  用户提到领歌项目、Sprint、看板清单、泳道、列、board_uuid、看板链接、切换企业时使用。
  依赖 leangoo-shared 的登录与会话。
---

# leangoo Sprint / 项目 / 企业

概念对应：

| 用户说法 | CLI / 数据 |
|----------|------------|
| 企业 | `ent`（团队版 id=`-1`） |
| 项目 | `project` |
| Sprint / 看板 | `sprint`（别名 `board`），标识常用 `board_uuid` |

## 用户粘贴看板链接时

```bash
# 仅 Sprint（可带 #/board_view）
leangoo sprint get 'https://www.lg.team/kanban/board/go/1f17691e-2963-6670-8314-0242ac140003#/board_view'
leangoo story list --sprint 'https://www.lg.team/kanban/board/go/1f17691e-2963-6670-8314-0242ac140003#/board_view'
```

若链接还带 `/{task_uuid}`，改用 `leangoo-story` 的 `story get '<完整URL>'`。

## 前置

```bash
leangoo auth status
# 未登录 → 按 leangoo-shared 引导登录
```

## 企业

```bash
leangoo ent list
leangoo ent use 15599          # 或名称片段、或 -1 / 团队版
```

切换后后续 `project` / `sprint` / `story` 都基于当前企业。

## 项目

```bash
leangoo project list
```

从 JSON 中取 `project_id`（或返回结构里的等价 id 字段）供 Sprint 查询。

## Sprint（看板）

```bash
leangoo sprint list --project <project_id>
leangoo sprint get <board_uuid|board_url>    # 泳道行数、列数、名称
```

`sprint get` 的 `structure`：

- `lane_count` / `lane_names`：泳道（行）
- `list_count` / `list_names`：列

## Agent 工作流

1. 用户给了看板链接 → 直接 `sprint get` / `story list --sprint '<URL>'`
2. 否则：`ent list` → `ent use` → `project list` → `sprint list`
3. 卡片交给 `leangoo-story`
