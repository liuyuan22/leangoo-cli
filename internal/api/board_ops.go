package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/client"
	"github.com/deepglint/leangoo-cli/internal/config"
)

func CopyList(c *client.Client, fromBoardUUID, listQuery, toBoardUUID string) (map[string]any, error) {
	fromCtx, err := LoadBoardContext(c, fromBoardUUID)
	if err != nil {
		return nil, err
	}
	list, err := ResolveList(fromCtx, listQuery)
	if err != nil {
		return nil, err
	}
	if toBoardUUID == "" {
		toBoardUUID = fromBoardUUID
	}
	toCtx := fromCtx
	if toBoardUUID != fromBoardUUID {
		toCtx, err = LoadBoardContext(c, toBoardUUID)
		if err != nil {
			return nil, err
		}
	}
	form := url.Values{}
	form.Set("copy_from_board_id", fromCtx.BoardID)
	form.Set("copy_to_board_id", toCtx.BoardID)
	form.Set("list_id", list.ListID)
	form.Set("action", "copy")
	api, _, err := c.PostForm("/list_c/copy", form)
	if err := checkAPI(api, err, "复制列"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"from_board_id": fromCtx.BoardID,
		"to_board_id":   toCtx.BoardID,
		"list_id":       list.ListID,
		"message":       msg,
	}, nil
}

func ReorderList(c *client.Client, boardUUID, listQuery string, order int) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	list, err := ResolveList(ctx, listQuery)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("list_id", list.ListID)
	form.Set("list_order", strconv.Itoa(order))
	api, _, err := c.PostForm("/list_c/changeOrder", form)
	if err := checkAPI(api, err, "调整列顺序"); err != nil {
		return nil, err
	}
	return map[string]any{"list_id": list.ListID, "list_order": order}, nil
}

func CopyLane(c *client.Client, fromBoardUUID, laneQuery, toBoardUUID string) (map[string]any, error) {
	fromCtx, err := LoadBoardContext(c, fromBoardUUID)
	if err != nil {
		return nil, err
	}
	lane, err := ResolveLane(fromCtx, laneQuery)
	if err != nil {
		return nil, err
	}
	if toBoardUUID == "" {
		toBoardUUID = fromBoardUUID
	}
	toCtx := fromCtx
	if toBoardUUID != fromBoardUUID {
		toCtx, err = LoadBoardContext(c, toBoardUUID)
		if err != nil {
			return nil, err
		}
	}
	form := url.Values{}
	form.Set("copy_from_board_id", fromCtx.BoardID)
	form.Set("copy_to_board_id", toCtx.BoardID)
	form.Set("lane_id", lane.LaneID)
	api, _, err := c.PostForm("/lane/copy", form)
	if err := checkAPI(api, err, "复制泳道"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"from_board_id": fromCtx.BoardID,
		"to_board_id":   toCtx.BoardID,
		"lane_id":       lane.LaneID,
		"message":       msg,
	}, nil
}

func MoveLane(c *client.Client, fromBoardUUID, laneQuery, toBoardUUID string) (map[string]any, error) {
	fromCtx, err := LoadBoardContext(c, fromBoardUUID)
	if err != nil {
		return nil, err
	}
	lane, err := ResolveLane(fromCtx, laneQuery)
	if err != nil {
		return nil, err
	}
	if toBoardUUID == "" {
		return nil, fmt.Errorf("移动泳道需要 --to-sprint")
	}
	toCtx, err := LoadBoardContext(c, toBoardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("from_board_id", fromCtx.BoardID)
	form.Set("to_board_id", toCtx.BoardID)
	form.Set("from_lane_id", lane.LaneID)
	api, _, err := c.PostForm("/lane/move", form)
	if err := checkAPI(api, err, "移动泳道"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"from_board_id": fromCtx.BoardID,
		"to_board_id":   toCtx.BoardID,
		"lane_id":       lane.LaneID,
		"message":       msg,
	}, nil
}

func SetBoardVisibility(c *client.Client, boardUUID string, visibility int) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("visibility", strconv.Itoa(visibility))
	api, _, err := c.PostForm("/board/changeVisibility", form)
	if err := checkAPI(api, err, "设置可见性"); err != nil {
		return nil, err
	}
	return map[string]any{"board_id": ctx.BoardID, "visibility": visibility}, nil
}

func ToggleBoardStar(c *client.Client, boardUUID string, star bool) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	flag := "N"
	if star {
		flag = "Y"
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("star_flag", flag)
	api, _, err := c.PostForm("/board/toggleStarBoard", form)
	if err := checkAPI(api, err, "收藏看板"); err != nil {
		return nil, err
	}
	return map[string]any{"board_id": ctx.BoardID, "starred": star}, nil
}

func ListBoardActivity(c *client.Client, boardUUID, lastID string, deletedOnly bool) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	if lastID != "" {
		form.Set("last_id", lastID)
	}
	if deletedOnly {
		form.Set("is_delete_action", "Y")
	}
	api, _, err := c.PostForm("/board/getActivity", form)
	if err := checkAPI(api, err, "获取看板动态"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"board_id":   ctx.BoardID,
		"board_uuid": boardUUID,
		"activities": msg,
	}, nil
}

func ListRecycleBin(c *client.Client, boardUUID, lastID string) (map[string]any, error) {
	return ListBoardActivity(c, boardUUID, lastID, true)
}

func RestoreRecycleTask(c *client.Client, boardUUID, taskID, listQuery string) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	list, err := ResolveList(ctx, listQuery)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("task_id", strings.TrimSpace(taskID))
	form.Set("list_id", list.ListID)
	api, _, err := c.PostForm("/recycle/recycleTask", form)
	if err := checkAPI(api, err, "恢复卡片"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_id": taskID, "list_id": list.ListID, "message": msg}, nil
}

func RestoreRecycleList(c *client.Client, boardUUID, listID string) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("list_id", strings.TrimSpace(listID))
	api, _, err := c.PostForm("/recycle/recycleList", form)
	if err := checkAPI(api, err, "恢复列"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"list_id": listID, "message": msg}, nil
}

func RestoreRecycleLane(c *client.Client, boardUUID, laneID string) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("lane_id", strings.TrimSpace(laneID))
	api, _, err := c.PostForm("/recycle/recycleLane", form)
	if err := checkAPI(api, err, "恢复泳道"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"lane_id": laneID, "message": msg}, nil
}

func ExportBoardExcelURL(c *client.Client, boardUUID string) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	base, err := c.NewAPIBase()
	if err != nil {
		return nil, err
	}
	u := strings.TrimRight(base, "/") + "/export_board_to_excel?board_id=" + url.QueryEscape(ctx.BoardID)
	return map[string]any{
		"board_id":    ctx.BoardID,
		"board_uuid":  boardUUID,
		"export_url":  u,
		"note":        "浏览器打开该 URL（需已登录同一会话 Cookie）即可下载 Excel",
		"kanban_host": config.BaseURL,
	}, nil
}

func ExportBoardWord(c *client.Client, boardUUID string, withAttachment bool) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	if withAttachment {
		form.Set("is_export_attachment", "1")
	} else {
		form.Set("is_export_attachment", "0")
	}
	api, _, err := c.PostForm("/board/export_word", form)
	if err := checkAPI(api, err, "导出 Word"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"board_id": ctx.BoardID, "message": msg}, nil
}
