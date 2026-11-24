---
name: leangoo-story
description: >-
  用 leangoo CLI 完整操作 Sprint 下 Story（卡片）：列表筛选、详情、创建/编辑/删除、
  复制/移动/排序/拆前后端(split)、引用、评论、日期、完成勾选、点数与实际工时、卡片类型、
  成员/标签/检查项/附件/URL 关联/前后置依赖、批量删除与批量成员标签。
  拆前后端时点数可能在描述（含飞书文档链接）、评论或检查项里，需先定位再 split。
  支持 lg.team 看板与卡片链接。用户提到领歌 Story、卡片、任务、拆前后端、标签、点数、
  检查项、附件、关联、评论时使用。写评论时合并成一条，标题用「前端修改内容」，
  不写读者交代、测试例子，以及改完又和改之前一样的过程问题。
  依赖已登录；导航见 leangoo-sprint。
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
leangoo story reorder --sprint <board> --task <总> --task <前端> --task <后端>
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

## 评论怎么写

说明性评论只留一条。CLI 只能新增评论，不能改、不能删。要替换时先删掉已有评论，再发一条，不要另补一条把内容拆开。

删评论：从 `story comments` 取出 `comment_id`，用 `~/.leangoo-cli/session.json` 里的 `new_leangoo_web_url` 和 Cookie，对 `DELETE {new_leangoo_web_url}/tasks/{task_uuid}/comments/{comment_id}` 发请求。不要把会话内容写进回复。

正文遵守：

1. 开头写「前端修改内容」。不要写「供测试和产品看」这类交代读者身份的话。
2. 只写最终相对改之前的行为变化。修改过程中出现、改完又和改之前一样的问题不要写。
3. 不要写某一次测试的具体例子，例如在哪个数据上勾了几张、数量从多少变成多少。

## 成员 / 标签

```bash
leangoo story member-add <task> --sprint <board> --user me
leangoo story member-remove <task> --sprint <board> --user <昵称|邮箱|id>
leangoo story tag-ensure --sprint <board> --tag 应用前端   # 无则创建看板标签
leangoo story tag-add <task> --sprint <board> --tag <名称|uuid> [--create]
leangoo story tag-remove <task> --sprint <board> --tag <名称|uuid>
```

`--create` / `tag-ensure`：看板上没有「应用前端」「应用后端」等标签时先创建再挂卡。

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
# 侧栏「关联」——挂接另一张卡片或 URL（拆前后端用这个）
leangoo story connect-add <task> --sprint <board> --to <另一张卡>
leangoo story connect-add <task> --sprint <board> --url 'https://...' [--title '标题']
leangoo story connect-search <task> --sprint <board> --q 16682   # 搜可关联卡片
leangoo story connects <task> --sprint <board>
leangoo story connect-remove <task> --sprint <board> --id <connect_id> --yes

# 前后置依赖（甘特/依赖线，不是侧栏「关联」）
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

### 拆分前后端 Story（常用：一条命令）

```bash
leangoo story split '<卡片URL或task>' --sprint '<看板>' --yes
# 点数已明确时显式传入更稳妥：
leangoo story split <task> --sprint <board> --fe-estimate 0.5 --be-estimate 0.5 --yes
```

会自动：确保/创建「应用前端」「应用后端」标签 → 复制两份 → 各留对应 tag → 改副本点数 → 总卡点数清 0 → 侧栏互相关联 → 排序「总 → 前端 → 后端」。  
**不要**再改副本标题/描述；不要用 `relate`（那是前后置依赖）。

#### 点数从哪里找（Agent 必做）

拆分前先确定前端/后端各自点数。按下面顺序收集，找到即可：

1. **卡片描述**（`story get` 的 `description`）  
   - 常见写法：`前：…（0.5）` / `后：…（0.5）`，或「前端 0.5」「后端 1」等  
   - 描述里常有 **飞书文档 / Wiki 链接**：点数可能只写在文档正文里，不要只看卡片文本；用飞书文档 skill（`lark-doc`）打开链接再搜前后端点数
2. **评论**（`story comments`）——评审/估点往往写在评论
3. **检查项**（`story checklist`）——验收或拆分条目里也可能写「前端 x / 后端 y」

`split` 未传 `--fe-estimate` / `--be-estimate` 时，CLI **只会**从描述 + 评论做文本解析，**不会**打开飞书链接，也**不会**读检查项。  
因此：描述里只有飞书链接、或点数在检查项时，Agent 必须先读出点数，再带 `--fe-estimate` / `--be-estimate` 调用 `split`。解析失败时不要反复盲跑 `split`，先补全来源再拆。

## 注意

- 输出 JSON；卡片多时先摘要标题/点数/成员/标签
- 会话失效 → `leangoo-shared` 重新登录
- 列/泳道/看板元数据 → `leangoo-sprint`
