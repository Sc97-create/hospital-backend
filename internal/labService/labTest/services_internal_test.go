package labservice

import (
	"strings"
	"testing"

	"hospital-backend/internal/labService/labTest/dto"
)

func TestBuildLabOrderListAndCountQuery(t *testing.T) {
	svc := NewLabOrderService(nil)
	req := dto.ListLabOrdersRequest{
		OrganisationID: "org-1",
		PatientID:      "pat-1",
		Status:         OrderStatusPending,
		Limit:          10,
		PageNo:         2,
	}
	req.DBLimit, req.DBOffset = svc.parsePagination(req.Limit, req.PageNo)

	listQuery, listArgs := svc.buildListQuery(req)
	if !strings.Contains(listQuery, "lo.organisation_id = $1") {
		t.Fatal("expected organisation filter")
	}
	if !strings.Contains(listQuery, "patient_name") || !strings.Contains(listQuery, "doctor_name") {
		t.Fatal("expected patient/doctor name joins")
	}
	if !strings.Contains(listQuery, "lo.patient_id = $2") {
		t.Fatal("expected patient filter")
	}
	if !strings.Contains(listQuery, "lo.status = $3") {
		t.Fatal("expected status filter")
	}
	if !strings.Contains(listQuery, "LIMIT") || !strings.Contains(listQuery, "OFFSET") {
		t.Fatal("expected pagination")
	}
	if listArgs[0] != "org-1" || listArgs[1] != "pat-1" || listArgs[2] != OrderStatusPending {
		t.Fatalf("unexpected list args: %#v", listArgs)
	}
	if req.DBLimit != 10 || req.DBOffset != 10 {
		t.Fatalf("unexpected pagination: limit=%d offset=%d", req.DBLimit, req.DBOffset)
	}

	countQuery, countArgs := svc.buildCountQuery(req)
	if !strings.Contains(countQuery, "COUNT(*)") {
		t.Fatal("expected count query")
	}
	if !strings.Contains(countQuery, "lo.patient_id = $2") || !strings.Contains(countQuery, "lo.status = $3") {
		t.Fatal("expected filters on count query")
	}
	if countArgs[0] != "org-1" {
		t.Fatalf("unexpected count args: %#v", countArgs)
	}
}

func TestBuildLabOrderGetQueryIncludesPatientDetails(t *testing.T) {
	svc := NewLabOrderService(nil)
	q, args := svc.buildGetQuery("lo-1")
	if !strings.Contains(q, "patient_code") || !strings.Contains(q, "patient_age") {
		t.Fatalf("expected patient code/age in get query: %s", q)
	}
	if !strings.Contains(q, "p.uh_id") || !strings.Contains(q, "p.age") {
		t.Fatalf("expected patient join columns in get query: %s", q)
	}
	if !strings.Contains(q, "doctor_name") {
		t.Fatal("expected doctor_name in get query")
	}
	if len(args) != 1 || args[0] != "lo-1" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestParsePaginationDefaults(t *testing.T) {
	svc := NewLabOrderService(nil)
	limit, offset := svc.parsePagination(0, 0)
	if limit != 10 || offset != 0 {
		t.Fatalf("defaults: got limit=%d offset=%d", limit, offset)
	}
}

func TestIsValidOrderAndTestStatus(t *testing.T) {
	if !isValidOrderStatus(OrderStatusPending) || !isValidOrderStatus(OrderStatusPaid) || !isValidOrderStatus(OrderStatusCompleted) {
		t.Fatal("expected order statuses valid")
	}
	if isValidOrderStatus("active") {
		t.Fatal("active should be invalid for orders")
	}
	if !isValidTestStatus(TestStatusPending) || !isValidTestStatus(TestStatusCancelled) {
		t.Fatal("expected test statuses valid")
	}
	if isValidTestStatus("inactive") {
		t.Fatal("inactive should be invalid for tests")
	}
}

func TestBuildLabOrderQueryVariants(t *testing.T) {
	svc := NewLabOrderService(nil)

	t.Run("org only", func(t *testing.T) {
		req := dto.ListLabOrdersRequest{OrganisationID: "org-1", DBLimit: 5, DBOffset: 0}
		q, args := svc.buildListQuery(req)
		if strings.Contains(q, "lo.status =") || strings.Contains(q, "lo.patient_id =") {
			t.Fatalf("unexpected filters in %s", q)
		}
		if len(args) != 3 || args[0] != "org-1" || args[1] != 5 || args[2] != 0 {
			t.Fatalf("unexpected args: %#v", args)
		}
	})

	t.Run("status without patient", func(t *testing.T) {
		req := dto.ListLabOrdersRequest{
			OrganisationID: "org-1",
			Status:         OrderStatusCancelled,
			DBLimit:        20,
			DBOffset:       40,
		}
		q, args := svc.buildListQuery(req)
		if !strings.Contains(q, "lo.status = $2") {
			t.Fatalf("expected status placeholder, got %s", q)
		}
		if strings.Contains(q, "lo.patient_id =") {
			t.Fatal("did not expect patient filter")
		}
		if args[1] != OrderStatusCancelled || args[2] != 20 || args[3] != 40 {
			t.Fatalf("unexpected args: %#v", args)
		}
	})

	t.Run("patient without status", func(t *testing.T) {
		req := dto.ListLabOrdersRequest{
			OrganisationID: "org-1",
			PatientID:      "pat-9",
			DBLimit:        10,
			DBOffset:       0,
		}
		q, args := svc.buildListQuery(req)
		if !strings.Contains(q, "lo.patient_id = $2") {
			t.Fatalf("expected patient at $2, got %s", q)
		}
		if args[1] != "pat-9" {
			t.Fatalf("unexpected patient arg: %#v", args[1])
		}
	})
}

func TestAllTestsCompleted(t *testing.T) {
	if allTestsCompleted(nil) || allTestsCompleted(OrderedLabTestList{}) {
		t.Fatal("empty should not be completed")
	}
	if allTestsCompleted(OrderedLabTestList{
		{Status: TestStatusCompleted},
		{Status: TestStatusPending},
	}) {
		t.Fatal("mixed should not be completed")
	}
	if !allTestsCompleted(OrderedLabTestList{
		{Status: TestStatusCompleted},
		{Status: TestStatusCompleted},
	}) {
		t.Fatal("all completed should return true")
	}
}
