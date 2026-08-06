package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/client"
)

const (
	checklistItemUnfinished = 1
	checklistItemFinished   = 99
	deadlineNotFinished     = 0
	deadlineFinished        = 1
)

type BoardTag struct {
	TagCode   string `json:"tag_code,omitempty"`
	TagUUID   string `json:"tag_uuid,omitempty"`
	TagName   string `json:"tag_name,omitempty"`
	ColorCode string `json:"color_code,omitempty"`
	Color     string `json:"color_value,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	GroupUUID string `json:"group_uuid,omitempty"`
}

func (t BoardTag) Code() string {
	if t.TagCode != "" {
		return t.TagCode
	}
	return t.TagUUID
}

func (t BoardTag) ColorOrCode() string {
	if t.ColorCode != "" {
		return t.ColorCode
	}
	return t.Color
}

// ListBoardTags returns tag definitions available on the board (NEW API).
func ListBoardTags(c *client.Client, boardUUID string) ([]BoardTag, error) {
	api, raw, err := c.GetNewJSON("/tags/list_board_tag_groups/"+boardUUID, nil)
	if err != nil {
		return nil, err
	}
	var payload any
	if api != nil && len(api.Message) > 0 {
		_ = json.Unmarshal(api.Message, &payload)
	}
	if payload == nil {
		_ = json.Unmarshal(raw, &payload)
	}
	tags := flattenBoardTags(payload)
	if len(tags) == 0 && api != nil && !api.OK() {
		return nil, fmt.Errorf("列出看板标签失败: %s", api.MessageString())
	}
	return tags, nil
}

func flattenBoardTags(v any) []BoardTag {
	var out []BoardTag
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			out = append(out, flattenBoardTags(item)...)
		}
	case map[string]any:
		for _, key := range []string{
			"board_tag_groups", "tag_groups", "groups",
			"ent_tags", "tags", "custom_tags", "tag_list",
		} {
			if nested, ok := x[key]; ok {
				out = append(out, flattenBoardTags(nested)...)
			}
		}
		name := firstString(x, "tag_name", "default_tag_name")
		uuid := firstString(x, "tag_uuid", "tag_code")
		if name != "" || uuid != "" {
			if name == "" {
				name = firstString(x, "default_tag_name")
			}
			out = append(out, BoardTag{
				TagCode:   firstString(x, "tag_code", "tag_uuid"),
				TagUUID:   firstString(x, "tag_uuid"),
				TagName:   name,
				ColorCode: firstString(x, "color_code"),
				Color:     firstString(x, "color_value", "color", "tag_color"),
				GroupName: firstString(x, "group_name"),
				GroupUUID: firstString(x, "group_uuid"),
			})
		}
	}
	// dedupe by code
	seen := map[string]bool{}
	uniq := make([]BoardTag, 0, len(out))
	for _, t := range out {
		code := t.Code()
		if code == "" && t.TagName == "" {
			continue
		}
		key := code + "|" + t.TagName
		if seen[key] {
			continue
		}
		seen[key] = true
		uniq = append(uniq, t)
	}
	return uniq
}

// ResolveBoardTag matches tag name substring / uuid / tag_code.
func ResolveBoardTag(c *client.Client, boardUUID, query string) (*BoardTag, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("标签不能为空")
	}
	tags, err := ListBoardTags(c, boardUUID)
	if err != nil || len(tags) == 0 {
		// fallback: tags seen on stories
		_, stories, err2 := ListStories(c, boardUUID)
		if err2 != nil && err != nil {
			return nil, err
		}
		if err2 == nil {
			for _, st := range CollectStoryTags(stories) {
				tags = append(tags, BoardTag{
					TagUUID: st.TagUUID,
					TagCode: st.TagUUID,
					TagName: st.TagName,
					Color:   st.Color,
					GroupName: st.GroupName,
					GroupUUID: st.GroupUUID,
				})
			}
		}
		if len(tags) == 0 && err != nil {
			return nil, err
		}
	}
	lower := strings.ToLower(query)
	var matches []BoardTag
	for _, t := range tags {
		if t.Code() == query || t.TagUUID == query || strings.EqualFold(t.TagName, query) {
			tt := t
			return &tt, nil
		}
		if t.TagName != "" && strings.Contains(strings.ToLower(t.TagName), lower) {
			matches = append(matches, t)
		}
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, m.TagName)
		}
		return nil, fmt.Errorf("标签 %q 匹配到多个: %s", query, strings.Join(names, ", "))
	}
	return nil, fmt.Errorf("未找到标签: %s", query)
}

func AddStoryTagByQuery(c *client.Client, boardUUID, taskKey, tagQuery string) (map[string]any, error) {
	tag, err := ResolveBoardTag(c, boardUUID, tagQuery)
	if err != nil {
		return nil, err
	}
	code := tag.Code()
	if code == "" {
		return nil, fmt.Errorf("标签缺少 tag_code/tag_uuid: %s", tag.TagName)
	}
	out, err := AddStoryTag(c, boardUUID, taskKey, code, tag.ColorOrCode())
	if err != nil {
		return nil, err
	}
	out["tag_name"] = tag.TagName
	out["color"] = tag.ColorOrCode()
	return out, nil
}

func DeleteStoryTagByQuery(c *client.Client, boardUUID, taskKey, tagQuery string) (map[string]any, error) {
	tag, err := ResolveBoardTag(c, boardUUID, tagQuery)
	if err != nil {
		return nil, err
	}
	code := tag.Code()
	out, err := DeleteStoryTag(c, boardUUID, taskKey, code, tag.ColorOrCode())
	if err != nil {
		return nil, err
	}
	out["tag_name"] = tag.TagName
	return out, nil
}

func storyUUID(c *client.Client, boardUUID, taskKey string) (*BoardContext, *Story, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, nil, err
	}
	if st.TaskUUID == "" {
		return nil, nil, fmt.Errorf("卡片缺少 task_uuid")
	}
	return ctx, st, nil
}

func ListStoryChecklists(c *client.Client, boardUUID, taskKey string) (map[string]any, error) {
	_, st, err := storyUUID(c, boardUUID, taskKey)
	if err != nil {
		return nil, err
	}
	api, raw, err := c.GetNewJSON("/tasks/"+st.TaskUUID+"/checklists-and-items", nil)
	if err := checkAPI(api, err, "列出检查项"); err != nil {
		var arr any
		if json.Unmarshal(raw, &arr) == nil {
			return map[string]any{"task_uuid": st.TaskUUID, "checklists": arr}, nil
		}
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_id": st.TaskID, "task_uuid": st.TaskUUID, "checklists": msg}, nil
}

func AddStoryChecklist(c *client.Client, boardUUID, taskKey, name string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("检查清单名称不能为空")
	}
	_, st, err := storyUUID(c, boardUUID, taskKey)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"checklist_name": name,
		"task_uuid":      st.TaskUUID,
		"position":       1,
	}
	api, _, err := c.PostJSON("/tasks/"+st.TaskUUID+"/checklists", body)
	if err := checkAPI(api, err, "添加检查清单"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_uuid": st.TaskUUID, "checklist_name": name, "message": msg}, nil
}

func AddStoryChecklistItem(c *client.Client, boardUUID, taskKey, checklistUUID, itemName string) (map[string]any, error) {
	itemName = strings.TrimSpace(itemName)
	if itemName == "" {
		return nil, fmt.Errorf("检查项名称不能为空")
	}
	_, st, err := storyUUID(c, boardUUID, taskKey)
	if err != nil {
		return nil, err
	}
	if checklistUUID == "" || checklistUUID == "new" {
		// create a default checklist then add item
		created, err := AddStoryChecklist(c, boardUUID, taskKey, "检查清单")
		if err != nil {
			return nil, err
		}
		if m, ok := created["message"].(map[string]any); ok {
			checklistUUID = firstString(m, "checklist_uuid", "uuid")
		}
		if checklistUUID == "" {
			return nil, fmt.Errorf("无法解析新建检查清单 uuid")
		}
	}
	body := map[string]any{
		"item_name":      itemName,
		"task_uuid":      st.TaskUUID,
		"checklist_uuid": checklistUUID,
	}
	api, _, err := c.PostJSON("/tasks/"+st.TaskUUID+"/checklists/"+checklistUUID+"/items", body)
	if err := checkAPI(api, err, "添加检查项"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{
		"task_uuid":      st.TaskUUID,
		"checklist_uuid": checklistUUID,
		"item_name":      itemName,
		"message":        msg,
	}, nil
}

func SetStoryChecklistItemDone(c *client.Client, boardUUID, taskKey, checklistUUID, itemUUID string, done bool) (map[string]any, error) {
	_, st, err := storyUUID(c, boardUUID, taskKey)
	if err != nil {
		return nil, err
	}
	status := checklistItemUnfinished
	if done {
		status = checklistItemFinished
	}
	body := map[string]any{
		"item_uuid":      itemUUID,
		"status":         status,
		"task_uuid":      st.TaskUUID,
		"checklist_uuid": checklistUUID,
	}
	api, _, err := c.PutJSON("/tasks/"+st.TaskUUID+"/checklists/"+checklistUUID+"/items/"+itemUUID, body)
	if err := checkAPI(api, err, "更新检查项状态"); err != nil {
		return nil, err
	}
	return map[string]any{
		"task_uuid":      st.TaskUUID,
		"checklist_uuid": checklistUUID,
		"item_uuid":      itemUUID,
		"done":           done,
		"status":         status,
	}, nil
}

func DeleteStoryChecklist(c *client.Client, boardUUID, taskKey, checklistUUID string) (map[string]any, error) {
	_, st, err := storyUUID(c, boardUUID, taskKey)
	if err != nil {
		return nil, err
	}
	api, _, err := c.DeleteJSON("/tasks/"+st.TaskUUID+"/checklists/"+checklistUUID, map[string]any{})
	if err := checkAPI(api, err, "删除检查清单"); err != nil {
		return nil, err
	}
	return map[string]any{"task_uuid": st.TaskUUID, "checklist_uuid": checklistUUID, "deleted": true}, nil
}

func DeleteStoryChecklistItem(c *client.Client, boardUUID, taskKey, checklistUUID, itemUUID string) (map[string]any, error) {
	_, st, err := storyUUID(c, boardUUID, taskKey)
	if err != nil {
		return nil, err
	}
	api, _, err := c.DeleteJSON("/tasks/"+st.TaskUUID+"/checklists/"+checklistUUID+"/items/"+itemUUID, map[string]any{})
	if err := checkAPI(api, err, "删除检查项"); err != nil {
		return nil, err
	}
	return map[string]any{
		"task_uuid":      st.TaskUUID,
		"checklist_uuid": checklistUUID,
		"item_uuid":      itemUUID,
		"deleted":        true,
	}, nil
}

func SetStoryDone(c *client.Client, boardUUID, taskKey string, done bool) (map[string]any, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	st, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	status := deadlineNotFinished
	if done {
		status = deadlineFinished
	}
	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("task_id", fmt.Sprint(st.TaskID))
	form.Set("dead_line_status", strconv.Itoa(status))
	api, _, err := c.PostForm("/task/saveDeadlineStatus", form)
	if err := checkAPI(api, err, "设置完成状态"); err != nil {
		return nil, err
	}
	return map[string]any{"task_id": st.TaskID, "done": done, "dead_line_status": status}, nil
}

func SetStoryType(c *client.Client, boardUUID, taskKey, typeID string) (map[string]any, error) {
	typeID = strings.TrimSpace(typeID)
	if typeID == "" {
		return nil, fmt.Errorf("请指定 --type-id")
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
	form.Set("type_id", typeID)
	if st.CardTypeName != "" {
		form.Set("old_type_name", st.CardTypeName)
	}
	api, _, err := c.PostForm("/task/fields/change_card_type", form)
	if err := checkAPI(api, err, "修改卡片类型"); err != nil {
		return nil, err
	}
	var msg any
	_ = json.Unmarshal(api.Message, &msg)
	return map[string]any{"task_id": st.TaskID, "type_id": typeID, "message": msg}, nil
}

func SetStoryActualHours(c *client.Client, boardUUID, taskKey, hours string) (map[string]any, error) {
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
	hours = strings.TrimSpace(hours)
	if hours != "" {
		form.Set("working_hour", hours)
	}
	api, _, err := c.PostForm("/task/actual_working_hour/set", form)
	if err := checkAPI(api, err, "设置实际工时"); err != nil {
		return nil, err
	}
	return map[string]any{"task_id": st.TaskID, "working_hour": hours}, nil
}

func CreateStoryShortcut(c *client.Client, fromBoardUUID, taskKey, toBoardUUID, toList string) (map[string]any, error) {
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
	api, _, err := c.PostForm("/task/create_shortcut1", form)
	if err := checkAPI(api, err, "创建引用"); err != nil {
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
