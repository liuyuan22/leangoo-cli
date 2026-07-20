---
name: leangoo-shared
description: >-
  Leangoo（领歌）CLI 共享规则：安装 leangoo 二进制、auth 登录/退出/状态、会话目录、
  企业切换前提、JSON 输出约定。在用户提到领歌、lg.team、Leangoo、看板登录，
  或执行任何 leangoo 命令前未登录/会话过期时使用。
---

# leangoo-cli 共享规则

通过 **`leangoo`** 命令行操作领歌在线版（`lg.team`）。非官方网页接口，改版可能失效。

## 安装与可用性

优先确认二进制可用：

```bash
which leangoo || ./bin/leangoo version
```

若没有：

```bash
# 在本仓库
go build -o bin/leangoo ./cmd/leangoo
# 或官方安装
curl -fsSL https://raw.githubusercontent.com/liuyuan22/leangoo-cli/main/scripts/install.sh | bash
```

Agent 调用时优先用 PATH 中的 `leangoo`；若仅有仓库产物，用绝对路径或 `./bin/leangoo`。

## 认证（每次操作前）

会话文件：`~/.leangoo-cli/session.json`（Cookie + 当前企业，**不含明文密码**）。

**Agent 硬性步骤：** 调用任何 `ent` / `project` / `sprint` / `story` 命令前，先执行：

```bash
leangoo auth status
```

| 结果 | 做法 |
|------|------|
| `logged_in: true` | 继续业务命令 |
| `logged_in: false` 或命令报「未登录」 | **立刻停下来**，明确告诉用户：请在本机终端运行 `leangoo auth login`，登录后再说「继续」；不要假装已查到数据，也不要在 Agent 里交互输入密码 |

```bash
leangoo auth status          # 是否已登录（未登录会带 hint）
leangoo auth logout          # 清除本地会话
```

### 登录方式

交互式（推荐，给用户自己跑）：

```bash
leangoo auth login
```

非交互（仅当用户主动提供凭证时）：

```bash
leangoo auth login --phone <账号> --password '<密码>'
leangoo auth send-code --phone <手机号>
leangoo auth login --phone <手机号> --code <验证码>
```

**不要**把密码写入仓库、日志或 skill 文件。

## 输出约定

- 命令默认输出 **JSON**（便于解析）
- 先选企业再查项目/Sprint/Story：`leangoo ent list` → `leangoo ent use <id>`
- 团队版企业 id 为 `-1`，入口为 `/kanban/board_list`

## 领域技能

| Skill | 用途 |
|-------|------|
| `leangoo-sprint` | 企业 / 项目 / Sprint（看板）列表与结构 |
| `leangoo-story` | Story（卡片）列表与详情（描述、动态） |

未登录或报会话无效时，回到本 skill 处理认证。
