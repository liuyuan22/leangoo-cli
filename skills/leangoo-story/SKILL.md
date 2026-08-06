---
name: leangoo-story
description: >-
  用 leangoo CLI 完整操作 Sprint 下 Story（卡片）：列表筛选、详情、创建/编辑/删除、
  复制/移动/引用、评论、日期、完成勾选、点数与实际工时、卡片类型、
  成员/标签/检查项/附件/URL 关联/前后置依赖、批量删除与批量成员标签。
  支持 lg.team 看板与卡片链接。用户提到领歌 Story、卡片、任务、拆前后端、标签、点数、
  检查项、附件、关联、评论时使用。依赖已登录；导航见 leangoo-sprint。
---

# leangoo Story（卡片）

Story = 看板上的 Task 卡片。破坏性命令加 `--yes`。`<task>` 可为 `task_id` / `task_uuid` / 标题 / 完整卡片 URL。

## 链接（优先整 URL，勿让用户手拆 UUID）

| 链接 | 做法 |
|------|------|
| `.../board/go/{board_uuid}` | `story list --sprint '<URL>'` 或 `sprint get '<URL>'` |
| `.../board/go/{board_uuid}/{task_uuid}` | `story get '<URL>'` |

`--sprint` 也接受裸 `board_uuid`。

## 读

```bash
leangoo story list --sprint <board|url>
leangoo story list --sprint <board|url> --user me          # me|id|昵称|邮箱
leangoo story list --sprint <board|url> --tag 工时          # 可多次，AND
leangoo story get <task|url> [--sprint <board>]
leangoo story users --sprint <board|url>
leangoo story tags --sprint <board|url>                    # 板上出现过的标签（去重）
leangoo story board-tags --sprint <board|url>              # 看板可用标签定义
leangoo story comments <task> --sprint <board>
leangoo story checklist <task> --sprint <board>
leangoo story attachments <task> --sprint <board>
leangoo story connects <task> --sprint <board>
```

列表字段：`task_id` / `task_uuid` / `task_name` / `estimate` / `members` / `tags` / `has_desc`（`Y`/`N`，不含正文）。  
`get` 追加 `description`、`activities`。

## 生命周期

```bash
leangoo story create --sprint <board> --title '标题' [--list todo] [--lane ...]
leangoo story edit <task> --sprint <board> [--title ...] [--desc ...] [--estimate 3]
leangoo story delete <task> --sprint <board> --yes
leangoo story copy <task> --sprint <board> --to-list <列> [--to-sprint <board>] --yes
leangoo story move <task> --sprint <board> --to-list <列> [--to-sprint <board>] --yes
leangoo story shortcut <task> --sprint <board> --to-list <列> [--to-sprint <board>] --yes
```

`copy` 返回里 `message.succeed_tasks[].task_id` 为新卡 id。

## 字段

```bash
leangoo story comment <task> --sprint <board> --text '内容'
leangoo story set-dates <task> --sprint <board> --start 'YYYY-MM-DD HH:mm' --due '...'
leangoo story set-dates <task> --sprint <board> --clear-start --clear-due
leangoo story set-done <task> --sprint <board> --done=true|false
leangoo story set-hours <task> --sprint <board> --hours 2.5
leangoo story set-type <task> --sprint <board> --type-id -1    # -1 常为「任务」
```

日期无提醒；格式 `yyyy-MM-dd HH:mm`。`edit` 只改传入的 flag，可只传 `--estimate`。

## 成员 / 标签

```bash
leangoo story member-add <task> --sprint <board> --user me
leangoo story member-remove <task> --sprint <board> --user <昵称|邮箱|id>
leangoo story tag-add <task> --sprint <board> --tag <名称|uuid>
leangoo story tag-remove <task> --sprint <board> --tag <名称|uuid>
```

## 检查清单

```bash
leangoo story checklist-add <task> --sprint <board> --name '验收'
leangoo story checklist-item-add <task> --sprint <board> --checklist <uuid> --text '项'
# 可省略 --checklist（自动建清单）
leangoo story checklist-item-done <task> --sprint <board> --checklist <uuid> --item <uuid> --done=true|false
leangoo story checklist-delete <task> --sprint <board> --checklist <uuid> --yes
```

## 附件

```bash
leangoo story attach-link <task> --sprint <board> --url 'https://example.com'
leangoo story attach-upload <task> --sprint <board> --file ./a.txt
leangoo story attach-delete <task> --sprint <board> --file-id <id> --yes
```

## 关联（两类，勿混用）

```bash
# 侧栏「关联」——URL 资源
leangoo story connect-add <task> --sprint <board> --url 'https://...' [--title '标题']
leangoo story connect-remove <task> --sprint <board> --id <connect_id> --yes

# 前后置依赖——同板两张 Story（拆卡挂接用这个）
leangoo story relate <taskA> --sprint <board> --to <taskB> --position pre|post
leangoo story unrelate <taskA> --sprint <board> --to <taskB> --yes
```

`position`：`pre` 前置，`post` 后置（默认）。

## 批量

```bash
leangoo story batch-delete --sprint <board> --task <id> --task <id2> --yes
leangoo story batch-member --sprint <board> --user me --task <id>... [--remove] --yes
leangoo story batch-tag --sprint <board> --tag 开发任务 --task <id>... [--remove] --yes
```

## Agent 工作流

1. 有链接 → 直接 `list` / `get`（优先完整 URL）
2. 无 board → `leangoo-sprint` 定位
3. 写前确认目标看板；删/移/复/批必须 `--yes`
4. 勿把 `has_desc=Y` 当成已有正文；要正文用 `get`
5. 不确定参数 → `leangoo story <子命令> --help`

### 拆分前后端 Story（常见）

原卡带「前端」「后端」等 tag，描述/评论里写了各方点数时：

1. `story copy` 两次（标题、描述、评论、成员等一并复制，**不要**再改标题/描述）
2. 各副本只调 tag：`tag-remove` 掉不属于自己的
3. 只改点数：`edit --estimate <n>`（**不要** `--title` / `--desc`）
4. `relate` 把原卡与两副本挂上

原则：**复制后除 tag 与 estimate 外一律不动。**

## 注意

- 输出 JSON；卡片多时先摘要标题/点数/成员/标签
- 会话失效 → `leangoo-shared` 重新登录
- 列/泳道/看板元数据 → `leangoo-sprint`
