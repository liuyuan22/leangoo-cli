package parse

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/deepglint/leangoo-cli/internal/session"
)

var (
	entListULRe  = regexp.MustCompile(`(?is)<ul[^>]*id=["']ent_list_ul["'][^>]*>(.*?)</ul>`)
	liRe         = regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	hrefRe       = regexp.MustCompile(`(?is)href=["']([^"']+)["']`)
	entURLRe     = regexp.MustCompile(`(?i)/kanban/ent/(-?\d+)/([a-f0-9]+)`)
	entURLRelRe  = regexp.MustCompile(`(?i)(?:^|/)ent/(-?\d+)/([a-f0-9]+)`)
	boardListRe  = regexp.MustCompile(`(?i)board_list`)
	entNameBtnRe = regexp.MustCompile(`(?is)id=["']ent_list_button["'][^>]*>.*?<span[^>]*class=["'][^"']*ent_name[^"']*["'][^>]*>(.*?)</span>`)
	textStripRe  = regexp.MustCompile(`(?is)<[^>]+>`)
	lgEntIDRe    = regexp.MustCompile(`(?i)LG\.ent\.ent_id\s*=\s*(-?\d+)`)
	lgEntSignRe  = regexp.MustCompile(`(?i)LG\.ent\.ent_id_sign\s*=\s*["']([^"']+)["']`)
	lgEntNameRe  = regexp.MustCompile(`(?i)LG\.ent\.ent_name\s*=\s*["']([^"']*)["']`)
	boardDataRe  = regexp.MustCompile(`(?s)(?:var\s+)?board_data\s*=\s*(\{.*?\});[\r\n]`)
	gBoardRe     = regexp.MustCompile(`(?s)(?:var\s+)?g_board\s*=\s*(\{.*?\});[\r\n]`)
	boardIDRe    = regexp.MustCompile(`(?i)(?:var\s+)?board_id\s*=\s*["']?(\d+)["']?`)
	newAPIRe     = regexp.MustCompile(`(?i)(?:var\s+)?NEW_LEANGOO_WEB_URL\s*=\s*["']([^"']+)["']`)
	locHrefRe    = regexp.MustCompile(`(?i)(?:location\.href|window\.location)\s*=\s*['"]([^'"]+)['"]`)
	dataHrefRe   = regexp.MustCompile(`(?i)data-(?:href|url)=["']([^"']+)["']`)
)

func stripTags(s string) string {
	s = textStripRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func mergeEnt(byID map[int64]session.Ent, order *[]int64, e session.Ent) {
	if e.Name == "" && e.ID == 0 {
		return
	}
	if strings.Contains(e.Name, "新建企业") || strings.Contains(e.Name, "升级") {
		return
	}
	e.Name = strings.TrimSpace(strings.TrimSuffix(e.Name, ""))
	prev, ok := byID[e.ID]
	if !ok {
		*order = append(*order, e.ID)
		byID[e.ID] = e
		return
	}
	if e.Sign != "" {
		prev.Sign = e.Sign
	}
	if e.HomeURL != "" {
		prev.HomeURL = e.HomeURL
	}
	if e.Name != "" && (prev.Name == "" || strings.HasPrefix(prev.Name, "企业 ")) {
		prev.Name = e.Name
	}
	byID[e.ID] = prev
}

func ParseEnterprises(pageHTML string) []session.Ent {
	byID := map[int64]session.Ent{}
	order := []int64{}

	if m := entListULRe.FindStringSubmatch(pageHTML); len(m) >= 2 {
		for _, li := range liRe.FindAllStringSubmatch(m[1], -1) {
			inner := li[1]
			name := stripTags(inner)
			href := ""
			if hm := hrefRe.FindStringSubmatch(inner); len(hm) >= 2 {
				href = hm[1]
			}
			if !entURLRe.MatchString(href) && !entURLRelRe.MatchString(href) && !boardListRe.MatchString(href) {
				if om := locHrefRe.FindStringSubmatch(inner); len(om) >= 2 {
					href = om[1]
				}
				if dm := dataHrefRe.FindStringSubmatch(inner); len(dm) >= 2 {
					href = dm[1]
				}
			}
			if ent := entFromHref(href, name); ent != nil {
				mergeEnt(byID, &order, *ent)
				continue
			}
			if em := entURLRelRe.FindStringSubmatch(inner); len(em) >= 3 {
				id, _ := strconv.ParseInt(em[1], 10, 64)
				mergeEnt(byID, &order, session.Ent{
					ID:      id,
					Name:    name,
					Sign:    em[2],
					HomeURL: fmt.Sprintf("https://www.lg.team/kanban/ent/%d/%s", id, em[2]),
				})
			}
		}
	}

	for _, m := range entURLRe.FindAllStringSubmatch(pageHTML, -1) {
		id, _ := strconv.ParseInt(m[1], 10, 64)
		mergeEnt(byID, &order, session.Ent{
			ID:      id,
			Name:    fmt.Sprintf("企业 %d", id),
			Sign:    m[2],
			HomeURL: fmt.Sprintf("https://www.lg.team/kanban/ent/%d/%s", id, m[2]),
		})
	}

	if idm := lgEntIDRe.FindStringSubmatch(pageHTML); len(idm) >= 2 {
		id, _ := strconv.ParseInt(idm[1], 10, 64)
		name := "当前企业"
		if nm := lgEntNameRe.FindStringSubmatch(pageHTML); len(nm) >= 2 {
			name = html.UnescapeString(nm[1])
		}
		if btn := entNameBtnRe.FindStringSubmatch(pageHTML); len(btn) >= 2 {
			if n := stripTags(btn[1]); n != "" {
				name = n
			}
		}
		sign := ""
		if sm := lgEntSignRe.FindStringSubmatch(pageHTML); len(sm) >= 2 {
			sign = sm[1]
		}
		home := PersonalHomeURL()
		if id != -1 && sign != "" {
			home = fmt.Sprintf("https://www.lg.team/kanban/ent/%d/%s", id, sign)
		}
		mergeEnt(byID, &order, session.Ent{ID: id, Name: name, Sign: sign, HomeURL: home})
	}

	mergeEnt(byID, &order, session.Ent{ID: -1, Name: "团队版", HomeURL: PersonalHomeURL()})

	out := make([]session.Ent, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func entFromHref(href, name string) *session.Ent {
	if boardListRe.MatchString(href) || ((href == "" || strings.HasPrefix(href, "javascript")) && (name == "团队版" || strings.Contains(name, "团队版"))) {
		if name == "" {
			name = "团队版"
		}
		return &session.Ent{ID: -1, Name: name, HomeURL: PersonalHomeURL()}
	}
	if m := entURLRe.FindStringSubmatch(href); len(m) >= 3 {
		id, _ := strconv.ParseInt(m[1], 10, 64)
		return &session.Ent{
			ID:      id,
			Name:    name,
			Sign:    m[2],
			HomeURL: fmt.Sprintf("https://www.lg.team/kanban/ent/%d/%s", id, m[2]),
		}
	}
	if m := entURLRelRe.FindStringSubmatch(href); len(m) >= 3 {
		id, _ := strconv.ParseInt(m[1], 10, 64)
		return &session.Ent{
			ID:      id,
			Name:    name,
			Sign:    m[2],
			HomeURL: fmt.Sprintf("https://www.lg.team/kanban/ent/%d/%s", id, m[2]),
		}
	}
	if name == "团队版" || strings.HasPrefix(name, "团队版") {
		return &session.Ent{ID: -1, Name: "团队版", HomeURL: PersonalHomeURL()}
	}
	return nil
}

func PersonalHomeURL() string {
	return "https://www.lg.team/kanban/board_list"
}

func CurrentEntFromPage(pageHTML, fallbackURL string) *session.Ent {
	ents := ParseEnterprises(pageHTML)
	if idm := lgEntIDRe.FindStringSubmatch(pageHTML); len(idm) >= 2 {
		id, _ := strconv.ParseInt(idm[1], 10, 64)
		for i := range ents {
			if ents[i].ID == id {
				return &ents[i]
			}
		}
	}
	if strings.Contains(fallbackURL, "board_list") {
		return &session.Ent{ID: -1, Name: "团队版", HomeURL: PersonalHomeURL()}
	}
	if m := entURLRe.FindStringSubmatch(fallbackURL); len(m) >= 3 {
		id, _ := strconv.ParseInt(m[1], 10, 64)
		name := "企业"
		if btn := entNameBtnRe.FindStringSubmatch(pageHTML); len(btn) >= 2 {
			if n := stripTags(btn[1]); n != "" {
				name = n
			}
		}
		return &session.Ent{ID: id, Name: name, Sign: m[2], HomeURL: strings.Split(fallbackURL, "#")[0]}
	}
	if len(ents) > 0 {
		e := ents[0]
		return &e
	}
	return nil
}

func ExtractJSObject(page, name string) (json.RawMessage, error) {
	var re *regexp.Regexp
	switch name {
	case "board_data":
		re = boardDataRe
	case "g_board":
		re = gBoardRe
	default:
		return nil, fmt.Errorf("unknown pattern")
	}
	if m := re.FindStringSubmatch(page); len(m) >= 2 && json.Valid([]byte(m[1])) {
		return json.RawMessage(m[1]), nil
	}
	return extractBalancedObject(page, name)
}

func extractBalancedObject(page, name string) (json.RawMessage, error) {
	needle := name
	searchFrom := 0
	for {
		idx := strings.Index(page[searchFrom:], needle)
		if idx < 0 {
			break
		}
		idx += searchFrom
		rest := page[idx:]
		eq := strings.Index(rest, "=")
		if eq < 0 || eq > 48 {
			searchFrom = idx + len(needle)
			continue
		}
		start := -1
		for i := eq; i < len(rest); i++ {
			if rest[i] == '{' {
				start = i
				break
			}
			if rest[i] == ';' {
				break
			}
		}
		if start < 0 {
			searchFrom = idx + len(needle)
			continue
		}
		depth := 0
		inStr := false
		esc := false
		var quote byte
		for i := start; i < len(rest); i++ {
			ch := rest[i]
			if inStr {
				if esc {
					esc = false
					continue
				}
				if ch == '\\' {
					esc = true
					continue
				}
				if ch == quote {
					inStr = false
				}
				continue
			}
			if ch == '"' || ch == '\'' {
				inStr = true
				quote = ch
				continue
			}
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					raw := rest[start : i+1]
					if json.Valid([]byte(raw)) {
						return json.RawMessage(raw), nil
					}
					return nil, fmt.Errorf("找到 %s 但不是合法 JSON", name)
				}
			}
		}
		searchFrom = idx + len(needle)
	}
	return nil, fmt.Errorf("页面中未找到 %s", name)
}

func ExtractBoardID(page string) string {
	if m := boardIDRe.FindStringSubmatch(page); len(m) >= 2 {
		return m[1]
	}
	return ""
}

func ExtractNewLeangooWebURL(page string) string {
	if m := newAPIRe.FindStringSubmatch(page); len(m) >= 2 {
		return strings.TrimRight(m[1], "/")
	}
	return ""
}
