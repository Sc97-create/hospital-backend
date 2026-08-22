package medicine

import (
	"strings"
	"testing"
	"time"

	meddto "hospital-backend/internal/medicine/dto"
)

func TestParsePagination(t *testing.T) {
	svc := NewSupplierService(nil)

	limit, offset := svc.parsePagination(10, 1)
	if limit != 10 || offset != 0 {
		t.Fatalf("page 1: got limit=%d offset=%d", limit, offset)
	}

	limit, offset = svc.parsePagination(10, 3)
	if limit != 10 || offset != 20 {
		t.Fatalf("page 3: got limit=%d offset=%d", limit, offset)
	}

	limit, offset = svc.parsePagination(0, 0)
	if limit != 10 || offset != 0 {
		t.Fatalf("defaults: got limit=%d offset=%d", limit, offset)
	}
}

func TestFindPaymentTerms(t *testing.T) {
	svc := NewSupplierService(nil)

	tests := []struct {
		input string
		want  Paymentterms
	}{
		{input: "Cash", want: ICash},
		{input: "Net 30", want: Net30},
		{input: "Net 15", want: Net15},
		{input: "Net 45", want: Net45},
		{input: "unknown", want: Advance},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := svc.findPaymentTerms(tt.input)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestAppendSupplierFiltersAndBuildSupplierListQuery(t *testing.T) {
	svc := NewSupplierService(nil)

	req := meddto.SupplierListReq{
		OrganisationID: "org-1",
		Search:         "acme",
		Limit:          10,
		PageNo:         1,
	}
	req.DBLimit, req.DBOffset = svc.parsePagination(req.Limit, req.PageNo)

	listQuery, listArgs := svc.buildSupplierListQuery(req)
	if !strings.Contains(listQuery, "organisation_id = $1") {
		t.Fatal("expected organisation filter in list query")
	}
	if !strings.Contains(listQuery, "ILIKE") {
		t.Fatal("expected search filter in list query")
	}
	if !strings.Contains(listQuery, "LIMIT") || !strings.Contains(listQuery, "OFFSET") {
		t.Fatal("expected pagination in list query")
	}
	if len(listArgs) < 4 {
		t.Fatalf("expected at least 4 args, got %d", len(listArgs))
	}
	if listArgs[0] != "org-1" {
		t.Fatalf("expected org-1 arg, got %v", listArgs[0])
	}

	countQuery, countArgs := svc.buildSupplierCountQuery(req)
	if !strings.Contains(countQuery, "COUNT(*)") {
		t.Fatal("expected count query")
	}
	if !strings.Contains(countQuery, "ILIKE") {
		t.Fatal("expected search filter in count query")
	}
	if countArgs[0] != "org-1" {
		t.Fatalf("expected org-1 in count args, got %v", countArgs[0])
	}

	noSearchReq := meddto.SupplierListReq{OrganisationID: "org-1", Limit: 5, PageNo: 1}
	noSearchReq.DBLimit, noSearchReq.DBOffset = svc.parsePagination(noSearchReq.Limit, noSearchReq.PageNo)
	baseQuery := `SELECT id FROM suppliers WHERE organisation_id = $1`
	filtered, args, _ := svc.appendSupplierFilters(baseQuery, noSearchReq, []interface{}{noSearchReq.OrganisationID}, 2)
	if filtered != baseQuery {
		t.Fatal("expected no extra filters without search")
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg without search, got %d", len(args))
	}
}

func TestCreateCode(t *testing.T) {
	svc := NewSupplierService(nil)
	code := svc.createCode(SUPP)
	if !strings.HasPrefix(code, "SUPP-") {
		t.Fatalf("expected SUPP- prefix, got %q", code)
	}
	parts := strings.Split(code, "-")
	if len(parts) != 2 || len(parts[1]) != 4 {
		t.Fatalf("expected SUPP-XXXX format, got %q", code)
	}
}

func TestToSupplierList(t *testing.T) {
	svc := NewSupplierService(nil)
	createdAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	empty := svc.toSupplierList(nil)
	if len(empty) != 0 {
		t.Fatalf("expected empty list, got %d items", len(empty))
	}

	suppliers := []Supplier{
		{ID: "sup-1", SupplierCode: "SUPP-1001", Name: "A", CreatedAt: createdAt},
		{ID: "sup-2", SupplierCode: "SUPP-1002", Name: "B", PaymentTerms: Net30, SupplierStatus: Active, CreatedAt: createdAt},
	}
	list := svc.toSupplierList(suppliers)
	if len(list) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list))
	}
	if list[0].CreatedAt != "15 Jan 2026" {
		t.Fatalf("expected formatted date, got %q", list[0].CreatedAt)
	}
	if list[1].PaymentTerms != "Net 30" || list[1].SupplierStatus != "Active" {
		t.Fatalf("unexpected mapping: %+v", list[1])
	}
}
