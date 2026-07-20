---
name: leangoo-story
description: >-
  用 leangoo CLI 列出 Sprint 下 Story（卡片/Task），按成员(--user)或标签(--tag)筛选，
  获取标题、点数、成员、标签、描述与动态历史。支持用户粘贴的 lg.team 看板链接
  （/kanban/board/go/{board_uuid} 或 /{board_uuid}/{task_uuid}）。
  用户提到领歌 Story、卡片、任务、点数、标签、工时标签、描述、动态、看板链接时使用。
  依赖已登录会话；企业/项目导航见 leangoo-sprint。
---

# leangoo Story（卡片）

Story = 看板上的 Task 卡片。

## 用户粘贴链接时（优先）

识别 `lg.team` / `leangoo` 看板 URL，**不要让用户再手动拆 UUID**：

| 链接形态 | 做法 |
|----------|------|
| `.../board/go/{board_uuid}#/board_view` | `leangoo story list --sprint '<完整URL>'` 或 `leangoo sprint get '<完整URL>'` |
| `.../board/go/{board_uuid}/{task_uuid}` | `leangoo story get '<完整URL>'`（可省略 `--sprint`） |

```bash
# 整个 Sprint
leangoo story list --sprint 'https://www.lg.team/kanban/board/go/1f17691e-2963-6670-8314-0242ac140003#/board_view'

# 单张 Story
leangoo story get 'https://www.lg.team/kanban/board/go/1f17691e-2963-6670-8314-0242ac140003/1f1803da-fd59-6e70-a664-0242c0a8e003'
```

`--sprint` 也接受裸 `board_uuid`。

## 列表

```bash
leangoo story list --sprint <board_uuid|url>
# 按成员筛选（me = current_user；也支持用户 id / 昵称 / 邮箱）
leangoo story list --sprint <board_uuid|url> --user me
leangoo story list --sprint <board_uuid|url> --user 刘源
# 按标签筛选（tag_name 子串或 tag_uuid；多个 --tag 为 AND）
leangoo story list --sprint <board_uuid|url> --tag 工时
leangoo story list --sprint <board_uuid|url> --user me --tag 工时
leangoo story users --sprint <board_uuid|url>
leangoo story tags --sprint <board_uuid|url>
```

筛选依据：

- 成员：`linked_users` ↔ `board_data.users` / `current_user`
- 标签：Story 的 `tags[]`（`tag_name` / `tag_uuid`）

列表层字段（来自看板数据）：

| 字段 | 含义 |
|------|------|
| `task_id` / `task_uuid` | 标识 |
| `task_name` | 标题 |
| `estimate` | 点数 |
| `linked_users` | 成员 id |
| `members` | 解析后的成员（id / nick_name / email） |
| `tags` | 标签（tag_name / tag_uuid / group_name） |
| `has_desc` | 是否有描述（`Y`/`N`，不含正文） |

同时返回 `structure`（泳道/列数量）。

## 详情

```bash
leangoo story get <task_id|uuid|标题> --sprint <board_uuid|url>
leangoo story get '<含 board+task 的完整 URL>'
```

在列表字段基础上追加：

- `description`：正文（`getTaskDesc`）
- `activities`：动态/历史（`getTaskActivity`）

## Agent 工作流

1. 用户给了链接 → 按上表直接调用（优先完整 URL）
2. 若无链接也无 `board_uuid` → 用 `leangoo-sprint` 定位
3. `story list` 浏览/筛选；需要正文/历史再 `story get`
4. 总结时区分「列表摘要」与「详情」，勿把 `has_desc=Y` 当成已有正文

## 注意

- 输出为 JSON；卡片很多时先摘要标题/点数/成员/标签
- 会话失效时按 `leangoo-shared` 重新登录
