---
name: leangoo-shared
description: >-
  Leangoo（领歌）CLI 共享规则：安装/更新 leangoo 二进制、auth 登录/退出/状态、会话目录、
  企业切换前提、JSON 输出约定、顶层命令索引。在用户提到领歌、lg.team、Leangoo、看板登录，
  或执行任何 leangoo 命令前未登录/会话过期时使用。写操作（建卡、改卡、列泳道等）也依赖本 skill 的登录检查。
---

# leangoo-cli 共享规则

通过 **`leangoo`** 命令行操作领歌在线版（`lg.team`）。非官方网页接口，改版可能失效。

## 安装与可用性

```bash
which leangoo || ./bin/leangoo version
```

若没有：

```bash
go build -o bin/leangoo ./cmd/leangoo
# 或
curl -fsSL https://raw.githubusercontent.com/liuyuan22/leangoo-cli/main/scripts/install.sh | bash
```

更新（覆盖当前二进制，默认同步 Skills）：

```bash
leangoo update              # 最新
leangoo update --check
leangoo update --version v1.1.0
```

CLI 启动时会偶尔检查 GitHub 是否有新版本（约每天一次，结果缓存于 `~/.leangoo-cli/version_check.json`），有更新时在 stderr 提示 `leangoo update`。可用 `--no-update-check` 或环境变量 `LEANGOO_NO_UPDATE_CHECK=1` 关闭。

若当前二进制还没有 `update` 子命令，先再跑一次安装脚本。

Agent 优先用 PATH 中的 `leangoo`；仅有仓库产物时用绝对路径或 `./bin/leangoo`。

## 认证（每次操作前）

会话：`~/.leangoo-cli/session.json`（Cookie + 当前企业，**不含明文密码**）。

**硬性步骤：** 调用任何 `ent` / `project` / `sprint` / `story` / `list` / `lane` / `recycle` 前先：

```bash
leangoo auth status
```

| 结果 | 做法 |
|------|------|
| `logged_in: true` | 继续 |
| 未登录 / 会话无效 | **立刻停下**，让用户在本机执行 `leangoo auth login`，登录后再继续；不要在 Agent 里交互输密码 |

```bash
leangoo auth login            # 交互（用户本机跑）
leangoo auth logout
leangoo auth send-code --phone <手机号>
leangoo auth login --phone <账号> --password '<密码>'   # 仅用户主动提供时
leangoo auth login --phone <手机号> --code <验证码>
```

**不要**把密码写入仓库、日志或 skill。

## 输出与导航约定

- 默认输出 **JSON**
- 先企业再业务：`ent list` → `ent use <id|名称>` → `project` / `sprint` / `story`
- 团队版企业 id 为 `-1`（`ent use -- -1` 或名称「团队版」）
- 破坏性写操作（删除/移动/复制/批量等）必须加 `--yes`
- 看板/卡片参数普遍接受 `board_uuid` 或完整 `lg.team` URL

## 顶层命令索引（全量）

| 命令 | Skill | 用途 |
|------|--------|------|
| `auth` | 本 skill | 登录/退出/状态 |
| `update` | 本 skill | 从 GitHub Releases 自更新 |
| `ent` / `project` / `sprint` | `leangoo-sprint` | 企业、项目、看板读写、动态、导出、收藏 |
| `list` / `lane` / `recycle` | `leangoo-sprint` | 列、泳道、回收站 |
| `story` | `leangoo-story` | 卡片读写（含附件/标签/检查项/关联/批量） |
| `version` | — | 版本 |

不确定子命令时：先 `leangoo <cmd> --help`，再按上表打开对应 skill。
