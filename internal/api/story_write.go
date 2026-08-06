package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/client"
)

func newClientTaskID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func checkAPI(api *client.APIResponse, err error, op string) error {
	if err != nil {
		return err
	}
	if api == nil {
		return fmt.Errorf("%s: 空响应", op)
	}
	if !api.OK() {
		msg := api.MessageString()
		if msg == "" {
			msg = fmt.Sprintf("error_code=%v", api.ErrorCode)
		}
		return fmt.Errorf("%s 失败: %s", op, msg)
	}
	return nil
}

func FindStory(stories []Story, key string) (*Story, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("story 标识为空")
	}
	for i := range stories {
		id := fmt.Sprint(stories[i].TaskID)
		if id == key || stories[i].TaskUUID == key || strings.EqualFold(stories[i].TaskName, key) {
			return &stories[i], nil
		}
	}
	return nil, fmt.Errorf("未找到 story: %s", key)
}

func ResolveList(ctx *BoardContext, query string) (*ListDef, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		if len(ctx.ListDefs) == 0 {
			return nil, fmt.Errorf("看板没有列")
		}
		return &ctx.ListDefs[0], nil
	}
	for i := range ctx.ListDefs {
		l := &ctx.ListDefs[i]
		if l.ListID == q || strings.EqualFold(l.ListName, q) {
			return l, nil
		}
	}
	for i := range ctx.ListDefs {
		l := &ctx.ListDefs[i]
		if strings.Contains(strings.ToLower(l.ListName), strings.ToLower(q)) {
			return l, nil
		}
	}
	return nil, fmt.Errorf("未找到列: %s", query)
}

func ResolveLane(ctx *BoardContext, query string) (*LaneDef, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		if len(ctx.LaneDefs) == 0 {
			return nil, fmt.Errorf("看板没有泳道")
		}
		return &ctx.LaneDefs[0], nil
	}
	for i := range ctx.LaneDefs {
		l := &ctx.LaneDefs[i]
		if l.LaneID == q || strings.EqualFold(l.LaneName, q) {
			return l, nil
		}
	}
	for i := range ctx.LaneDefs {
		l := &ctx.LaneDefs[i]
		if strings.Contains(strings.ToLower(l.LaneName), strings.ToLower(q)) {
			return l, nil
		}
	}
	return nil, fmt.Errorf("未找到泳道: %s", query)
}

// ResolveBlockID finds block_id for lane+list from existing stories, or lane blocks in raw lanes.
func ResolveBlockID(ctx *BoardContext, stories []Story, laneID, listID string) string {
	for _, s := range stories {
		if fmt.Sprint(s.LaneID) == laneID && fmt.Sprint(s.ListID) == listID && s.BlockID != "" {
			return s.BlockID
		}
	}
	// scan raw lanes JSON
	var lanes []map[string]json.RawMessage
	if json.Unmarshal(ctx.Lanes, &lanes) != nil {
		return ""
	}
	for _, lane := range lanes {
		var lid string
		_ = json.Unmarshal(lane["lane_id"], &lid)
		if lid != laneID && fmt.Sprint(lane["lane_id"]) != `"`+laneID+`"` {
			// compare loosely
			if strings.Trim(string(lane["lane_id"]), `"`) != laneID {
				continue
			}
		}
		blocksRaw, ok := lane["blocks"]
		if !ok {
			continue
		}
		var blocks []map[string]any
		if json.Unmarshal(blocksRaw, &blocks) != nil {
			continue
		}
		for _, b := range blocks {
			if fmt.Sprint(b["list_id"]) == listID {
				if id := fmt.Sprint(b["block_id"]); id != "" && id != "<nil>" {
					return id
				}
			}
		}
	}
	return ""
}

type CreateStoryOpts struct {
	Title  string
	Lane   string
	List   string
	TypeID string
}

func CreateStory(c *client.Client, boardUUID string, opts CreateStoryOpts) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		return nil, fmt.Errorf("--title 不能为空")
	}
	lane, err := ResolveLane(ctx, opts.Lane)
	if err != nil {
		return nil, err
	}
	list, err := ResolveList(ctx, opts.List)
	if err != nil {
		return nil, err
	}
	blockID := ResolveBlockID(ctx, stories, lane.LaneID, list.ListID)
	if blockID == "" {
		blockID = newClientTaskID() // fallback; server may accept/replace
	}
	taskID := newClientTaskID()
	orderIDs := []string{}
	for _, s := range stories {
		if fmt.Sprint(s.LaneID) == lane.LaneID && fmt.Sprint(s.ListID) == list.ListID {
			orderIDs = append(orderIDs, fmt.Sprint(s.TaskID))
		}
	}
	orderIDs = append(orderIDs, taskID)

	typeID := strings.TrimSpace(opts.TypeID)
	cat := "0"
	if typeID == "" {
		// pick from an existing card in same board
		for _, s := range stories {
			if s.CardTypeID != "" {
				typeID = s.CardTypeID
				cat = fmt.Sprint(s.CardTypeCat)
				break
			}
		}
	}
	if typeID == "" {
		typeID = "0"
	}

	form := url.Values{}
	form.Set("new_task_name", title)
	form.Set("board_id", ctx.BoardID)
	form.Set("lane_id", lane.LaneID)
	form.Set("list_id", list.ListID)
	form.Set("block_id", blockID)
	form.Set("task_id", taskID)
	form.Set("position_before_task", "")
	form.Set("task_order", strings.Join(orderIDs, ","))
	form.Set("type_id", typeID)
	form.Set("card_type_category", cat)

	api, raw, err := c.PostForm("/task/add", form)
	if err := checkAPI(api, err, "创建卡片"); err != nil {
		return nil, err
	}
	out := map[string]any{
		"board_id":   ctx.BoardID,
		"board_uuid": boardUUID,
		"task_id":    taskID,
		"lane_id":    lane.LaneID,
		"list_id":    list.ListID,
		"block_id":   blockID,
		"title":      title,
	}
	var msg map[string]any
	if json.Unmarshal(api.Message, &msg) == nil {
		out["message"] = msg
		if u, ok := msg["task_uuid"]; ok {
			out["task_uuid"] = u
		}
	} else {
		out["raw"] = json.RawMessage(raw)
	}
	return out, nil
}

type EditStoryOpts struct {
	Title    *string
	Desc     *string
	Estimate *string
}

func EditStory(c *client.Client, boardUUID, taskKey string, opts EditStoryOpts) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	tid := fmt.Sprint(st.TaskID)
	changed := map[string]any{}

	if opts.Title != nil {
		title := strings.TrimSpace(*opts.Title)
		if title == "" {
			return nil, fmt.Errorf("--title 不能为空")
		}
		form := url.Values{}
		form.Set("edit_task_name", title)
		form.Set("task_id", tid)
		form.Set("board_id", ctx.BoardID)
		api, _, err := c.PostForm("/task/edit", form)
		if err := checkAPI(api, err, "改标题"); err != nil {
			return nil, err
		}
		changed["title"] = title
	}
	if opts.Desc != nil {
		form := url.Values{}
		form.Set("board_id", ctx.BoardID)
		form.Set("task_id", tid)
		form.Set("task_desc", *opts.Desc)
		form.Set("action", "add_task_desc")
		api, _, err := c.PostForm("/task/editTaskDesc", form)
		if err := checkAPI(api, err, "改描述"); err != nil {
			return nil, err
		}
		changed["description"] = *opts.Desc
	}
	if opts.Estimate != nil {
		est := strings.TrimSpace(*opts.Estimate)
		path := "/task/setEstimateValue/" + tid + "/" + url.PathEscape(est)
		form := url.Values{}
		form.Set("board_id", ctx.BoardID)
		form.Set("task_id", tid)
		form.Set("estimate", est)
		api, _, err := c.PostForm(path, form)
		if err := checkAPI(api, err, "改点数"); err != nil {
			return nil, err
		}
		changed["estimate"] = est
	}
	if len(changed) == 0 {
		return nil, fmt.Errorf("请至少指定 --title / --desc / --estimate 之一")
	}
	return map[string]any{
		"board_id":   ctx.BoardID,
		"task_id":    tid,
		"task_uuid":  st.TaskUUID,
		"changed":    changed,
	}, nil
}

func DeleteStory(c *client.Client, boardUUID, taskKey string) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("task_id", fmt.Sprint(st.TaskID))
	api, _, err := c.PostForm("/task/delete", form)
	if err := checkAPI(api, err, "删除卡片"); err != nil {
		return nil, err
	}
	return map[string]any{
		"deleted":   true,
		"task_id":   st.TaskID,
		"task_uuid": st.TaskUUID,
		"task_name": st.TaskName,
	}, nil
}

func CopyStory(c *client.Client, fromBoardUUID, taskKey, toBoardUUID, toList string) (map[string]any, error) {
	fromCtx, stories, err := ListStories(c, fromBoardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
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
	list, err := ResolveList(toCtx, toList)
	if err != nil {
		return nil, err
	}
	// Frontend axios serializes arrays via String(arr) → "id" / "id1,id2" in form body.
	form := url.Values{}
	form.Set("from_board_id", fromCtx.BoardID)
	form.Set("to_board_id", toCtx.BoardID)
	form.Set("to_list_id", list.ListID)
	form.Set("tasks_id", fmt.Sprint(st.TaskID))
	api, _, err := c.PostForm("/task/copy_task", form)
	if err := checkAPI(api, err, "复制卡片"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"from_board_id": fromCtx.BoardID,
		"to_board_id":   toCtx.BoardID,
		"to_list_id":    list.ListID,
		"task_id":       st.TaskID,
		"message":       msg,
	}, nil
}

func MoveStory(c *client.Client, fromBoardUUID, taskKey, toBoardUUID, toList string) (map[string]any, error) {
	fromCtx, stories, err := ListStories(c, fromBoardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
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
	list, err := ResolveList(toCtx, toList)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("from_board_id", fromCtx.BoardID)
	form.Set("to_board_id", toCtx.BoardID)
	form.Set("to_list_id", list.ListID)
	form.Set("tasks_id", fmt.Sprint(st.TaskID))
	api, _, err := c.PostForm("/task/move_task1", form)
	if err := checkAPI(api, err, "移动卡片"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"moved":         true,
		"from_board_id": fromCtx.BoardID,
		"to_board_id":   toCtx.BoardID,
		"to_list_id":    list.ListID,
		"task_id":       st.TaskID,
		"message":       msg,
	}, nil
}

// ReorderStories places the given stories in the listed relative order within their block
// (same lane+list cell), via /task/change_order. Other cards keep relative positions;
// the group is inserted at the earliest original index among the named cards.
func ReorderStories(c *client.Client, boardUUID string, taskKeys []string) (map[string]any, error) {
	if len(taskKeys) < 2 {
		return nil, fmt.Errorf("请至少指定 2 张卡的顺序，例如 --task 总 --task 前端 --task 后端")
	}
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	ordered := make([]*Story, 0, len(taskKeys))
	seen := map[string]bool{}
	for _, key := range taskKeys {
		st, err := FindStory(stories, key)
		if err != nil {
			return nil, err
		}
		id := fmt.Sprint(st.TaskID)
		if seen[id] {
			return nil, fmt.Errorf("重复的卡片: %s", key)
		}
		seen[id] = true
		ordered = append(ordered, st)
	}
	anchor := ordered[0]
	blockID := fmt.Sprint(anchor.BlockID)
	listID := fmt.Sprint(anchor.ListID)
	laneID := fmt.Sprint(anchor.LaneID)
	if blockID == "" || listID == "" {
		return nil, fmt.Errorf("卡片缺少 block_id/list_id，无法排序")
	}
	// Ensure all targets share the same block; otherwise move them into the anchor block first.
	for _, st := range ordered[1:] {
		if fmt.Sprint(st.BlockID) != blockID || fmt.Sprint(st.ListID) != listID {
			listName := listID
			if ld, err := ResolveList(ctx, listID); err == nil {
				listName = ld.ListName
			}
			if _, err := MoveStory(c, boardUUID, fmt.Sprint(st.TaskID), boardUUID, listName); err != nil {
				return nil, fmt.Errorf("先把 %v 移到同列失败: %w", st.TaskID, err)
			}
		}
	}
	// Refresh after possible moves.
	ctx, stories, err = ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	ordered = ordered[:0]
	for _, key := range taskKeys {
		st, err := FindStory(stories, key)
		if err != nil {
			return nil, err
		}
		ordered = append(ordered, st)
	}
	blockID = fmt.Sprint(ordered[0].BlockID)
	listID = fmt.Sprint(ordered[0].ListID)
	laneID = fmt.Sprint(ordered[0].LaneID)

	var inBlock []Story
	for _, s := range stories {
		if fmt.Sprint(s.BlockID) == blockID && fmt.Sprint(s.ListID) == listID {
			inBlock = append(inBlock, s)
		}
	}
	if len(inBlock) == 0 {
		return nil, fmt.Errorf("未找到 block=%s 下的卡片", blockID)
	}

	wantIDs := make([]string, 0, len(ordered))
	wantSet := map[string]bool{}
	for _, st := range ordered {
		id := fmt.Sprint(st.TaskID)
		wantIDs = append(wantIDs, id)
		wantSet[id] = true
	}
	insertAt := -1
	rest := make([]string, 0, len(inBlock))
	for i, s := range inBlock {
		id := fmt.Sprint(s.TaskID)
		if wantSet[id] {
			if insertAt < 0 {
				insertAt = i
			}
			continue
		}
		rest = append(rest, id)
	}
	if insertAt < 0 {
		insertAt = 0
	}
	if insertAt > len(rest) {
		insertAt = len(rest)
	}
	newOrder := append([]string{}, rest[:insertAt]...)
	newOrder = append(newOrder, wantIDs...)
	newOrder = append(newOrder, rest[insertAt:]...)

	listName := listID
	if ld, err := ResolveList(ctx, listID); err == nil {
		listName = ld.ListName
	}

	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("list_id_to", listID)
	form.Set("lane_id_to", laneID)
	form.Set("list_name_to", listName)
	form.Set("block_id_to", blockID)
	form.Set("task_order_to", strings.Join(newOrder, ","))
	for _, id := range wantIDs {
		form.Add("task_ids[]", id)
	}
	// jQuery sometimes sends task_ids without []; also send comma form as fallback field.
	form.Set("task_ids", strings.Join(wantIDs, ","))

	api, raw, err := c.PostForm("/task/change_order", form)
	if err := checkAPI(api, err, "调整卡片顺序"); err != nil {
		return nil, fmt.Errorf("%w; body=%s", err, string(raw))
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"ok":            true,
		"board_id":      ctx.BoardID,
		"block_id":      blockID,
		"list_id":       listID,
		"lane_id":       laneID,
		"task_order":    newOrder,
		"reordered":     wantIDs,
		"message":       msg,
	}, nil
}

func AddComment(c *client.Client, boardUUID, taskKey, text string) (map[string]any, error) {
	_, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	if st.TaskUUID == "" {
		return nil, fmt.Errorf("卡片缺少 task_uuid，无法评论")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("--text 不能为空")
	}
	body := map[string]any{
		"comment":   text,
		"task_uuid": st.TaskUUID,
		// API expects string (often comma-separated), not JSON array
		"at_user_ids": "",
	}
	api, _, err := c.PostJSON("/tasks/"+st.TaskUUID+"/comments", body)
	if err := checkAPI(api, err, "添加评论"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_uuid": st.TaskUUID, "comment": text, "message": msg}, nil
}

func ListComments(c *client.Client, boardUUID, taskKey string) (map[string]any, error) {
	_, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	if st.TaskUUID == "" {
		return nil, fmt.Errorf("卡片缺少 task_uuid")
	}
	api, raw, err := c.GetNewJSON("/tasks/"+st.TaskUUID+"/comments", nil)
	if err := checkAPI(api, err, "列出评论"); err != nil {
		// some endpoints return array at top-level
		var arr any
		if json.Unmarshal(raw, &arr) == nil {
			return map[string]any{"task_uuid": st.TaskUUID, "comments": arr}, nil
		}
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_uuid": st.TaskUUID, "comments": msg}, nil
}

type SetDatesOpts struct {
	Start      string
	Due        string
	ClearStart bool
	ClearDue   bool
}

func SetStoryDates(c *client.Client, boardUUID, taskKey string, opts SetDatesOpts) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	tid := fmt.Sprint(st.TaskID)
	changed := map[string]any{}

	// Reminder defaults: no reminder → time_steps=-1, time_unit="".
	// Invalid units (e.g. "m") or JSON body yield invalid_submit.
	if opts.ClearStart {
		form := url.Values{}
		form.Set("board_id", ctx.BoardID)
		form.Set("task_id", tid)
		form.Set("time_steps", "-1")
		form.Set("time_unit", "")
		api, _, err := c.PostForm("/task/removeStartTime", form)
		if err := checkAPI(api, err, "清除开始时间"); err != nil {
			return nil, err
		}
		changed["start_cleared"] = true
	} else if opts.Start != "" {
		form := url.Values{}
		form.Set("board_id", ctx.BoardID)
		form.Set("task_id", tid)
		form.Set("start_time", opts.Start)
		form.Set("time_steps", "-1")
		form.Set("time_unit", "")
		api, _, err := c.PostForm("/task/setStartTime", form)
		if err := checkAPI(api, err, "设置开始时间"); err != nil {
			return nil, err
		}
		changed["start_time"] = opts.Start
	}

	if opts.ClearDue {
		form := url.Values{}
		form.Set("board_id", ctx.BoardID)
		form.Set("task_id", tid)
		form.Set("time_steps", "-1")
		form.Set("time_unit", "")
		api, _, err := c.PostForm("/task/removeDeadLine", form)
		if err := checkAPI(api, err, "清除截止时间"); err != nil {
			return nil, err
		}
		changed["due_cleared"] = true
	} else if opts.Due != "" {
		form := url.Values{}
		form.Set("board_id", ctx.BoardID)
		form.Set("task_id", tid)
		form.Set("deadline", opts.Due)
		form.Set("time_steps", "-1")
		form.Set("time_unit", "")
		api, _, err := c.PostForm("/task/setDeadLine", form)
		if err := checkAPI(api, err, "设置截止时间"); err != nil {
			return nil, err
		}
		changed["deadline"] = opts.Due
	}

	if len(changed) == 0 {
		return nil, fmt.Errorf("请指定 --start / --due / --clear-start / --clear-due")
	}
	return map[string]any{"task_id": tid, "task_uuid": st.TaskUUID, "changed": changed}, nil
}

func LinkStoryUser(c *client.Client, boardUUID, taskKey, userQuery string, link bool) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	uid, user, err := ResolveBoardUser(userQuery, ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("task_id", fmt.Sprint(st.TaskID))
	form.Set("user_id", uid)
	path := "/task/linkUser"
	op := "添加成员"
	if !link {
		path = "/task/unlinkUser"
		op = "移除成员"
	}
	api, _, err := c.PostForm(path, form)
	if err := checkAPI(api, err, op); err != nil {
		return nil, err
	}
	return map[string]any{
		"task_id": st.TaskID,
		"user":    user,
		"linked":  link,
	}, nil
}

func AddStoryTag(c *client.Client, boardUUID, taskKey, tagCode, colorCode string) (map[string]any, error) {
	_, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	if st.TaskUUID == "" {
		return nil, fmt.Errorf("卡片缺少 task_uuid")
	}
	// Legacy /task/addTag returns refresh_page_alert; use NEW API.
	body := map[string]any{
		"tag_uuid":   tagCode,
		"task_uuids": st.TaskUUID,
	}
	_ = colorCode
	api, _, err := c.PostJSON("/tags/add_tag_to_tasks", body)
	if err := checkAPI(api, err, "添加标签"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_id": st.TaskID, "task_uuid": st.TaskUUID, "tag_uuid": tagCode, "added": true, "message": msg}, nil
}

func DeleteStoryTag(c *client.Client, boardUUID, taskKey, tagCode, colorCode string) (map[string]any, error) {
	_, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	if st.TaskUUID == "" {
		return nil, fmt.Errorf("卡片缺少 task_uuid")
	}
	body := map[string]any{
		"tag_uuid":   tagCode,
		"task_uuids": st.TaskUUID,
	}
	_ = colorCode
	api, _, err := c.PostJSON("/tags/remove_tag_from_tasks", body)
	if err := checkAPI(api, err, "删除标签"); err != nil {
		return nil, err
	}
	return map[string]any{"task_id": st.TaskID, "task_uuid": st.TaskUUID, "tag_uuid": tagCode, "deleted": true}, nil
}
