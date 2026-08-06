package api

import "testing"

func TestParseFEBEEstimates(t *testing.T) {
	fe, be := parseFEBEEstimates("前：展示持续告警信息（0.5）\n后：themis后端接口（0.5）")
	if fe != "0.5" || be != "0.5" {
		t.Fatalf("got fe=%q be=%q", fe, be)
	}
	fe, be = parseFEBEEstimates("前端 3\n后端 5")
	if fe != "3" || be != "5" {
		t.Fatalf("got fe=%q be=%q", fe, be)
	}
	fe, be = parseFEBEEstimates("无点数")
	if fe != "" || be != "" {
		t.Fatalf("expected empty, got fe=%q be=%q", fe, be)
	}
}
