package update

import "testing"

func TestIsOutdated(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"1.1.0", "1.1.0", false},
		{"1.1.0", "1.1.1", true},
		{"dev", "1.1.1", true},
		{"1.1.1-dev", "1.1.1", true},
		{"1.1.1", "1.1.1", false},
	}
	for _, c := range cases {
		if got := isOutdated(c.cur, c.latest); got != c.want {
			t.Fatalf("isOutdated(%q,%q)=%v want %v", c.cur, c.latest, got, c.want)
		}
	}
}
