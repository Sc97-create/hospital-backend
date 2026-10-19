package diagnosticcentre

import (
	"strings"
	"testing"

	"hospital-backend/internal/labService/diagnosticCentre/dto"
)

func TestBuildDiagnosticCentreListAndCountQuery(t *testing.T) {
	svc := NewDiagnosticCentreService(nil)
	req := dto.ListDiagnosticCentresRequest{
		OrganisationID: "org-1",
		Search:         "city",
		Status:         StatusActive,
		Limit:          10,
		PageNo:         2,
	}
	req.DBLimit, req.DBOffset = svc.parsePagination(req.Limit, req.PageNo)

	listQuery, listArgs := svc.buildListQuery(req)
	if !strings.Contains(listQuery, "organisation_id = $1") {
		t.Fatal("expected organisation filter")
	}
	if !strings.Contains(listQuery, "status = $2") {
		t.Fatal("expected status filter")
	}
	if !strings.Contains(listQuery, "ILIKE") {
		t.Fatal("expected search filter")
	}
	if !strings.Contains(listQuery, "LIMIT") || !strings.Contains(listQuery, "OFFSET") {
		t.Fatal("expected pagination")
	}
	if listArgs[0] != "org-1" || listArgs[1] != StatusActive || listArgs[2] != "%city%" {
		t.Fatalf("unexpected list args: %#v", listArgs)
	}
	if req.DBLimit != 10 || req.DBOffset != 10 {
		t.Fatalf("unexpected pagination: limit=%d offset=%d", req.DBLimit, req.DBOffset)
	}

	countQuery, countArgs := svc.buildCountQuery(req)
	if !strings.Contains(countQuery, "COUNT(*)") {
		t.Fatal("expected count query")
	}
	if countArgs[0] != "org-1" {
		t.Fatalf("unexpected count args: %#v", countArgs)
	}
}

func TestParsePaginationDefaults(t *testing.T) {
	svc := NewDiagnosticCentreService(nil)
	limit, offset := svc.parsePagination(0, 0)
	if limit != 10 || offset != 0 {
		t.Fatalf("defaults: got limit=%d offset=%d", limit, offset)
	}
}

func TestIsValidStatus(t *testing.T) {
	if !isValidStatus(StatusActive) || !isValidStatus(StatusInactive) {
		t.Fatal("expected active/inactive valid")
	}
	if isValidStatus("pending") {
		t.Fatal("pending should be invalid")
	}
}
