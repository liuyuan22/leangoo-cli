package parse

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	// /kanban/board/go/{board_uuid}[/{task_uuid}]
	boardGoPathRe = regexp.MustCompile(`(?i)/kanban/board/go/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(?:/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}))?`)
	uuidOnlyRe    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// BoardRef is a board (sprint) and optional task extracted from a UUID or lg.team URL.
type BoardRef struct {
	BoardUUID string
	TaskUUID  string
	Raw       string
}

// ParseBoardRef accepts:
//   - board UUID
//   - https://www.lg.team/kanban/board/go/{board_uuid}#/board_view
//   - https://www.lg.team/kanban/board/go/{board_uuid}/{task_uuid}
func ParseBoardRef(input string) (*BoardRef, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("链接或 UUID 为空")
	}

	ref := &BoardRef{Raw: input}

	if uuidOnlyRe.MatchString(input) {
		ref.BoardUUID = strings.ToLower(input)
		return ref, nil
	}

	// Allow path-only input
	s := input
	if !strings.Contains(s, "://") && strings.HasPrefix(s, "/kanban/") {
		s = "https://www.lg.team" + s
	}

	if u, err := url.Parse(s); err == nil && u.Path != "" {
		if m := boardGoPathRe.FindStringSubmatch(u.Path); len(m) >= 2 {
			ref.BoardUUID = strings.ToLower(m[1])
			if len(m) >= 3 && m[2] != "" {
				ref.TaskUUID = strings.ToLower(m[2])
			}
			return ref, nil
		}
		// Also search full string (in case of unusual hosts)
	}

	if m := boardGoPathRe.FindStringSubmatch(input); len(m) >= 2 {
		ref.BoardUUID = strings.ToLower(m[1])
		if len(m) >= 3 && m[2] != "" {
			ref.TaskUUID = strings.ToLower(m[2])
		}
		return ref, nil
	}

	return nil, fmt.Errorf("无法解析看板链接或 UUID: %s", input)
}

// IsBoardURL reports whether s looks like an lg.team board URL.
func IsBoardURL(s string) bool {
	s = strings.TrimSpace(s)
	return strings.Contains(s, "/kanban/board/go/") || strings.Contains(s, "lg.team")
}
