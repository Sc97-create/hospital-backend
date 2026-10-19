package labservice

import "testing"

func TestDefaultLabTestsCatalog(t *testing.T) {
	if len(DefaultLabTests) != 21 {
		t.Fatalf("expected 21 default tests, got %d", len(DefaultLabTests))
	}
	cbc, ok := DefaultLabTests["HEM-01"]
	if !ok || cbc.TestName == "" || cbc.TATMinutes != 120 {
		t.Fatalf("unexpected CBC template: %+v", cbc)
	}
	list := DefaultLabTestList()
	if len(list) != len(DefaultLabTests) {
		t.Fatalf("list length mismatch: %d vs %d", len(list), len(DefaultLabTests))
	}
	if list[0].TestCode != "BIO-04" {
		t.Fatalf("expected first code BIO-04, got %s", list[0].TestCode)
	}
}

func TestParseTATMinutes(t *testing.T) {
	if got := parseTATMinutes("2 hrs TAT"); got != 120 {
		t.Fatalf("got %d", got)
	}
	if got := parseTATMinutes("48 hrs TAT"); got != 2880 {
		t.Fatalf("got %d", got)
	}
	if got := parseTATMinutes(""); got != 0 {
		t.Fatalf("got %d", got)
	}
}
