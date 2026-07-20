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

func RefreshEnterprises(c *client.Client) ([]session.Ent, *session.Ent, error) {
	home := c.Session.HomeURL
	if c.Session.CurrentEnt != nil && c.Session.CurrentEnt.HomeURL != "" {
		home = c.Session.CurrentEnt.HomeURL
	}
	if home == "" {
		home = parse.PersonalHomeURL()
	}
	html, err := c.GetHTML(home)
	if err != nil {
		// try relative path
		html, err = c.GetHTML(strings.TrimPrefix(home, "https://www.lg.team"))
		if err != nil {
			return nil, nil, err
		}
	}
	ents := parse.ParseEnterprises(html)
	cur := parse.CurrentEntFromPage(html, home)

	// Enrich javascript: entries that only have names by matching current / known URLs.
	for i := range ents {
		if ents[i].HomeURL == "" {
			if ents[i].ID == -1 {
				ents[i].HomeURL = parse.PersonalHomeURL()
			}
		}
		// If name matches current button and we have sign from LG.ent
		if cur != nil && ents[i].Name == cur.Name && ents[i].Sign == "" {
			ents[i].ID = cur.ID
			ents[i].Sign = cur.Sign
			ents[i].HomeURL = cur.HomeURL
		}
	}

	// Deduplicate by ID keeping richest entry.
	byID := map[int64]session.Ent{}
	order := []int64{}
	for _, e := range ents {
		prev, ok := byID[e.ID]
		if !ok {
			order = append(order, e.ID)
			byID[e.ID] = e
			continue
		}
		if e.Sign != "" && prev.Sign == "" {
			byID[e.ID] = e
		}
		if e.HomeURL != "" && prev.HomeURL == "" {
			merged := byID[e.ID]
			merged.HomeURL = e.HomeURL
			byID[e.ID] = merged
		}
	}
	out := make([]session.Ent, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}

	c.Session.Ents = out
	if cur != nil {
		c.Session.CurrentEnt = cur
	}
	if u := parse.ExtractNewLeangooWebURL(html); u != "" {
		c.Session.NewLeangooWebURL = u
	}
	c.PersistCookies(home)
	if err := session.Save(c.Session); err != nil {
		return out, cur, err
	}
	return out, cur, nil
}

func UseEnterprise(c *client.Client, idOrName string) (*session.Ent, error) {
	ents := c.Session.Ents
	if len(ents) == 0 {
		var err error
		ents, _, err = RefreshEnterprises(c)
		if err != nil {
			return nil, err
		}
	}
	var target *session.Ent
	if id, err := strconv.ParseInt(idOrName, 10, 64); err == nil {
		for i := range ents {
			if ents[i].ID == id {
				target = &ents[i]
				break
			}
		}
	}
	if target == nil {
		for i := range ents {
			if ents[i].Name == idOrName || strings.Contains(ents[i].Name, idOrName) {
				target = &ents[i]
				break
			}
		}
	}
	if target == nil {
		return nil, fmt.Errorf("未找到企业: %s", idOrName)
	}
	home := target.HomeURL
	if home == "" {
		if target.ID == -1 {
			home = parse.PersonalHomeURL()
		} else if target.Sign != "" {
			home = fmt.Sprintf("https://www.lg.team/kanban/ent/%d/%s", target.ID, target.Sign)
		} else {
			return nil, fmt.Errorf("企业缺少入口 URL，请先 leangoo ent list")
		}
		target.HomeURL = home
	}
	if _, err := c.HTTP.Get(home); err != nil {
		return nil, fmt.Errorf("切换企业失败: %w", err)
	}
	// Re-parse after switch to get accurate sign/name.
	html, err := c.GetHTML(home)
	if err == nil {
		cur := parse.CurrentEntFromPage(html, home)
		if cur != nil {
			target = cur
		}
		ents2 := parse.ParseEnterprises(html)
		if len(ents2) > 0 {
			c.Session.Ents = ents2
		}
	}
	c.Session.CurrentEnt = target
	c.Session.HomeURL = target.HomeURL
	c.PersistCookies(target.HomeURL)
	if err := session.Save(c.Session); err != nil {
		return target, err
	}
	return target, nil
}

type Project struct {
	ID   json.RawMessage `json:"project_id"`
	UUID string          `json:"project_uuid"`
	Name string          `json:"project_name"`
	Raw  json.RawMessage `json:"-"`
}

func ListProjects(c *client.Client) ([]json.RawMessage, error) {
	ent, err := c.Session.RequireEnt()
	if err != nil {
		return nil, err
	}
	if ent.ID == -1 {
		// Personal / 团队版: use getTargetPage2 with ent_id=-1 or board_list home data.
		form := url.Values{}
		form.Set("type", "project")
		form.Set("ent_id", "-1")
		api, raw, err := c.PostForm("/ent/getTargetPage2", form)
		if err != nil {
			return nil, err
		}
		if api.OK() {
			return extractProjectsFromTargetPage(api.Message)
		}
		// Fallback: privilege API may still work for some accounts
		_ = raw
	}
	api, _, err := c.Get("/project/get_have_view_privilege_projects/"+strconv.FormatInt(ent.ID, 10), nil)
	if err != nil {
		return nil, err
	}
	if !api.OK() {
		// try getTargetPage2
		form := url.Values{}
		form.Set("type", "project")
		form.Set("ent_id", strconv.FormatInt(ent.ID, 10))
		api2, _, err2 := c.PostForm("/ent/getTargetPage2", form)
		if err2 != nil {
			return nil, fmt.Errorf("获取项目失败: %s", api.MessageString())
		}
		if !api2.OK() {
			return nil, fmt.Errorf("获取项目失败: %s", api2.MessageString())
		}
		return extractProjectsFromTargetPage(api2.Message)
	}
	var list []json.RawMessage
	if err := json.Unmarshal(api.Message, &list); err != nil {
		// message may be object wrapping list
		var wrap map[string]json.RawMessage
		if err2 := json.Unmarshal(api.Message, &wrap); err2 == nil {
			for _, k := range []string{"projects", "project_list", "data", "list"} {
				if v, ok := wrap[k]; ok {
					if err3 := json.Unmarshal(v, &list); err3 == nil {
						return list, nil
					}
				}
			}
		}
		return []json.RawMessage{api.Message}, nil
	}
	return list, nil
}

func extractProjectsFromTargetPage(message json.RawMessage) ([]json.RawMessage, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(message, &root); err != nil {
		return nil, err
	}
	if data, ok := root["data"]; ok {
		var dataObj map[string]json.RawMessage
		if err := json.Unmarshal(data, &dataObj); err == nil {
			for _, k := range []string{"projects", "project_list", "list"} {
				if v, ok := dataObj[k]; ok {
					var list []json.RawMessage
					if err := json.Unmarshal(v, &list); err == nil {
						return list, nil
					}
				}
			}
			// data itself may be array
			var list []json.RawMessage
			if err := json.Unmarshal(data, &list); err == nil {
				return list, nil
			}
			return []json.RawMessage{data}, nil
		}
		var list []json.RawMessage
		if err := json.Unmarshal(data, &list); err == nil {
			return list, nil
		}
	}
	for _, k := range []string{"projects", "project_list"} {
		if v, ok := root[k]; ok {
			var list []json.RawMessage
			if err := json.Unmarshal(v, &list); err == nil {
				return list, nil
			}
		}
	}
	return []json.RawMessage{message}, nil
}

func ListBoards(c *client.Client, projectID string) (json.RawMessage, error) {
	api, _, err := c.Get("/project/get_project_boards/"+projectID, nil)
	if err != nil {
		return nil, err
	}
	if !api.OK() {
		return nil, fmt.Errorf("获取 Sprint/看板失败: %s", api.MessageString())
	}
	return api.Message, nil
}
