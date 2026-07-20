package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/client"
	"github.com/deepglint/leangoo-cli/internal/parse"
	"github.com/deepglint/leangoo-cli/internal/session"
)

type BoardContext struct {
	BoardUUID   string          `json:"board_uuid"`
	BoardID     string          `json:"board_id"`
	BoardData   json.RawMessage `json:"board_data,omitempty"`
	GBoard      json.RawMessage `json:"g_board,omitempty"`
	Lists       json.RawMessage `json:"lists,omitempty"`
	Lanes       json.RawMessage `json:"lanes,omitempty"`
	LaneTimes   int             `json:"lanes_times"`
	Structure   BoardStructure  `json:"structure"`
	Users       []BoardUser     `json:"users,omitempty"`
	CurrentUser *BoardUser      `json:"current_user,omitempty"`
}

type BoardStructure struct {
	LaneCount int      `json:"lane_count"`
	ListCount int      `json:"list_count"`
	LaneNames []string `json:"lane_names,omitempty"`
	ListNames []string `json:"list_names,omitempty"`
}

// BoardUser is a member from board_data.users / current_user.
type BoardUser struct {
	ID       string `json:"id"`
	Email    string `json:"email,omitempty"`
	NickName string `json:"nick_name,omitempty"`
}

type Story struct {
	TaskID       any             `json:"task_id"`
	TaskUUID     string          `json:"task_uuid,omitempty"`
	TaskName     string          `json:"task_name"`
	Estimate     any             `json:"estimate"`
	LinkedUsers  any             `json:"linked_users"`
	Members      []BoardUser     `json:"members,omitempty"`
	Tags         []StoryTag      `json:"tags,omitempty"`
	HasDesc      any             `json:"has_desc"`
	LaneID       any             `json:"lane_id,omitempty"`
	ListID       any             `json:"list_id,omitempty"`
	CardTypeName string          `json:"card_type_name,omitempty"`
	Raw          json.RawMessage `json:"raw,omitempty"`
}

// StoryTag is a label on a card (from task.tags).
type StoryTag struct {
	TagUUID   string `json:"tag_uuid,omitempty"`
	TagName   string `json:"tag_name,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	GroupUUID string `json:"group_uuid,omitempty"`
	Color     string `json:"color_value,omitempty"`
}

type StoryDetail struct {
	Story       Story           `json:"story"`
	Description string          `json:"description"`
	Activities  json.RawMessage `json:"activities"`
	BoardID     string          `json:"board_id"`
}

type StoryListOptions struct {
	// User filters by id / email / nick_name, or "me"/"current" for current_user.
	User string
	// Tags filters by tag_name (substring) or tag_uuid. Multiple tags = AND.
	Tags []string
}

func LoadBoardContext(c *client.Client, boardUUID string) (*BoardContext, error) {
	path := "/board/go/" + boardUUID
	html, err := c.GetHTML(path)
	if err != nil {
		return nil, err
	}
	bd, err := parse.ExtractJSObject(html, "board_data")
	if err != nil {
		return nil, fmt.Errorf("解析 board_data: %w", err)
	}
	gb, _ := parse.ExtractJSObject(html, "g_board")
	boardID := parse.ExtractBoardID(html)

	var bdObj map[string]json.RawMessage
	_ = json.Unmarshal(bd, &bdObj)

	ctx := &BoardContext{
		BoardUUID: boardUUID,
		BoardID:   boardID,
		BoardData: bd,
		GBoard:    gb,
	}
	if boardID == "" {
		if b, ok := bdObj["board"]; ok {
			var board map[string]any
			if json.Unmarshal(b, &board) == nil {
				ctx.BoardID = fmt.Sprint(board["board_id"])
			}
		}
	}
	if u := parse.ExtractNewLeangooWebURL(html); u != "" {
		c.Session.NewLeangooWebURL = u
		_ = session.Save(c.Session)
	}

	if v, ok := bdObj["lists"]; ok {
		ctx.Lists = v
		ctx.Structure.ListCount, ctx.Structure.ListNames = countNamed(v, "list_name", "name")
	}
	if v, ok := bdObj["lanes"]; ok {
		ctx.Lanes = v
		ctx.Structure.LaneCount, ctx.Structure.LaneNames = countNamed(v, "lane_name", "name")
	}
	if v, ok := bdObj["lanes_times"]; ok {
		var n json.Number
		if json.Unmarshal(v, &n) == nil {
			i, _ := n.Int64()
			ctx.LaneTimes = int(i)
		} else {
			var f float64
			if json.Unmarshal(v, &f) == nil {
				ctx.LaneTimes = int(f)
			}
		}
	}
	if ctx.LaneTimes <= 0 {
		ctx.LaneTimes = 1
	}
	ctx.Users, ctx.CurrentUser = extractBoardUsers(bdObj, gb)
	c.PersistCookies(c.URL(path))
	_ = session.Save(c.Session)
	return ctx, nil
}

func extractBoardUsers(bdObj map[string]json.RawMessage, gBoard json.RawMessage) ([]BoardUser, *BoardUser) {
	var users []BoardUser
	if raw, ok := bdObj["users"]; ok {
		users = parseUserList(raw)
	}
	if len(users) == 0 && len(gBoard) > 0 {
		var gb map[string]json.RawMessage
		if json.Unmarshal(gBoard, &gb) == nil {
			if raw, ok := gb["users"]; ok {
				users = parseUserList(raw)
			}
		}
	}

	var current *BoardUser
	if raw, ok := bdObj["current_user"]; ok {
		current = parseCurrentUser(raw, users)
	}
	return users, current
}

func parseUserList(raw json.RawMessage) []BoardUser {
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil
	}
	out := make([]BoardUser, 0, len(arr))
	for _, u := range arr {
		out = append(out, BoardUser{
			ID:       fmt.Sprint(u["id"]),
			Email:    fmt.Sprint(nilToEmpty(u["email"])),
			NickName: fmt.Sprint(nilToEmpty(u["nick_name"])),
		})
	}
	return out
}

func parseCurrentUser(raw json.RawMessage, users []BoardUser) *BoardUser {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	id := firstString(m, "current_user_id", "id", "user_id")
	email := firstString(m, "current_user_email", "email")
	name := firstString(m, "current_user_name", "nick_name", "user_name")
	if id == "" && email == "" && name == "" {
		return nil
	}
	cu := &BoardUser{ID: id, Email: email, NickName: name}
	// Prefer full profile from users list when id matches.
	for _, u := range users {
		if id != "" && u.ID == id {
			if cu.Email == "" {
				cu.Email = u.Email
			}
			if cu.NickName == "" {
				cu.NickName = u.NickName
			}
			break
		}
	}
	return cu
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func nilToEmpty(v any) any {
	if v == nil {
		return ""
	}
	return v
}

func countNamed(raw json.RawMessage, keys ...string) (int, []string) {
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return 0, nil
	}
	names := make([]string, 0, len(arr))
	for _, item := range arr {
		name := ""
		for _, k := range keys {
			if v, ok := item[k]; ok {
				name = fmt.Sprint(v)
				break
			}
		}
		names = append(names, name)
	}
	return len(arr), names
}

func ListStories(c *client.Client, boardUUID string) (*BoardContext, []Story, error) {
	return ListStoriesOpts(c, boardUUID, StoryListOptions{})
}

func ListStoriesOpts(c *client.Client, boardUUID string, opts StoryListOptions) (*BoardContext, []Story, error) {
	ctx, err := LoadBoardContext(c, boardUUID)
	if err != nil {
		return nil, nil, err
	}
	if ctx.BoardID == "" {
		return ctx, nil, fmt.Errorf("无法从看板页解析 board_id")
	}

	var allLanes []json.RawMessage
	// If lists empty, lanes may already contain tasks.
	if len(ctx.Lists) == 0 || string(ctx.Lists) == "null" || string(ctx.Lists) == "[]" {
		if len(ctx.Lanes) > 0 {
			_ = json.Unmarshal(ctx.Lanes, &allLanes)
		}
	} else {
		for i := 0; i < ctx.LaneTimes; i++ {
			form := url.Values{}
			form.Set("board_id", ctx.BoardID)
			form.Set("current_times", strconv.Itoa(i))
			api, _, err := c.PostForm("/board/getLaneTasks", form)
			if err != nil {
				return ctx, nil, err
			}
			if !api.OK() {
				return ctx, nil, fmt.Errorf("getLaneTasks 失败: %s", api.MessageString())
			}
			var msg struct {
				Lanes []json.RawMessage `json:"lanes"`
			}
			if err := json.Unmarshal(api.Message, &msg); err != nil {
				var lanes []json.RawMessage
				if err2 := json.Unmarshal(api.Message, &lanes); err2 != nil {
					return ctx, nil, fmt.Errorf("解析 getLaneTasks: %w", err)
				}
				allLanes = append(allLanes, lanes...)
			} else {
				allLanes = append(allLanes, msg.Lanes...)
			}
		}
	}

	stories := extractStories(allLanes)
	enrichStoryMembers(stories, ctx.Users, ctx.CurrentUser)

	if opts.User != "" {
		uid, matched, err := ResolveBoardUser(opts.User, ctx)
		if err != nil {
			return ctx, nil, err
		}
		filtered := make([]Story, 0, len(stories))
		for _, s := range stories {
			if storyHasUser(s, uid) {
				filtered = append(filtered, s)
			}
		}
		stories = filtered
		_ = matched
	}

	if len(opts.Tags) > 0 {
		filtered := make([]Story, 0, len(stories))
		for _, s := range stories {
			if storyHasAllTags(s, opts.Tags) {
				filtered = append(filtered, s)
			}
		}
		stories = filtered
	}

	return ctx, stories, nil
}

func ResolveBoardUser(query string, ctx *BoardContext) (string, *BoardUser, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", nil, fmt.Errorf("用户不能为空")
	}
	lower := strings.ToLower(q)
	if lower == "me" || lower == "current" || lower == "current_user" || q == "当前用户" {
		if ctx.CurrentUser == nil || ctx.CurrentUser.ID == "" {
			return "", nil, fmt.Errorf("看板页未找到 current_user")
		}
		return ctx.CurrentUser.ID, ctx.CurrentUser, nil
	}

	// Exact id
	for i := range ctx.Users {
		u := &ctx.Users[i]
		if u.ID == q {
			return u.ID, u, nil
		}
	}
	if ctx.CurrentUser != nil && ctx.CurrentUser.ID == q {
		return ctx.CurrentUser.ID, ctx.CurrentUser, nil
	}
	// Exact email / nick (users + current_user)
	candidates := make([]*BoardUser, 0, len(ctx.Users)+1)
	for i := range ctx.Users {
		candidates = append(candidates, &ctx.Users[i])
	}
	if ctx.CurrentUser != nil {
		candidates = append(candidates, ctx.CurrentUser)
	}
	for _, u := range candidates {
		if strings.EqualFold(u.Email, q) || strings.EqualFold(u.NickName, q) {
			return u.ID, u, nil
		}
	}
	// Partial nick / email
	var hits []BoardUser
	seen := map[string]struct{}{}
	for _, u := range candidates {
		if u.ID == "" {
			continue
		}
		if strings.Contains(strings.ToLower(u.NickName), lower) || strings.Contains(strings.ToLower(u.Email), lower) {
			if _, ok := seen[u.ID]; ok {
				continue
			}
			seen[u.ID] = struct{}{}
			hits = append(hits, *u)
		}
	}
	if len(hits) == 1 {
		return hits[0].ID, &hits[0], nil
	}
	if len(hits) > 1 {
		names := make([]string, 0, len(hits))
		for _, h := range hits {
			names = append(names, fmt.Sprintf("%s(%s)", h.NickName, h.ID))
		}
		return "", nil, fmt.Errorf("匹配到多个用户，请用 id 或更精确名称: %s", strings.Join(names, ", "))
	}
	// Allow raw numeric id even if not in users list (still on cards).
	if _, err := strconv.ParseInt(q, 10, 64); err == nil {
		return q, &BoardUser{ID: q}, nil
	}
	return "", nil, fmt.Errorf("未找到用户: %s", query)
}

func storyHasUser(s Story, userID string) bool {
	ids := linkedUserIDs(s.LinkedUsers)
	for _, id := range ids {
		if id == userID {
			return true
		}
	}
	return false
}

func parseStoryTags(v any) []StoryTag {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil
	}
	out := make([]StoryTag, 0, len(arr))
	for _, t := range arr {
		tag := StoryTag{
			TagUUID:   firstString(t, "tag_uuid", "uuid", "id"),
			TagName:   firstString(t, "tag_name", "name", "default_tag_name"),
			GroupName: firstString(t, "group_name"),
			GroupUUID: firstString(t, "group_uuid"),
			Color:     firstString(t, "color_value", "color"),
		}
		if tag.TagUUID == "" && tag.TagName == "" {
			continue
		}
		out = append(out, tag)
	}
	return out
}

// storyHasAllTags requires every query to match at least one tag (AND).
// Each query matches tag_uuid exactly, or tag_name case-insensitive substring.
func storyHasAllTags(s Story, queries []string) bool {
	if len(queries) == 0 {
		return true
	}
	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		if !storyHasTag(s, q) {
			return false
		}
	}
	return true
}

func storyHasTag(s Story, query string) bool {
	lower := strings.ToLower(strings.TrimSpace(query))
	for _, tag := range s.Tags {
		if tag.TagUUID != "" && (tag.TagUUID == query || strings.EqualFold(tag.TagUUID, query)) {
			return true
		}
		if tag.TagName != "" && (strings.EqualFold(tag.TagName, query) || strings.Contains(strings.ToLower(tag.TagName), lower)) {
			return true
		}
		if tag.GroupName != "" && strings.Contains(strings.ToLower(tag.GroupName), lower) {
			return true
		}
	}
	return false
}

// CollectStoryTags returns unique tags across stories (for story tags command).
func CollectStoryTags(stories []Story) []StoryTag {
	seen := map[string]StoryTag{}
	order := []string{}
	for _, s := range stories {
		for _, t := range s.Tags {
			key := t.TagUUID
			if key == "" {
				key = "name:" + t.TagName
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = t
			order = append(order, key)
		}
	}
	out := make([]StoryTag, 0, len(order))
	for _, k := range order {
		out = append(out, seen[k])
	}
	return out
}

func linkedUserIDs(v any) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		parts := strings.Split(t, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			out = append(out, fmt.Sprint(x))
		}
		return out
	case []string:
		return t
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return nil
		}
		return linkedUserIDs(s)
	}
}

func enrichStoryMembers(stories []Story, users []BoardUser, current *BoardUser) {
	byID := map[string]BoardUser{}
	for _, u := range users {
		byID[u.ID] = u
	}
	if current != nil && current.ID != "" {
		if prev, ok := byID[current.ID]; ok {
			if prev.NickName == "" {
				prev.NickName = current.NickName
			}
			if prev.Email == "" {
				prev.Email = current.Email
			}
			byID[current.ID] = prev
		} else {
			byID[current.ID] = *current
		}
	}
	for i := range stories {
		ids := linkedUserIDs(stories[i].LinkedUsers)
		if len(ids) == 0 {
			continue
		}
		members := make([]BoardUser, 0, len(ids))
		for _, id := range ids {
			if u, ok := byID[id]; ok {
				members = append(members, u)
			} else {
				members = append(members, BoardUser{ID: id})
			}
		}
		stories[i].Members = members
	}
}

func extractStories(lanes []json.RawMessage) []Story {
	var out []Story
	for _, laneRaw := range lanes {
		var lane map[string]json.RawMessage
		if err := json.Unmarshal(laneRaw, &lane); err != nil {
			continue
		}
		laneID := lane["lane_id"]
		blocksRaw, ok := lane["blocks"]
		if !ok {
			// tasks directly on lane?
			if tasksRaw, ok := lane["tasks"]; ok {
				out = append(out, storiesFromTasks(tasksRaw, laneID, nil)...)
			}
			continue
		}
		var blocks []map[string]json.RawMessage
		if err := json.Unmarshal(blocksRaw, &blocks); err != nil {
			continue
		}
		for _, block := range blocks {
			listID := block["list_id"]
			if tasksRaw, ok := block["tasks"]; ok {
				out = append(out, storiesFromTasks(tasksRaw, laneID, listID)...)
			}
		}
	}
	return out
}

func storiesFromTasks(tasksRaw json.RawMessage, laneID, listID json.RawMessage) []Story {
	var tasks []map[string]any
	if err := json.Unmarshal(tasksRaw, &tasks); err != nil {
		return nil
	}
	out := make([]Story, 0, len(tasks))
	for _, t := range tasks {
		raw, _ := json.Marshal(t)
		s := Story{
			TaskID:   t["task_id"],
			TaskName: fmt.Sprint(t["task_name"]),
			Estimate: t["estimate"],
			HasDesc:  t["has_desc"],
			Raw:      raw,
		}
		if v, ok := t["task_uuid"]; ok {
			s.TaskUUID = fmt.Sprint(v)
		}
		if v, ok := t["linked_users"]; ok {
			s.LinkedUsers = v
		}
		if v, ok := t["tags"]; ok {
			s.Tags = parseStoryTags(v)
		}
		if v, ok := t["card_type_name"]; ok {
			s.CardTypeName = fmt.Sprint(v)
		}
		if laneID != nil {
			_ = json.Unmarshal(laneID, &s.LaneID)
		}
		if listID != nil {
			_ = json.Unmarshal(listID, &s.ListID)
		}
		out = append(out, s)
	}
	return out
}

func GetStory(c *client.Client, boardUUID, taskID string) (*StoryDetail, error) {
	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	var found *Story
	for i := range stories {
		id := fmt.Sprint(stories[i].TaskID)
		if id == taskID || stories[i].TaskUUID == taskID || strings.EqualFold(stories[i].TaskName, taskID) {
			found = &stories[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("未找到 story: %s", taskID)
	}

	detail := &StoryDetail{Story: *found, BoardID: ctx.BoardID}

	q := url.Values{}
	q.Set("task_id", fmt.Sprint(found.TaskID))
	q.Set("board_id", ctx.BoardID)
	api, _, err := c.Get("/task/getTaskDesc", q)
	if err == nil && api.OK() {
		var desc string
		if err := json.Unmarshal(api.Message, &desc); err == nil {
			detail.Description = desc
		} else {
			detail.Description = api.MessageString()
		}
	}

	form := url.Values{}
	form.Set("board_id", ctx.BoardID)
	form.Set("task_id", fmt.Sprint(found.TaskID))
	api2, _, err := c.PostForm("/task/getTaskActivity", form)
	if err == nil && api2.OK() {
		detail.Activities = api2.Message
	}

	return detail, nil
}
