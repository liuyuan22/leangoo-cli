package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/client"
)

func ListStoryAttachments(c *client.Client, boardUUID, taskKey string) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	raw, err := c.GetRaw(fmt.Sprintf("/task/uploadFile/%s/%v", ctx.BoardID, st.TaskID), nil)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]any{"task_id": st.TaskID, "raw": json.RawMessage(raw)}, nil
	}
	return map[string]any{
		"task_id":   st.TaskID,
		"task_uuid": st.TaskUUID,
		"files":     payload["files"],
		"message":   payload,
	}, nil
}

func AddStoryLinkAttachment(c *client.Client, boardUUID, taskKey, link string) (map[string]any, error) {
	link = strings.TrimSpace(link)
	if link == "" {
		return nil, fmt.Errorf("--url 不能为空")
	}
	if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
		link = "http://" + link
	}
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
	form.Set("link", link)
	api, _, err := c.PostForm("/task/submitLinkAttachment", form)
	if err := checkAPI(api, err, "添加链接附件"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_id": st.TaskID, "link": link, "message": msg}, nil
}

func UploadStoryAttachment(c *client.Client, boardUUID, taskKey, filePath string) (map[string]any, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("--file 不能为空")
	}
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	name := filepath.Base(filePath)
	// Frontend switches upload URL to /task/uploadFile/{board_id}/{task_id}.
	path := fmt.Sprintf("/task/uploadFile/%s/%v", ctx.BoardID, st.TaskID)
	api, raw, err := c.PostMultipart(path, nil, "userfile", name, f)
	if err != nil {
		return nil, fmt.Errorf("上传附件: %w", err)
	}
	// jquery-fileupload may return {files:[...]} without succeed
	if api != nil && api.OK() {
		var msg any
		_ = json.Unmarshal(api.Message, &msg)
		return map[string]any{"task_id": st.TaskID, "file": name, "message": msg}, nil
	}
	var payload any
	_ = json.Unmarshal(raw, &payload)
	if payload != nil {
		return map[string]any{"task_id": st.TaskID, "file": name, "message": payload}, nil
	}
	if err := checkAPI(api, err, "上传附件"); err != nil {
		return nil, err
	}
	return map[string]any{"task_id": st.TaskID, "file": name}, nil
}

func DeleteStoryAttachment(c *client.Client, boardUUID, taskKey, fileID string) (map[string]any, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return nil, fmt.Errorf("--file-id 不能为空")
	}
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
	form.Set("id", fileID)
	raw, err := c.PostFormRaw("/task/deleteFile", form)
	if err != nil {
		return nil, fmt.Errorf("删除附件: %w", err)
	}
	// Upstream may return [{sucess:true,"file":false}] (typo) or {succeed:true}.
	body := string(raw)
	ok := strings.Contains(body, `"sucess":true`) || strings.Contains(body, `"success":true`) || strings.Contains(body, `"succeed":true`)
	if !ok {
		if api, _, err2 := c.PostForm("/task/deleteFile", form); err2 == nil && api != nil && api.OK() {
			ok = true
		}
	}
	if !ok && strings.Contains(body, "invalid_submit") {
		return nil, fmt.Errorf("删除附件 失败: invalid_submit")
	}
	var payload any
	_ = json.Unmarshal(raw, &payload)
	return map[string]any{"task_id": st.TaskID, "file_id": fileID, "deleted": true, "message": payload}, nil
}

func connectBaseBody(c *client.Client, ctx *BoardContext, taskID any) map[string]any {
	boardID, _ := strconv.ParseInt(ctx.BoardID, 10, 64)
	var entID any = 0
	if c.Session != nil && c.Session.CurrentEnt != nil {
		entID = c.Session.CurrentEnt.ID
	}
	return map[string]any{
		"card":    []any{},
		"board":   []any{},
		"mindmap": []any{},
		"project": []any{},
		"file":    []any{},
		"folder":  "",
		"ent_id":  entID,
		"board_id": boardID,
		"task_id":  fmt.Sprint(taskID),
	}
}

// AddStoryURLConnect adds a URL-type connection on the card sidebar「关联」.
func AddStoryURLConnect(c *client.Client, boardUUID, taskKey, linkURL, title string) (map[string]any, error) {
	linkURL = strings.TrimSpace(linkURL)
	if linkURL == "" {
		return nil, fmt.Errorf("--url 不能为空")
	}
	if !strings.HasPrefix(linkURL, "http://") && !strings.HasPrefix(linkURL, "https://") {
		linkURL = "http://" + linkURL
	}
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	body := connectBaseBody(c, ctx, st.TaskID)
	body["link"] = map[string]any{
		"url":   linkURL,
		"title": title,
	}
	api, _, err := c.PostJSONKanban("/task_conn/add_connect", body)
	if err := checkAPI(api, err, "添加关联"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_id": st.TaskID, "url": linkURL, "message": msg}, nil
}

// AddStoryCardConnect links another card via sidebar「关联」(task_conn/add_connect).
func AddStoryCardConnect(c *client.Client, boardUUID, taskKey, linkTaskKey string) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	link, err := FindStory(stories, linkTaskKey)
	if err != nil {
		return nil, fmt.Errorf("关联目标: %w", err)
	}
	body := connectBaseBody(c, ctx, st.TaskID)
	body["card"] = []map[string]any{{
		"to_task_id":  fmt.Sprint(link.TaskID),
		"task_name":   link.TaskName,
		"to_board_id": ctx.BoardID,
	}}
	api, _, err := c.PostJSONKanban("/task_conn/add_connect", body)
	if err := checkAPI(api, err, "添加卡片关联"); err != nil {
		// already linked is OK — still try interconnect
		if api == nil || !strings.Contains(api.MessageString(), "已关联") {
			return nil, err
		}
	}
	var msg any
	if api != nil {
		_ = json.Unmarshal(api.Message, &msg)
	}
	out := map[string]any{
		"task_id":     st.TaskID,
		"to_task_id":  link.TaskID,
		"to_board_id": ctx.BoardID,
		"message":     msg,
	}
	// Make association visible on both cards (侧栏「互相关联」).
	if connectID, ierr := findCardConnectID(c, boardUUID, fmt.Sprint(st.TaskID), fmt.Sprint(link.TaskID)); ierr == nil && connectID != "" {
		if inter, ierr := InterconnectStoryConnect(c, boardUUID, connectID); ierr == nil {
			out["interconnected"] = true
			out["connect_id"] = connectID
			out["interconnect"] = inter
		} else {
			out["interconnected"] = false
			out["interconnect_error"] = ierr.Error()
			out["connect_id"] = connectID
		}
	}
	return out, nil
}

func findCardConnectID(c *client.Client, boardUUID, fromTaskID, toTaskID string) (string, error) {
	listed, err := ListStoryConnects(c, boardUUID, fromTaskID)
	if err != nil {
		return "", err
	}
	raw, _ := listed["connects"]
	arr, _ := raw.([]any)
	if arr == nil {
		// message may already be typed via json.Unmarshal into []any through re-marshal
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &arr)
	}
	for _, item := range arr {
		m, _ := item.(map[string]any)
		if m == nil {
			continue
		}
		if fmt.Sprint(m["to_task_id"]) == toTaskID || fmt.Sprint(m["task_id"]) == toTaskID {
			if id := firstString(m, "id"); id != "" {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("未找到关联记录 %s -> %s", fromTaskID, toTaskID)
}

// InterconnectStoryConnect enables bidirectional sidebar association for an existing connect id.
func InterconnectStoryConnect(c *client.Client, boardUUID, connectID string) (map[string]any, error) {
	connectID = strings.TrimSpace(connectID)
	if connectID == "" {
		return nil, fmt.Errorf("connect id 不能为空")
	}
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("id", connectID)
	api, _, err := c.PostForm("/task_conn/interconnect", form)
	if err := checkAPI(api, err, "互相关联"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"id": connectID, "board_id": ctx.BoardID, "message": msg}, nil
}

// SearchConnectableResources searches cards/resources for sidebar「关联」.
func SearchConnectableResources(c *client.Client, boardUUID, taskKey, keywords string, typ int) (map[string]any, error) {
	keywords = strings.TrimSpace(keywords)
	if keywords == "" {
		return nil, fmt.Errorf("--q 不能为空")
	}
	if typ <= 0 {
		typ = 1 // card
	}
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	entID := "0"
	if c.Session != nil && c.Session.CurrentEnt != nil {
		entID = fmt.Sprint(c.Session.CurrentEnt.ID)
	}
	q := url.Values{}
	q.Set("ent_id", entID)
	q.Set("board_id", ctx.BoardID)
	q.Set("task_id", fmt.Sprint(st.TaskID))
	q.Set("type", strconv.Itoa(typ))
	q.Set("keywords", keywords)
	api, _, err := c.Get("/task_conn/get_connectable_resources", q)
	if err := checkAPI(api, err, "搜索可关联资源"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"task_id":  st.TaskID,
		"keywords": keywords,
		"type":     typ,
		"results":  msg,
	}, nil
}

func ListStoryConnects(c *client.Client, boardUUID, taskKey string) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("board_id", ctx.BoardID)
	q.Set("task_id", fmt.Sprint(st.TaskID))
	api, _, err := c.Get("/task_conn/index", q)
	if err := checkAPI(api, err, "列出关联"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_id": st.TaskID, "connects": msg}, nil
}

func RemoveStoryConnect(c *client.Client, boardUUID, taskKey, connectID string) (map[string]any, error) {
	connectID = strings.TrimSpace(connectID)
	if connectID == "" {
		return nil, fmt.Errorf("--id 不能为空")
	}
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"board_id": ctx.BoardID,
		"task_id":  fmt.Sprint(st.TaskID),
		"id":       connectID,
	}
	api, _, err := c.PostJSONKanban("/task_conn/remove_connect", body)
	if err := checkAPI(api, err, "移除关联"); err != nil {
		// fallback form
		form := url.Values{}
		form.Set("board_id", ctx.BoardID)
		form.Set("task_id", fmt.Sprint(st.TaskID))
		form.Set("id", connectID)
		api, _, err = c.PostForm("/task_conn/remove_connect", form)
		if err := checkAPI(api, err, "移除关联"); err != nil {
			return nil, err
		}
	}
	return map[string]any{"task_id": st.TaskID, "id": connectID, "deleted": true}, nil
}

const (
	linkTypeFinishStart = "2"
	positionPreTask     = "0"
	positionPostTask    = "1"
)

// AddStoryRelation links another task as predecessor/successor (FINISH_START).
func AddStoryRelation(c *client.Client, boardUUID, taskKey, linkTaskKey, position string) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	link, err := FindStory(stories, linkTaskKey)
	if err != nil {
		return nil, fmt.Errorf("关联目标: %w", err)
	}
	pos := positionPostTask
	switch strings.ToLower(strings.TrimSpace(position)) {
	case "pre", "predecessor", "0":
		pos = positionPreTask
	case "post", "successor", "1", "":
		pos = positionPostTask
	default:
		return nil, fmt.Errorf("--position 须为 pre|post")
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("task_id", fmt.Sprint(st.TaskID))
	form.Set("link_board_id", ctx.BoardID)
	form.Set("link_task_id", fmt.Sprint(link.TaskID))
	form.Set("link_type", linkTypeFinishStart)
	form.Set("position", pos)
	api, _, err := c.PostForm("/task/set_dependency", form)
	if err := checkAPI(api, err, "添加依赖"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"task_id":      st.TaskID,
		"link_task_id": link.TaskID,
		"position":     pos,
		"message":      msg,
	}, nil
}

func RemoveStoryRelation(c *client.Client, boardUUID, taskKey, linkTaskKey string) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	link, err := FindStory(stories, linkTaskKey)
	if err != nil {
		return nil, fmt.Errorf("关联目标: %w", err)
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("task_id", fmt.Sprint(st.TaskID))
	form.Set("link_task_id", fmt.Sprint(link.TaskID))
	api, _, err := c.PostForm("/task/remove_dependency", form)
	if err := checkAPI(api, err, "移除依赖"); err != nil {
		return nil, err
	}
	return map[string]any{"task_id": st.TaskID, "link_task_id": link.TaskID, "deleted": true}, nil
}

func BatchDeleteStories(c *client.Client, boardUUID string, taskKeys []string) (map[string]any, error) {
	if len(taskKeys) == 0 {
		return nil, fmt.Errorf("请指定至少一个 task")
	}
	ids := make([]string, 0, len(taskKeys))
	for _, key := range taskKeys {
		out, err := DeleteStory(c, boardUUID, key)
		if err != nil {
			return nil, err
		}
		ids = append(ids, fmt.Sprint(out["task_id"]))
	}
	return map[string]any{"deleted": true, "task_ids": ids, "count": len(ids)}, nil
}

func BatchLinkUsers(c *client.Client, boardUUID string, taskKeys []string, userQuery string, link bool) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	uid, user, err := ResolveBoardUser(userQuery, ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(taskKeys))
	for _, key := range taskKeys {
		st, err := FindStory(stories, key)
		if err != nil {
			return nil, err
		}
		ids = append(ids, fmt.Sprint(st.TaskID))
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("user_id", uid)
	form.Set("tasks", strings.Join(ids, ","))
	path := "/task/link_user_in_batches"
	op := "批量添加成员"
	if !link {
		path = "/task/unlink_user_in_batches"
		op = "批量移除成员"
	}
	api, _, err := c.PostForm(path, form)
	if err := checkAPI(api, err, op); err != nil {
		return nil, err
	}
	return map[string]any{"task_ids": ids, "user": user, "linked": link}, nil
}

func BatchSetTag(c *client.Client, boardUUID string, taskKeys []string, tagQuery string, add bool) (map[string]any, error) {
	tag, err := ResolveBoardTag(c, boardUUID, tagQuery)
	if err != nil {
		return nil, err
	}
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(taskKeys))
	for _, key := range taskKeys {
		st, err := FindStory(stories, key)
		if err != nil {
			return nil, err
		}
		ids = append(ids, fmt.Sprint(st.TaskID))
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("tasks", strings.Join(ids, ","))
	form.Set("tag_code", tag.Code())
	if color := tag.ColorOrCode(); color != "" {
		form.Set("color_code", color)
	}
	path := "/task/add_tag_in_batches"
	op := "批量加标签"
	if !add {
		path = "/task/delete_tag_in_batches"
		op = "批量去标签"
	}
	api, _, err := c.PostForm(path, form)
	if err := checkAPI(api, err, op); err != nil {
		// prefer NEW API single-loop for add
		if add {
			for _, stKey := range taskKeys {
				if _, err2 := AddStoryTagByQuery(c, boardUUID, stKey, tagQuery); err2 != nil {
					return nil, err2
				}
			}
			return map[string]any{"task_ids": ids, "tag": tag.TagName, "added": true, "via": "per-task"}, nil
		}
		return nil, err
	}
	return map[string]any{"task_ids": ids, "tag": tag.TagName, "added": add}, nil
}
