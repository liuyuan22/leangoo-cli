package parse_test

import (
	"testing"

	"github.com/deepglint/leangoo-cli/internal/parse"
)

func TestParseEnterprises(t *testing.T) {
	html := `
<ul id="ent_list_ul">
  <li><a href="/kanban/board_list">团队版</a></li>
  <li class="active"><a href="/kanban/ent/15599/9e626234604d4360bd48e5318dd9e385">北京格灵深瞳信息技术股份有限公司</a></li>
  <li class="add-ent-li js-create-new-ent-nav-li"><a href="#">新建企业...</a></li>
</ul>
<script>
LG.ent.ent_id = 15599;
LG.ent.ent_id_sign = "9e626234604d4360bd48e5318dd9e385";
LG.ent.ent_name = "北京格灵深瞳信息技术股份有限公司";
</script>
`
	ents := parse.ParseEnterprises(html)
	if len(ents) < 2 {
		t.Fatalf("expected >=2 ents, got %#v", ents)
	}
	foundPersonal, foundEnt := false, false
	for _, e := range ents {
		if e.ID == -1 {
			foundPersonal = true
		}
		if e.ID == 15599 && e.Sign != "" {
			foundEnt = true
		}
	}
	if !foundPersonal || !foundEnt {
		t.Fatalf("missing ents: %#v", ents)
	}
}

func TestExtractBoardID(t *testing.T) {
	html := `var board_id = 12345;`
	if got := parse.ExtractBoardID(html); got != "12345" {
		t.Fatalf("got %q", got)
	}
}
