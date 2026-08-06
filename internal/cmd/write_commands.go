package cmd

import (
	"fmt"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/api"
	"github.com/deepglint/leangoo-cli/internal/client"
	"github.com/deepglint/leangoo-cli/internal/output"
	"github.com/deepglint/leangoo-cli/internal/parse"
	"github.com/spf13/cobra"
)

func requireYes(yes bool, action string) {
	if !yes {
		output.Fatal(fmt.Errorf("%s 需要确认，请加 --yes", action))
	}
}

func resolveBoardUUID(sprintOrURL string) string {
	u, err := requireBoardUUID(sprintOrURL)
	if err != nil {
		output.Fatal(err)
	}
	return u
}

func resolveTaskAndBoard(arg, sprintFlag string) (boardUUID, taskKey string) {
	arg = strings.TrimSpace(arg)
	if ref, err := parse.ParseBoardRef(arg); err == nil && ref.TaskUUID != "" {
		return ref.BoardUUID, ref.TaskUUID
	}
	boardUUID = resolveBoardUUID(sprintFlag)
	return boardUUID, arg
}

func addStoryWriteCommands(cmd *cobra.Command) {
	var sprint, title, desc, estimate, lane, list, toList, toSprint, text, start, due string
	var yes, clearStart, clearDue bool

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "新建 Story（卡片）",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.CreateStory(c, boardUUID, api.CreateStoryOpts{Title: title, Lane: lane, List: list})
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	createCmd.Flags().StringVar(&sprint, "sprint", "", "看板 UUID 或链接")
	createCmd.Flags().StringVar(&title, "title", "", "卡片标题")
	createCmd.Flags().StringVar(&lane, "lane", "", "泳道 id/名称（默认第一泳道）")
	createCmd.Flags().StringVar(&list, "list", "", "列 id/名称（默认第一列）")
	_ = createCmd.MarkFlagRequired("sprint")
	_ = createCmd.MarkFlagRequired("title")
	cmd.AddCommand(createCmd)

	editCmd := &cobra.Command{
		Use:   "edit [task_id|uuid|name|url]",
		Short: "编辑 Story 标题/描述/点数",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			opts := api.EditStoryOpts{}
			if cmd.Flags().Changed("title") {
				opts.Title = &title
			}
			if cmd.Flags().Changed("desc") {
				opts.Desc = &desc
			}
			if cmd.Flags().Changed("estimate") {
				opts.Estimate = &estimate
			}
			out, err := api.EditStory(c, boardUUID, key, opts)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	editCmd.Flags().StringVar(&sprint, "sprint", "", "看板 UUID 或链接（完整 story URL 时可省略）")
	editCmd.Flags().StringVar(&title, "title", "", "新标题")
	editCmd.Flags().StringVar(&desc, "desc", "", "新描述")
	editCmd.Flags().StringVar(&estimate, "estimate", "", "点数")
	cmd.AddCommand(editCmd)

	delCmd := &cobra.Command{
		Use:   "delete [task_id|uuid|url]",
		Short: "删除 Story（需 --yes）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "删除卡片")
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.DeleteStory(c, boardUUID, key)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	delCmd.Flags().StringVar(&sprint, "sprint", "", "看板 UUID 或链接")
	delCmd.Flags().BoolVar(&yes, "yes", false, "确认删除")
	cmd.AddCommand(delCmd)

	copyCmd := &cobra.Command{
		Use:   "copy [task_id|uuid|url]",
		Short: "复制 Story 到目标列（需 --yes）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "复制卡片")
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			to := boardUUID
			if toSprint != "" {
				to = resolveBoardUUID(toSprint)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.CopyStory(c, boardUUID, key, to, toList)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	copyCmd.Flags().StringVar(&sprint, "sprint", "", "源看板")
	copyCmd.Flags().StringVar(&toList, "to-list", "", "目标列 id/名称")
	copyCmd.Flags().StringVar(&toSprint, "to-sprint", "", "目标看板（默认同源）")
	copyCmd.Flags().BoolVar(&yes, "yes", false, "确认复制")
	_ = copyCmd.MarkFlagRequired("to-list")
	cmd.AddCommand(copyCmd)

	moveCmd := &cobra.Command{
		Use:   "move [task_id|uuid|url]",
		Short: "移动 Story 到目标列（需 --yes）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "移动卡片")
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			to := boardUUID
			if toSprint != "" {
				to = resolveBoardUUID(toSprint)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.MoveStory(c, boardUUID, key, to, toList)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	moveCmd.Flags().StringVar(&sprint, "sprint", "", "源看板")
	moveCmd.Flags().StringVar(&toList, "to-list", "", "目标列 id/名称")
	moveCmd.Flags().StringVar(&toSprint, "to-sprint", "", "目标看板（默认同源）")
	moveCmd.Flags().BoolVar(&yes, "yes", false, "确认移动")
	_ = moveCmd.MarkFlagRequired("to-list")
	cmd.AddCommand(moveCmd)

	var reorderTasks []string
	reorderCmd := &cobra.Command{
		Use:   "reorder --sprint <board> --task <a> --task <b> [--task <c>...]",
		Short: "调整同列单元格内卡片顺序（按 --task 先后排列）",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.ReorderStories(c, boardUUID, reorderTasks)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	reorderCmd.Flags().StringVar(&sprint, "sprint", "", "看板")
	reorderCmd.Flags().StringArrayVar(&reorderTasks, "task", nil, "按期望顺序重复传入 task_id/uuid/标题")
	_ = reorderCmd.MarkFlagRequired("sprint")
	_ = reorderCmd.MarkFlagRequired("task")
	cmd.AddCommand(reorderCmd)

	var feEst, beEst, feTag, beTag string
	splitCmd := &cobra.Command{
		Use:   "split [task] --sprint <board> --yes [--fe-estimate n] [--be-estimate n]",
		Short: "拆成前后端 Story（复制/标签/点数/清零/关联/排序一键完成）",
		Long: `将一张产品 Story 拆成「应用前端」「应用后端」两张开发卡：

  1. 确保前后端标签存在（无则创建），原卡补齐标签
  2. 复制两份（不改标题/描述）
  3. 各副本只保留对应标签，并设置点数
  4. 总卡点数清 0
  5. 侧栏关联（互相关联）总卡 ↔ 前后端
  6. 同列排序：总 → 前端 → 后端

点数优先用 --fe-estimate / --be-estimate；未传则从描述/评论解析
「前：…（n）」「后：…（n）」这类写法。`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "拆分前后端 Story")
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SplitStoryFEBE(c, boardUUID, key, api.SplitStoryOpts{
				FEEstimate: feEst,
				BEEstimate: beEst,
				FETag:      feTag,
				BETag:      beTag,
				ToList:     toList,
			})
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	splitCmd.Flags().StringVar(&sprint, "sprint", "", "看板")
	splitCmd.Flags().StringVar(&toList, "to-list", "", "副本目标列（默认与原卡同列）")
	splitCmd.Flags().StringVar(&feEst, "fe-estimate", "", "前端点数（可从描述解析）")
	splitCmd.Flags().StringVar(&beEst, "be-estimate", "", "后端点数（可从描述解析）")
	splitCmd.Flags().StringVar(&feTag, "fe-tag", "应用前端", "前端标签名")
	splitCmd.Flags().StringVar(&beTag, "be-tag", "应用后端", "后端标签名")
	splitCmd.Flags().BoolVar(&yes, "yes", false, "确认拆分")
	cmd.AddCommand(splitCmd)

	commentCmd := &cobra.Command{
		Use:   "comment [task_id|uuid|url]",
		Short: "给 Story 添加评论",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.AddComment(c, boardUUID, key, text)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	commentCmd.Flags().StringVar(&sprint, "sprint", "", "看板")
	commentCmd.Flags().StringVar(&text, "text", "", "评论内容")
	_ = commentCmd.MarkFlagRequired("text")
	cmd.AddCommand(commentCmd)

	commentsCmd := &cobra.Command{
		Use:   "comments [task_id|uuid|url]",
		Short: "列出 Story 评论",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.ListComments(c, boardUUID, key)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	commentsCmd.Flags().StringVar(&sprint, "sprint", "", "看板")
	cmd.AddCommand(commentsCmd)

	datesCmd := &cobra.Command{
		Use:   "set-dates [task_id|uuid|url]",
		Short: "设置/清除 Story 开始与截止时间",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SetStoryDates(c, boardUUID, key, api.SetDatesOpts{
				Start: start, Due: due, ClearStart: clearStart, ClearDue: clearDue,
			})
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	datesCmd.Flags().StringVar(&sprint, "sprint", "", "看板")
	datesCmd.Flags().StringVar(&start, "start", "", "开始时间，如 2026-08-06 10:00")
	datesCmd.Flags().StringVar(&due, "due", "", "截止时间")
	datesCmd.Flags().BoolVar(&clearStart, "clear-start", false, "清除开始时间")
	datesCmd.Flags().BoolVar(&clearDue, "clear-due", false, "清除截止时间")
	cmd.AddCommand(datesCmd)

	memberAdd := &cobra.Command{
		Use:   "member-add [task] --user <me|id|昵称|邮箱>",
		Short: "给 Story 添加成员",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			user := cmd.Flag("user").Value.String()
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.LinkStoryUser(c, boardUUID, key, user, true)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	memberAdd.Flags().StringVar(&sprint, "sprint", "", "看板")
	memberAdd.Flags().String("user", "", "成员")
	_ = memberAdd.MarkFlagRequired("user")
	cmd.AddCommand(memberAdd)

	memberRm := &cobra.Command{
		Use:   "member-remove [task] --user <...>",
		Short: "移除 Story 成员",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			user := cmd.Flag("user").Value.String()
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.LinkStoryUser(c, boardUUID, key, user, false)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	memberRm.Flags().StringVar(&sprint, "sprint", "", "看板")
	memberRm.Flags().String("user", "", "成员")
	_ = memberRm.MarkFlagRequired("user")
	cmd.AddCommand(memberRm)

	var tag, checklistUUID, itemUUID, typeID, hours string
	var doneFlag bool

	var createTag bool
	tagAdd := &cobra.Command{
		Use:   "tag-add [task] --tag <名称|uuid> [--create]",
		Short: "给 Story 添加标签（--create：不存在则在看板创建）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.AddStoryTagByQueryOpts(c, boardUUID, key, tag, createTag)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	tagAdd.Flags().StringVar(&sprint, "sprint", "", "看板")
	tagAdd.Flags().StringVar(&tag, "tag", "", "标签名称或 uuid")
	tagAdd.Flags().BoolVar(&createTag, "create", false, "标签不存在时自动创建看板自定义标签")
	_ = tagAdd.MarkFlagRequired("tag")
	cmd.AddCommand(tagAdd)

	var tagColor string
	tagEnsure := &cobra.Command{
		Use:   "tag-ensure --sprint <board> --tag <名称> [--color #RRGGBB]",
		Short: "确保看板上存在某标签（无则创建）",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			// Warm NEW_LEANGOO base URL from board page when needed.
			_, _ = api.LoadBoardContext(c, boardUUID)
			t, err := api.EnsureBoardTagColor(c, boardUUID, tag, tagColor)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(map[string]any{"ok": true, "tag": t})
		},
	}
	tagEnsure.Flags().StringVar(&sprint, "sprint", "", "看板")
	tagEnsure.Flags().StringVar(&tag, "tag", "", "标签名称")
	tagEnsure.Flags().StringVar(&tagColor, "color", "", "颜色 #RRGGBB（仅新建时优先色）")
	_ = tagEnsure.MarkFlagRequired("tag")
	_ = tagEnsure.MarkFlagRequired("sprint")
	cmd.AddCommand(tagEnsure)

	tagRm := &cobra.Command{
		Use:   "tag-remove [task] --tag <名称|uuid>",
		Short: "移除 Story 标签",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.DeleteStoryTagByQuery(c, boardUUID, key, tag)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	tagRm.Flags().StringVar(&sprint, "sprint", "", "看板")
	tagRm.Flags().StringVar(&tag, "tag", "", "标签名称或 uuid")
	_ = tagRm.MarkFlagRequired("tag")
	cmd.AddCommand(tagRm)

	boardTags := &cobra.Command{
		Use:   "board-tags",
		Short: "列出看板上可用的标签定义",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			tags, err := api.ListBoardTags(c, boardUUID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(map[string]any{"board_uuid": boardUUID, "count": len(tags), "tags": tags})
		},
	}
	boardTags.Flags().StringVar(&sprint, "sprint", "", "看板")
	_ = boardTags.MarkFlagRequired("sprint")
	cmd.AddCommand(boardTags)

	clList := &cobra.Command{
		Use:   "checklist [task]",
		Short: "列出 Story 检查清单与检查项",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.ListStoryChecklists(c, boardUUID, key)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	clList.Flags().StringVar(&sprint, "sprint", "", "看板")
	cmd.AddCommand(clList)

	clAdd := &cobra.Command{
		Use:   "checklist-add [task] --name <名称>",
		Short: "添加检查清单",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.AddStoryChecklist(c, boardUUID, key, cmd.Flag("name").Value.String())
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	clAdd.Flags().StringVar(&sprint, "sprint", "", "看板")
	clAdd.Flags().String("name", "", "清单名称")
	_ = clAdd.MarkFlagRequired("name")
	cmd.AddCommand(clAdd)

	clItemAdd := &cobra.Command{
		Use:   "checklist-item-add [task] --text <内容> [--checklist <uuid>]",
		Short: "添加检查项（可省略 --checklist，将自动建清单）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.AddStoryChecklistItem(c, boardUUID, key, checklistUUID, cmd.Flag("text").Value.String())
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	clItemAdd.Flags().StringVar(&sprint, "sprint", "", "看板")
	clItemAdd.Flags().StringVar(&checklistUUID, "checklist", "", "检查清单 uuid")
	clItemAdd.Flags().String("text", "", "检查项内容")
	_ = clItemAdd.MarkFlagRequired("text")
	cmd.AddCommand(clItemAdd)

	clItemDone := &cobra.Command{
		Use:   "checklist-item-done [task] --checklist <uuid> --item <uuid> --done=true|false",
		Short: "勾选/取消检查项",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SetStoryChecklistItemDone(c, boardUUID, key, checklistUUID, itemUUID, doneFlag)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	clItemDone.Flags().StringVar(&sprint, "sprint", "", "看板")
	clItemDone.Flags().StringVar(&checklistUUID, "checklist", "", "检查清单 uuid")
	clItemDone.Flags().StringVar(&itemUUID, "item", "", "检查项 uuid")
	clItemDone.Flags().BoolVar(&doneFlag, "done", true, "是否完成")
	_ = clItemDone.MarkFlagRequired("checklist")
	_ = clItemDone.MarkFlagRequired("item")
	cmd.AddCommand(clItemDone)

	clDel := &cobra.Command{
		Use:   "checklist-delete [task] --checklist <uuid> --yes",
		Short: "删除检查清单",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !cmd.Flag("yes").Changed {
				output.Fatal(fmt.Errorf("删除需加 --yes"))
			}
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.DeleteStoryChecklist(c, boardUUID, key, checklistUUID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	clDel.Flags().StringVar(&sprint, "sprint", "", "看板")
	clDel.Flags().StringVar(&checklistUUID, "checklist", "", "检查清单 uuid")
	clDel.Flags().Bool("yes", false, "确认删除")
	_ = clDel.MarkFlagRequired("checklist")
	cmd.AddCommand(clDel)

	setDone := &cobra.Command{
		Use:   "set-done [task] --done=true|false",
		Short: "勾选/取消 Story「已完成」",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SetStoryDone(c, boardUUID, key, doneFlag)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	setDone.Flags().StringVar(&sprint, "sprint", "", "看板")
	setDone.Flags().BoolVar(&doneFlag, "done", true, "是否完成")
	cmd.AddCommand(setDone)

	setType := &cobra.Command{
		Use:   "set-type [task] --type-id <id>",
		Short: "修改卡片类型（type_id，如 -1=任务）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SetStoryType(c, boardUUID, key, typeID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	setType.Flags().StringVar(&sprint, "sprint", "", "看板")
	setType.Flags().StringVar(&typeID, "type-id", "", "卡片类型 id")
	_ = setType.MarkFlagRequired("type-id")
	cmd.AddCommand(setType)

	setHours := &cobra.Command{
		Use:   "set-hours [task] --hours <数字>",
		Short: "设置实际工时",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SetStoryActualHours(c, boardUUID, key, hours)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	setHours.Flags().StringVar(&sprint, "sprint", "", "看板")
	setHours.Flags().StringVar(&hours, "hours", "", "实际工时")
	_ = setHours.MarkFlagRequired("hours")
	cmd.AddCommand(setHours)

	shortcut := &cobra.Command{
		Use:   "shortcut [task] --to-list <列> [--to-sprint <看板>] --yes",
		Short: "创建卡片引用（快捷方式）到目标列",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if !cmd.Flag("yes").Changed {
				output.Fatal(fmt.Errorf("创建引用需加 --yes"))
			}
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			to := toSprint
			if to == "" {
				to = boardUUID
			} else {
				to = resolveBoardUUID(to)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.CreateStoryShortcut(c, boardUUID, key, to, toList)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	shortcut.Flags().StringVar(&sprint, "sprint", "", "源看板")
	shortcut.Flags().StringVar(&toList, "to-list", "", "目标列")
	shortcut.Flags().StringVar(&toSprint, "to-sprint", "", "目标看板（默认同源）")
	shortcut.Flags().Bool("yes", false, "确认")
	_ = shortcut.MarkFlagRequired("to-list")
	cmd.AddCommand(shortcut)
}

func addSprintWriteCommands(cmd *cobra.Command) {
	var status, begin, end, name string

	statusCmd := &cobra.Command{
		Use:   "set-status [board_uuid|url]",
		Short: "设置看板业务状态 unset|pending|ongoing|finished",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ref, err := parse.ParseBoardRef(args[0])
			if err != nil {
				output.Fatal(err)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SetBoardBizStatus(c, ref.BoardUUID, status)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	statusCmd.Flags().StringVar(&status, "status", "", "unset|pending|ongoing|finished")
	_ = statusCmd.MarkFlagRequired("status")
	cmd.AddCommand(statusCmd)

	periodCmd := &cobra.Command{
		Use:   "set-period [board_uuid|url]",
		Short: "设置看板起止日期",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ref, err := parse.ParseBoardRef(args[0])
			if err != nil {
				output.Fatal(err)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SetBoardPeriod(c, ref.BoardUUID, begin, end)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	periodCmd.Flags().StringVar(&begin, "begin", "", "YYYY-MM-DD")
	periodCmd.Flags().StringVar(&end, "end", "", "YYYY-MM-DD")
	_ = periodCmd.MarkFlagRequired("begin")
	_ = periodCmd.MarkFlagRequired("end")
	cmd.AddCommand(periodCmd)

	renameCmd := &cobra.Command{
		Use:   "rename [board_uuid|url]",
		Short: "重命名看板",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ref, err := parse.ParseBoardRef(args[0])
			if err != nil {
				output.Fatal(err)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.RenameBoard(c, ref.BoardUUID, name)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	renameCmd.Flags().StringVar(&name, "name", "", "新名称")
	_ = renameCmd.MarkFlagRequired("name")
	cmd.AddCommand(renameCmd)
}

func listCmd() *cobra.Command {
	var sprint, name string
	var yes, done bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "看板列（添加/重命名/删除/完成列）",
	}
	add := &cobra.Command{
		Use:   "add",
		Short: "添加列",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.AddList(c, boardUUID, name)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	add.Flags().StringVar(&sprint, "sprint", "", "看板")
	add.Flags().StringVar(&name, "name", "", "列名")
	_ = add.MarkFlagRequired("sprint")
	_ = add.MarkFlagRequired("name")
	cmd.AddCommand(add)

	rename := &cobra.Command{
		Use:   "rename [list_id|name]",
		Short: "重命名列",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.RenameList(c, boardUUID, args[0], name)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	rename.Flags().StringVar(&sprint, "sprint", "", "看板")
	rename.Flags().StringVar(&name, "name", "", "新列名")
	_ = rename.MarkFlagRequired("sprint")
	_ = rename.MarkFlagRequired("name")
	cmd.AddCommand(rename)

	del := &cobra.Command{
		Use:   "delete [list_id|name]",
		Short: "删除列（需 --yes）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "删除列")
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.DeleteList(c, boardUUID, args[0])
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	del.Flags().StringVar(&sprint, "sprint", "", "看板")
	del.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = del.MarkFlagRequired("sprint")
	cmd.AddCommand(del)

	setDone := &cobra.Command{
		Use:   "set-done [list_id|name]",
		Short: "设为/取消完成列（--done=false 取消）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SetListDone(c, boardUUID, args[0], done)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	setDone.Flags().StringVar(&sprint, "sprint", "", "看板")
	setDone.Flags().BoolVar(&done, "done", true, "是否完成列")
	_ = setDone.MarkFlagRequired("sprint")
	cmd.AddCommand(setDone)
	addListExtraCommands(cmd)
	return cmd
}

func laneCmd() *cobra.Command {
	var sprint, name string
	var yes bool
	cmd := &cobra.Command{
		Use:   "lane",
		Short: "泳道（添加/删除）",
	}
	add := &cobra.Command{
		Use:   "add",
		Short: "添加泳道",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.AddLane(c, boardUUID, name)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	add.Flags().StringVar(&sprint, "sprint", "", "看板")
	add.Flags().StringVar(&name, "name", "", "泳道名")
	_ = add.MarkFlagRequired("sprint")
	_ = add.MarkFlagRequired("name")
	cmd.AddCommand(add)

	del := &cobra.Command{
		Use:   "delete [lane_id|name]",
		Short: "删除泳道（需 --yes）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "删除泳道")
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.DeleteLane(c, boardUUID, args[0])
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	del.Flags().StringVar(&sprint, "sprint", "", "看板")
	del.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = del.MarkFlagRequired("sprint")
	cmd.AddCommand(del)
	addLaneExtraCommands(cmd)
	return cmd
}
