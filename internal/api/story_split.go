package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/client"
)

type SplitStoryOpts struct {
	FEEstimate string // e.g. "0.5"; empty → try parse from description/comments
	BEEstimate string
	FETag      string // default 应用前端
	BETag      string // default 应用后端
	ToList     string // default: same list as original
}

type SplitStoryResult struct {
	OK         bool           `json:"ok"`
	BoardUUID  string         `json:"board_uuid"`
	Original   map[string]any `json:"original"`
	Frontend   map[string]any `json:"frontend"`
	Backend    map[string]any `json:"backend"`
	FEEstimate string         `json:"fe_estimate"`
	BEEstimate string         `json:"be_estimate"`
	FETag      string         `json:"fe_tag"`
	BETag      string         `json:"be_tag"`
	Steps      []string       `json:"steps"`
}

// SplitStoryFEBE copies a product story into FE/BE cards, adjusts tags/estimates,
// zeros the original estimate, interconnects, and reorders: original → FE → BE.
func SplitStoryFEBE(c *client.Client, boardUUID, taskKey string, opts SplitStoryOpts) (*SplitStoryResult, error) {
	feTag := strings.TrimSpace(opts.FETag)
	beTag := strings.TrimSpace(opts.BETag)
	if feTag == "" {
		feTag = "应用前端"
	}
	if beTag == "" {
		beTag = "应用后端"
	}

	ctx, stories, err := ListStories(c, boardUUID)
	if err != nil {
		return nil, err
	}
	orig, err := FindStory(stories, taskKey)
	if err != nil {
		return nil, err
	}
	listID := fmt.Sprint(orig.ListID)
	toList := strings.TrimSpace(opts.ToList)
	if toList == "" {
		if ld, err := ResolveList(ctx, listID); err == nil {
			toList = ld.ListName
		} else {
			toList = listID
		}
	}

	detail, err := GetStory(c, boardUUID, fmt.Sprint(orig.TaskID))
	if err != nil {
		return nil, fmt.Errorf("读取原卡详情: %w", err)
	}
	feEst := strings.TrimSpace(opts.FEEstimate)
	beEst := strings.TrimSpace(opts.BEEstimate)
	if feEst == "" || beEst == "" {
		parsedFE, parsedBE := parseFEBEEstimates(detail.Description)
		if feEst == "" {
			feEst = parsedFE
		}
		if beEst == "" {
			beEst = parsedBE
		}
	}
	if feEst == "" || beEst == "" {
		if comments, err := ListComments(c, boardUUID, fmt.Sprint(orig.TaskID)); err == nil {
			text := commentsText(comments)
			parsedFE, parsedBE := parseFEBEEstimates(text)
			if feEst == "" {
				feEst = parsedFE
			}
			if beEst == "" {
				beEst = parsedBE
			}
		}
	}
	if feEst == "" || beEst == "" {
		return nil, fmt.Errorf("无法确定前后端点数：请传 --fe-estimate / --be-estimate，或在描述/评论写明「前：…（n）」「后：…（n）」")
	}

	steps := []string{}
	step := func(s string) { steps = append(steps, s) }

	// Ensure tags exist on board and on original card.
	if _, err := EnsureBoardTag(c, boardUUID, feTag); err != nil {
		return nil, fmt.Errorf("确保前端标签: %w", err)
	}
	if _, err := EnsureBoardTag(c, boardUUID, beTag); err != nil {
		return nil, fmt.Errorf("确保后端标签: %w", err)
	}
	step(fmt.Sprintf("ensure tags: %s, %s", feTag, beTag))

	if !storyHasTag(*orig, feTag) {
		if _, err := AddStoryTagByQueryOpts(c, boardUUID, fmt.Sprint(orig.TaskID), feTag, true); err != nil {
			return nil, fmt.Errorf("原卡加前端标签: %w", err)
		}
		step("original tag-add " + feTag)
	}
	if !storyHasTag(*orig, beTag) {
		if _, err := AddStoryTagByQueryOpts(c, boardUUID, fmt.Sprint(orig.TaskID), beTag, true); err != nil {
			return nil, fmt.Errorf("原卡加后端标签: %w", err)
		}
		step("original tag-add " + beTag)
	}

	feCopy, err := CopyStory(c, boardUUID, fmt.Sprint(orig.TaskID), boardUUID, toList)
	if err != nil {
		return nil, fmt.Errorf("复制前端卡: %w", err)
	}
	feID, err := extractCopiedTaskID(feCopy)
	if err != nil {
		return nil, fmt.Errorf("解析前端副本 id: %w", err)
	}
	step("copy frontend → " + feID)

	beCopy, err := CopyStory(c, boardUUID, fmt.Sprint(orig.TaskID), boardUUID, toList)
	if err != nil {
		return nil, fmt.Errorf("复制后端卡: %w", err)
	}
	beID, err := extractCopiedTaskID(beCopy)
	if err != nil {
		return nil, fmt.Errorf("解析后端副本 id: %w", err)
	}
	step("copy backend → " + beID)

	// FE: keep feTag, remove beTag
	if _, err := DeleteStoryTagByQuery(c, boardUUID, feID, beTag); err != nil {
		return nil, fmt.Errorf("前端卡去掉后端标签: %w", err)
	}
	feEstPtr := feEst
	if _, err := EditStory(c, boardUUID, feID, EditStoryOpts{Estimate: &feEstPtr}); err != nil {
		return nil, fmt.Errorf("前端卡改点数: %w", err)
	}
	step(fmt.Sprintf("frontend tags/estimate=%s", feEst))

	// BE: keep beTag, remove feTag
	if _, err := DeleteStoryTagByQuery(c, boardUUID, beID, feTag); err != nil {
		return nil, fmt.Errorf("后端卡去掉前端标签: %w", err)
	}
	beEstPtr := beEst
	if _, err := EditStory(c, boardUUID, beID, EditStoryOpts{Estimate: &beEstPtr}); err != nil {
		return nil, fmt.Errorf("后端卡改点数: %w", err)
	}
	step(fmt.Sprintf("backend tags/estimate=%s", beEst))

	zero := "0"
	if _, err := EditStory(c, boardUUID, fmt.Sprint(orig.TaskID), EditStoryOpts{Estimate: &zero}); err != nil {
		return nil, fmt.Errorf("总卡清零点数: %w", err)
	}
	step("original estimate=0")

	if _, err := AddStoryCardConnect(c, boardUUID, fmt.Sprint(orig.TaskID), feID); err != nil {
		return nil, fmt.Errorf("关联前端卡: %w", err)
	}
	if _, err := AddStoryCardConnect(c, boardUUID, fmt.Sprint(orig.TaskID), beID); err != nil {
		return nil, fmt.Errorf("关联后端卡: %w", err)
	}
	step("connect + interconnect")

	if _, err := ReorderStories(c, boardUUID, []string{fmt.Sprint(orig.TaskID), feID, beID}); err != nil {
		return nil, fmt.Errorf("调整顺序: %w", err)
	}
	step("reorder original → frontend → backend")

	// Refresh summaries
	_, stories, _ = ListStories(c, boardUUID)
	orig2, _ := FindStory(stories, fmt.Sprint(orig.TaskID))
	fe2, _ := FindStory(stories, feID)
	be2, _ := FindStory(stories, beID)

	return &SplitStoryResult{
		OK:         true,
		BoardUUID:  boardUUID,
		Original:   storySummary(boardUUID, orig2, orig),
		Frontend:   storySummary(boardUUID, fe2, nil),
		Backend:    storySummary(boardUUID, be2, nil),
		FEEstimate: feEst,
		BEEstimate: beEst,
		FETag:      feTag,
		BETag:      beTag,
		Steps:      steps,
	}, nil
}

func storySummary(boardUUID string, st *Story, fallback *Story) map[string]any {
	if st == nil {
		st = fallback
	}
	if st == nil {
		return nil
	}
	tags := make([]string, 0, len(st.Tags))
	for _, t := range st.Tags {
		if t.TagName != "" {
			tags = append(tags, t.TagName)
		}
	}
	url := ""
	if st.TaskUUID != "" && boardUUID != "" {
		url = "https://www.lg.team/kanban/board/go/" + boardUUID + "/" + st.TaskUUID
	}
	return map[string]any{
		"task_id":   st.TaskID,
		"task_uuid": st.TaskUUID,
		"task_name": st.TaskName,
		"estimate":  st.Estimate,
		"tags":      tags,
		"list_id":   st.ListID,
		"url":       url,
	}
}

func extractCopiedTaskID(copyOut map[string]any) (string, error) {
	msg := copyOut["message"]
	b, _ := json.Marshal(msg)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m == nil {
		return "", fmt.Errorf("复制返回无 message")
	}
	arr, _ := m["succeed_tasks"].([]any)
	if len(arr) == 0 {
		return "", fmt.Errorf("复制返回无 succeed_tasks")
	}
	first, _ := arr[0].(map[string]any)
	if first == nil {
		return "", fmt.Errorf("succeed_tasks[0] 格式异常")
	}
	id := fmt.Sprint(first["task_id"])
	if id == "" || id == "<nil>" {
		return "", fmt.Errorf("succeed_tasks[0] 无 task_id")
	}
	return id, nil
}

var (
	reFEEst = []*regexp.Regexp{
		regexp.MustCompile(`(?i)前(?:端)?\s*[:：]\s*[^\n（(]{0,80}[（(]\s*(\d+(?:\.\d+)?)\s*[）)]`),
		regexp.MustCompile(`(?i)前(?:端)?\s*[:：]\s*(\d+(?:\.\d+)?)`),
		regexp.MustCompile(`(?i)前端[^0-9]{0,20}(\d+(?:\.\d+)?)`),
	}
	reBEEst = []*regexp.Regexp{
		regexp.MustCompile(`(?i)后(?:端)?\s*[:：]\s*[^\n（(]{0,80}[（(]\s*(\d+(?:\.\d+)?)\s*[）)]`),
		regexp.MustCompile(`(?i)后(?:端)?\s*[:：]\s*(\d+(?:\.\d+)?)`),
		regexp.MustCompile(`(?i)后端[^0-9]{0,20}(\d+(?:\.\d+)?)`),
	}
)

func parseFEBEEstimates(text string) (fe, be string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}
	for _, re := range reFEEst {
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			if _, err := strconv.ParseFloat(m[1], 64); err == nil {
				fe = m[1]
				break
			}
		}
	}
	for _, re := range reBEEst {
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			if _, err := strconv.ParseFloat(m[1], 64); err == nil {
				be = m[1]
				break
			}
		}
	}
	return fe, be
}

func commentsText(out map[string]any) string {
	raw := out["comments"]
	b, _ := json.Marshal(raw)
	var arr []map[string]any
	if err := json.Unmarshal(b, &arr); err != nil {
		return string(b)
	}
	var parts []string
	for _, c := range arr {
		for _, k := range []string{"content", "comment", "text", "message"} {
			if s := strings.TrimSpace(fmt.Sprint(c[k])); s != "" && s != "<nil>" {
				parts = append(parts, s)
				break
			}
		}
	}
	return strings.Join(parts, "\n")
}
