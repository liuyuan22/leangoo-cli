package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/client"
)

func SetBoardBizStatus(c *client.Client, boardUUID, status string) (map[string]any, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "unset", "pending", "ongoing", "finished":
	default:
		return nil, fmt.Errorf("--status 须为 unset|pending|ongoing|finished")
	}
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"board_id":   ctx.BoardID,
		"biz_status": status,
	}
	api, _, err := c.PostJSON("/board/biz_status", body)
	if err := checkAPI(api, err, "设置看板状态"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"board_id":   ctx.BoardID,
		"board_uuid": boardUUID,
		"biz_status": status,
		"message":    msg,
	}, nil
}

func SetBoardPeriod(c *client.Client, boardUUID, begin, end string) (map[string]any, error) {
	begin = strings.TrimSpace(begin)
	end = strings.TrimSpace(end)
	if begin == "" || end == "" {
		return nil, fmt.Errorf("需要 --begin 与 --end（YYYY-MM-DD）")
	}
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("begin_date", begin)
	form.Set("end_date", end)
	api, _, err := c.PostForm("/board/setBoardPeriod", form)
	if err := checkAPI(api, err, "设置看板周期"); err != nil {
		return nil, err
	}
	return map[string]any{
		"board_id":   ctx.BoardID,
		"board_uuid": boardUUID,
		"begin_date": begin,
		"end_date":   end,
	}, nil
}

func RenameBoard(c *client.Client, boardUUID, name string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("--name 不能为空")
	}
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("board_name", name)
	api, _, err := c.PostForm("/board/rename_board", form)
	if err := checkAPI(api, err, "重命名看板"); err != nil {
		return nil, err
	}
	return map[string]any{
		"board_id":   ctx.BoardID,
		"board_uuid": boardUUID,
		"board_name": name,
	}, nil
}

func AddList(c *client.Client, boardUUID, name string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("--name 不能为空")
	}
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("list_name", name)
	api, raw, err := c.PostForm("/list_c/add", form)
	if err := checkAPI(api, err, "添加列"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"board_id": ctx.BoardID, "list_name": name, "message": msg, "raw": json.RawMessage(raw)}, nil
}

func RenameList(c *client.Client, boardUUID, listQuery, name string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("--name 不能为空")
	}
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
	form.Set("list_name", name)
	api, _, err := c.PostForm("/list_c/rename_list", form)
	if err := checkAPI(api, err, "重命名列"); err != nil {
		return nil, err
	}
	return map[string]any{"list_id": list.ListID, "list_name": name}, nil
}

func DeleteList(c *client.Client, boardUUID, listQuery string) (map[string]any, error) {
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
	api, _, err := c.PostForm("/list_c/delete", form)
	if err := checkAPI(api, err, "删除列"); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "list_id": list.ListID, "list_name": list.ListName}, nil
}

func SetListDone(c *client.Client, boardUUID, listQuery string, done bool) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	list, err := ResolveList(ctx, listQuery)
	if err != nil {
		return nil, err
	}
	path := "/list_c/set_list_done"
	op := "设为完成列"
	if !done {
		path = "/list_c/unset_list_done"
		op = "取消完成列"
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("list_id", list.ListID)
	api, _, err := c.PostForm(path, form)
	if err := checkAPI(api, err, op); err != nil {
		return nil, err
	}
	return map[string]any{"list_id": list.ListID, "is_finish_list": done}, nil
}

func AddLane(c *client.Client, boardUUID, name string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("--name 不能为空")
	}
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("lane_name", name)
	api, raw, err := c.PostForm("/lane/add", form)
	if err := checkAPI(api, err, "添加泳道"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"board_id": ctx.BoardID, "lane_name": name, "message": msg, "raw": json.RawMessage(raw)}, nil
}

func DeleteLane(c *client.Client, boardUUID, laneQuery string) (map[string]any, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	lane, err := ResolveLane(ctx, laneQuery)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("lane_id", lane.LaneID)
	api, _, err := c.PostForm("/lane/delete", form)
	if err := checkAPI(api, err, "删除泳道"); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "lane_id": lane.LaneID, "lane_name": lane.LaneName}, nil
}
