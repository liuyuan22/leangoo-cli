package parse_test

import (
	"testing"

	"github.com/deepglint/leangoo-cli/internal/parse"
)

func TestParseBoardRef(t *testing.T) {
	cases := []struct {
		in    string
		board string
		task  string
	}{
		{"1f17691e-2963-6670-8314-0242ac140003", "1f17691e-2963-6670-8314-0242ac140003", ""},
		{"https://www.lg.team/kanban/board/go/1f17691e-2963-6670-8314-0242ac140003#/board_view", "1f17691e-2963-6670-8314-0242ac140003", ""},
		{"https://www.lg.team/kanban/board/go/1f17691e-2963-6670-8314-0242ac140003/1f1803da-fd59-6e70-a664-0242c0a8e003", "1f17691e-2963-6670-8314-0242ac140003", "1f1803da-fd59-6e70-a664-0242c0a8e003"},
		{"/kanban/board/go/1f17691e-2963-6670-8314-0242ac140003/", "1f17691e-2963-6670-8314-0242ac140003", ""},
	}
	for _, c := range cases {
		ref, err := parse.ParseBoardRef(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if ref.BoardUUID != c.board || ref.TaskUUID != c.task {
			t.Fatalf("%q => board=%q task=%q, want board=%q task=%q", c.in, ref.BoardUUID, ref.TaskUUID, c.board, c.task)
		}
	}
}
