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

func addStoryExtraCommands(cmd *cobra.Command) {
	var sprint, urlStr, title, file, fileID, connectID, linkTask, position, tag, user string
	var yes bool
	var tasks []string

	attList := &cobra.Command{
		Use:   "attachments [task]",
		Short: "列出 Story 附件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.ListStoryAttachments(c, boardUUID, key)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	attList.Flags().StringVar(&sprint, "sprint", "", "看板")
	cmd.AddCommand(attList)

	attLink := &cobra.Command{
		Use:   "attach-link [task] --url <url>",
		Short: "添加链接型附件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.AddStoryLinkAttachment(c, boardUUID, key, urlStr)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	attLink.Flags().StringVar(&sprint, "sprint", "", "看板")
	attLink.Flags().StringVar(&urlStr, "url", "", "链接")
	_ = attLink.MarkFlagRequired("url")
	cmd.AddCommand(attLink)

	attUpload := &cobra.Command{
		Use:   "attach-upload [task] --file <path>",
		Short: "上传本地文件附件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.UploadStoryAttachment(c, boardUUID, key, file)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	attUpload.Flags().StringVar(&sprint, "sprint", "", "看板")
	attUpload.Flags().StringVar(&file, "file", "", "本地文件路径")
	_ = attUpload.MarkFlagRequired("file")
	cmd.AddCommand(attUpload)

	attDel := &cobra.Command{
		Use:   "attach-delete [task] --file-id <id> --yes",
		Short: "删除附件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "删除附件")
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.DeleteStoryAttachment(c, boardUUID, key, fileID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	attDel.Flags().StringVar(&sprint, "sprint", "", "看板")
	attDel.Flags().StringVar(&fileID, "file-id", "", "附件 id")
	attDel.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = attDel.MarkFlagRequired("file-id")
	cmd.AddCommand(attDel)

	connList := &cobra.Command{
		Use:   "connects [task]",
		Short: "列出 Story 关联资源",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.ListStoryConnects(c, boardUUID, key)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	connList.Flags().StringVar(&sprint, "sprint", "", "看板")
	cmd.AddCommand(connList)

	connAdd := &cobra.Command{
		Use:   "connect-add [task] (--to <task>|--url <url>) [--title <标题>]",
		Short: "侧栏「关联」：关联另一张卡片（--to）或 URL（--url）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			to := strings.TrimSpace(linkTask)
			u := strings.TrimSpace(urlStr)
			if to == "" && u == "" {
				output.Fatal(fmt.Errorf("请指定 --to <卡片> 或 --url <链接>"))
			}
			if to != "" && u != "" {
				output.Fatal(fmt.Errorf("--to 与 --url 不能同时使用"))
			}
			var out map[string]any
			if to != "" {
				out, err = api.AddStoryCardConnect(c, boardUUID, key, to)
			} else {
				out, err = api.AddStoryURLConnect(c, boardUUID, key, u, title)
			}
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	connAdd.Flags().StringVar(&sprint, "sprint", "", "看板")
	connAdd.Flags().StringVar(&linkTask, "to", "", "关联目标卡片（task_id/uuid/标题）")
	connAdd.Flags().StringVar(&urlStr, "url", "", "关联 URL")
	connAdd.Flags().StringVar(&title, "title", "", "URL 标题（仅 --url）")
	cmd.AddCommand(connAdd)

	var connectQ string
	var connectType int
	connSearch := &cobra.Command{
		Use:   "connect-search [task] --q <关键词>",
		Short: "搜索可关联的卡片/资源（侧栏关联搜索）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.SearchConnectableResources(c, boardUUID, key, connectQ, connectType)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	connSearch.Flags().StringVar(&sprint, "sprint", "", "看板")
	connSearch.Flags().StringVar(&connectQ, "q", "", "关键词（卡片号/标题）")
	connSearch.Flags().IntVar(&connectType, "type", 1, "资源类型（1=卡片）")
	_ = connSearch.MarkFlagRequired("q")
	cmd.AddCommand(connSearch)

	connRm := &cobra.Command{
		Use:   "connect-remove [task] --id <connect_id> --yes",
		Short: "移除关联",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "移除关联")
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.RemoveStoryConnect(c, boardUUID, key, connectID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	connRm.Flags().StringVar(&sprint, "sprint", "", "看板")
	connRm.Flags().StringVar(&connectID, "id", "", "关联 id")
	connRm.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = connRm.MarkFlagRequired("id")
	cmd.AddCommand(connRm)

	relAdd := &cobra.Command{
		Use:   "relate [task] --to <task> [--position pre|post]",
		Short: "添加前后置依赖（同板）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.AddStoryRelation(c, boardUUID, key, linkTask, position)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	relAdd.Flags().StringVar(&sprint, "sprint", "", "看板")
	relAdd.Flags().StringVar(&linkTask, "to", "", "关联任务")
	relAdd.Flags().StringVar(&position, "position", "post", "pre|post")
	_ = relAdd.MarkFlagRequired("to")
	cmd.AddCommand(relAdd)

	relRm := &cobra.Command{
		Use:   "unrelate [task] --to <task> --yes",
		Short: "移除依赖",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "移除依赖")
			boardUUID, key := resolveTaskAndBoard(args[0], sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.RemoveStoryRelation(c, boardUUID, key, linkTask)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	relRm.Flags().StringVar(&sprint, "sprint", "", "看板")
	relRm.Flags().StringVar(&linkTask, "to", "", "关联任务")
	relRm.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = relRm.MarkFlagRequired("to")
	cmd.AddCommand(relRm)

	batchDel := &cobra.Command{
		Use:   "batch-delete --sprint <board> --task <id> [--task <id>...] --yes",
		Short: "批量删除卡片",
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "批量删除")
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.BatchDeleteStories(c, boardUUID, tasks)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	batchDel.Flags().StringVar(&sprint, "sprint", "", "看板")
	batchDel.Flags().StringArrayVar(&tasks, "task", nil, "task_id/uuid/标题，可多次")
	batchDel.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = batchDel.MarkFlagRequired("sprint")
	cmd.AddCommand(batchDel)

	batchMember := &cobra.Command{
		Use:   "batch-member --sprint <board> --user <...> --task <id>... [--remove] --yes",
		Short: "批量添加/移除成员",
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "批量改成员")
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			link := !cmd.Flag("remove").Changed
			out, err := api.BatchLinkUsers(c, boardUUID, tasks, user, link)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	batchMember.Flags().StringVar(&sprint, "sprint", "", "看板")
	batchMember.Flags().StringVar(&user, "user", "", "成员")
	batchMember.Flags().StringArrayVar(&tasks, "task", nil, "卡片，可多次")
	batchMember.Flags().Bool("remove", false, "移除而非添加")
	batchMember.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = batchMember.MarkFlagRequired("sprint")
	_ = batchMember.MarkFlagRequired("user")
	cmd.AddCommand(batchMember)

	batchTag := &cobra.Command{
		Use:   "batch-tag --sprint <board> --tag <名> --task <id>... [--remove] --yes",
		Short: "批量添加/移除标签",
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "批量改标签")
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			add := !cmd.Flag("remove").Changed
			out, err := api.BatchSetTag(c, boardUUID, tasks, tag, add)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	batchTag.Flags().StringVar(&sprint, "sprint", "", "看板")
	batchTag.Flags().StringVar(&tag, "tag", "", "标签")
	batchTag.Flags().StringArrayVar(&tasks, "task", nil, "卡片，可多次")
	batchTag.Flags().Bool("remove", false, "移除而非添加")
	batchTag.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = batchTag.MarkFlagRequired("sprint")
	_ = batchTag.MarkFlagRequired("tag")
	cmd.AddCommand(batchTag)
}

func addSprintExtraCommands(cmd *cobra.Command) {
	var visibility int
	var withAttach bool
	var lastID string

	visCmd := &cobra.Command{
		Use:   "set-visibility [board] --visibility <int>",
		Short: "设置看板可见性（页面 data-limit 数值）",
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
			out, err := api.SetBoardVisibility(c, ref.BoardUUID, visibility)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	visCmd.Flags().IntVar(&visibility, "visibility", 0, "可见性数值")
	_ = visCmd.MarkFlagRequired("visibility")
	cmd.AddCommand(visCmd)

	starCmd := &cobra.Command{
		Use:   "star [board] [--unstar]",
		Short: "收藏/取消收藏看板",
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
			doStar := !cmd.Flag("unstar").Changed
			out, err := api.ToggleBoardStar(c, ref.BoardUUID, doStar)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	starCmd.Flags().Bool("unstar", false, "取消收藏")
	cmd.AddCommand(starCmd)

	actCmd := &cobra.Command{
		Use:   "activity [board]",
		Short: "看板动态",
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
			out, err := api.ListBoardActivity(c, ref.BoardUUID, lastID, false)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	actCmd.Flags().StringVar(&lastID, "last-id", "", "分页游标")
	cmd.AddCommand(actCmd)

	exportExcel := &cobra.Command{
		Use:   "export-excel [board]",
		Short: "获取看板 Excel 导出 URL",
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
			out, err := api.ExportBoardExcelURL(c, ref.BoardUUID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	cmd.AddCommand(exportExcel)

	exportWord := &cobra.Command{
		Use:   "export-word [board]",
		Short: "导出看板 Word（返回服务端消息/下载参数）",
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
			out, err := api.ExportBoardWord(c, ref.BoardUUID, withAttach)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	exportWord.Flags().BoolVar(&withAttach, "with-attachment", false, "包含附件")
	cmd.AddCommand(exportWord)
}

func recycleCmd() *cobra.Command {
	var sprint, list, lastID string
	var yes bool
	cmd := &cobra.Command{
		Use:   "recycle",
		Short: "回收站（列表/恢复）",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出回收站（删除类动态）",
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.ListRecycleBin(c, boardUUID, lastID)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	listCmd.Flags().StringVar(&sprint, "sprint", "", "看板")
	listCmd.Flags().StringVar(&lastID, "last-id", "", "分页")
	_ = listCmd.MarkFlagRequired("sprint")
	cmd.AddCommand(listCmd)

	restoreTask := &cobra.Command{
		Use:   "restore-task [task_id] --list <列> --yes",
		Short: "从回收站恢复卡片",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "恢复卡片")
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.RestoreRecycleTask(c, boardUUID, args[0], list)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	restoreTask.Flags().StringVar(&sprint, "sprint", "", "看板")
	restoreTask.Flags().StringVar(&list, "list", "", "恢复到的列")
	restoreTask.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = restoreTask.MarkFlagRequired("sprint")
	_ = restoreTask.MarkFlagRequired("list")
	cmd.AddCommand(restoreTask)

	restoreList := &cobra.Command{
		Use:   "restore-list [list_id] --yes",
		Short: "恢复列",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "恢复列")
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.RestoreRecycleList(c, boardUUID, args[0])
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	restoreList.Flags().StringVar(&sprint, "sprint", "", "看板")
	restoreList.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = restoreList.MarkFlagRequired("sprint")
	cmd.AddCommand(restoreList)

	restoreLane := &cobra.Command{
		Use:   "restore-lane [lane_id] --yes",
		Short: "恢复泳道",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "恢复泳道")
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.RestoreRecycleLane(c, boardUUID, args[0])
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	restoreLane.Flags().StringVar(&sprint, "sprint", "", "看板")
	restoreLane.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = restoreLane.MarkFlagRequired("sprint")
	cmd.AddCommand(restoreLane)
	return cmd
}

func addListExtraCommands(cmd *cobra.Command) {
	var sprint, toSprint string
	var order int
	var yes bool

	copyCmd := &cobra.Command{
		Use:   "copy [list] [--to-sprint <board>] --yes",
		Short: "复制列",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "复制列")
			boardUUID := resolveBoardUUID(sprint)
			to := boardUUID
			if strings.TrimSpace(toSprint) != "" {
				to = resolveBoardUUID(toSprint)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.CopyList(c, boardUUID, args[0], to)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	copyCmd.Flags().StringVar(&sprint, "sprint", "", "源看板")
	copyCmd.Flags().StringVar(&toSprint, "to-sprint", "", "目标看板")
	copyCmd.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = copyCmd.MarkFlagRequired("sprint")
	cmd.AddCommand(copyCmd)

	reorder := &cobra.Command{
		Use:   "reorder [list] --order <n>",
		Short: "调整列顺序（list_order）",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			boardUUID := resolveBoardUUID(sprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.ReorderList(c, boardUUID, args[0], order)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	reorder.Flags().StringVar(&sprint, "sprint", "", "看板")
	reorder.Flags().IntVar(&order, "order", 0, "顺序值")
	_ = reorder.MarkFlagRequired("sprint")
	_ = reorder.MarkFlagRequired("order")
	cmd.AddCommand(reorder)
}

func addLaneExtraCommands(cmd *cobra.Command) {
	var sprint, toSprint string
	var yes bool

	copyCmd := &cobra.Command{
		Use:   "copy [lane] [--to-sprint <board>] --yes",
		Short: "复制泳道",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "复制泳道")
			boardUUID := resolveBoardUUID(sprint)
			to := boardUUID
			if strings.TrimSpace(toSprint) != "" {
				to = resolveBoardUUID(toSprint)
			}
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.CopyLane(c, boardUUID, args[0], to)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	copyCmd.Flags().StringVar(&sprint, "sprint", "", "源看板")
	copyCmd.Flags().StringVar(&toSprint, "to-sprint", "", "目标看板")
	copyCmd.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = copyCmd.MarkFlagRequired("sprint")
	cmd.AddCommand(copyCmd)

	moveCmd := &cobra.Command{
		Use:   "move [lane] --to-sprint <board> --yes",
		Short: "移动泳道到其他看板",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			requireYes(yes, "移动泳道")
			boardUUID := resolveBoardUUID(sprint)
			to := resolveBoardUUID(toSprint)
			c, err := client.NewFromSession()
			if err != nil {
				output.Fatal(err)
			}
			out, err := api.MoveLane(c, boardUUID, args[0], to)
			if err != nil {
				output.Fatal(err)
			}
			_ = output.Print(out)
		},
	}
	moveCmd.Flags().StringVar(&sprint, "sprint", "", "源看板")
	moveCmd.Flags().StringVar(&toSprint, "to-sprint", "", "目标看板")
	moveCmd.Flags().BoolVar(&yes, "yes", false, "确认")
	_ = moveCmd.MarkFlagRequired("sprint")
	_ = moveCmd.MarkFlagRequired("to-sprint")
	cmd.AddCommand(moveCmd)
}
