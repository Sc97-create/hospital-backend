package labservice_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	labservice "hospital-backend/internal/labService/labTest"
	"hospital-backend/internal/labService/labTest/dto"
	"hospital-backend/internal/testutil/controllertest"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// memoryLabOrderRepo is a stateful fake used for complex multi-API flows.
type memoryLabOrderRepo struct {
	mu   sync.Mutex
	rows map[string]labservice.LabOrders
}

func newMemoryLabOrderRepo() *memoryLabOrderRepo {
	return &memoryLabOrderRepo{rows: map[string]labservice.LabOrders{}}
}

func (m *memoryLabOrderRepo) Create(_ *zap.Logger, order *labservice.LabOrders) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if order.ID == "" {
		order.ID = uuid.New().String()
	}
	m.rows[order.ID] = *order
	return nil
}

func (m *memoryLabOrderRepo) FindByID(_ *zap.Logger, id string) (labservice.LabOrders, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok {
		return labservice.LabOrders{}, gorm.ErrRecordNotFound
	}
	return row, nil
}

func toListRow(row labservice.LabOrders) labservice.LabOrderListRow {
	return labservice.LabOrderListRow{
		ID:                 row.ID,
		OrganisationID:     row.OrganisationID,
		PatientID:          row.PatientID,
		PatientName:        "Patient " + row.PatientID,
		PatientCode:        "CODE-" + row.PatientID,
		PatientAge:         30,
		DoctorID:           row.DoctorID,
		DoctorName:         "Doctor " + row.DoctorID,
		DiagnosticCentreID: row.DiagnosticCentreID,
		AppointmentID:      row.AppointmentID,
		LabTests:           row.LabTests,
		Status:             row.Status,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func (m *memoryLabOrderRepo) FindMany(_ *zap.Logger, query string, args ...any) ([]labservice.LabOrderListRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// GetByID query filters by order id.
	if strings.Contains(query, "WHERE lo.id = $1") {
		id, _ := args[0].(string)
		row, ok := m.rows[id]
		if !ok {
			return []labservice.LabOrderListRow{}, nil
		}
		return []labservice.LabOrderListRow{toListRow(row)}, nil
	}

	orgID, _ := args[0].(string)
	patientFilter := ""
	statusFilter := ""
	limit := 10
	offset := 0

	argIdx := 1
	if strings.Contains(query, "patient_id =") {
		patientFilter, _ = args[argIdx].(string)
		argIdx++
	}
	if strings.Contains(query, "status =") {
		statusFilter, _ = args[argIdx].(string)
		argIdx++
	}
	if argIdx < len(args) {
		if v, ok := args[argIdx].(int); ok {
			limit = v
		}
	}
	if argIdx+1 < len(args) {
		if v, ok := args[argIdx+1].(int); ok {
			offset = v
		}
	}

	out := make([]labservice.LabOrderListRow, 0)
	for _, row := range m.rows {
		if row.OrganisationID != orgID {
			continue
		}
		if patientFilter != "" && row.PatientID != patientFilter {
			continue
		}
		if statusFilter != "" && row.Status != statusFilter {
			continue
		}
		out = append(out, toListRow(row))
	}
	if offset > len(out) {
		return []labservice.LabOrderListRow{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (m *memoryLabOrderRepo) Count(_ *zap.Logger, query string, args ...any) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	orgID, _ := args[0].(string)
	patientFilter := ""
	statusFilter := ""
	argIdx := 1
	if strings.Contains(query, "patient_id =") {
		patientFilter, _ = args[argIdx].(string)
		argIdx++
	}
	if strings.Contains(query, "status =") {
		statusFilter, _ = args[argIdx].(string)
	}
	var total int64
	for _, row := range m.rows {
		if row.OrganisationID != orgID {
			continue
		}
		if patientFilter != "" && row.PatientID != patientFilter {
			continue
		}
		if statusFilter != "" && row.Status != statusFilter {
			continue
		}
		total++
	}
	return total, nil
}

func (m *memoryLabOrderRepo) Save(_ *zap.Logger, order *labservice.LabOrders) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rows[order.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	m.rows[order.ID] = *order
	return nil
}

func (m *memoryLabOrderRepo) Exec(_ *zap.Logger, query string, args ...any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := strings.ToLower(query)
	if strings.Contains(q, "set status") {
		status, _ := args[0].(string)
		updatedAt, _ := args[1].(time.Time)
		id, _ := args[2].(string)
		row, ok := m.rows[id]
		if !ok {
			return gorm.ErrRecordNotFound
		}
		row.Status = status
		row.UpdatedAt = updatedAt
		m.rows[id] = row
		return nil
	}
	return fmt.Errorf("unsupported exec query")
}

func TestLabOrderComplexLifecycleAllAPIs(t *testing.T) {
	repo := newMemoryLabOrderRepo()
	svc := labservice.NewLabOrderService(repo)
	log := servicetest.NopLogger()

	orderID, err := svc.Create(log, dto.CreateLabOrderRequest{
		OrganisationID:     "org-1",
		PatientID:          "pat-1",
		DoctorID:           "doc-1",
		DiagnosticCentreID: "dc-1",
		Tests: []dto.CreateOrderedTestInput{
			{Name: "  CBC  ", SubTest: " Hb "},
			{Name: "Lipid Profile", SubTest: ""},
		},
	})
	if err != nil || orderID == "" {
		t.Fatalf("create failed: id=%q err=%v", orderID, err)
	}

	withAppt, err := svc.Create(log, dto.CreateLabOrderRequest{
		OrganisationID:     "org-1",
		PatientID:          "pat-2",
		DoctorID:           "doc-1",
		DiagnosticCentreID: "dc-1",
		AppointmentID:      "appt-9",
		Tests:              []dto.CreateOrderedTestInput{{Name: "Thyroid"}},
	})
	if err != nil {
		t.Fatalf("create with appointment failed: %v", err)
	}

	otherOrg, err := svc.Create(log, dto.CreateLabOrderRequest{
		OrganisationID:     "org-2",
		PatientID:          "pat-3",
		DoctorID:           "doc-2",
		DiagnosticCentreID: "dc-2",
		Tests:              []dto.CreateOrderedTestInput{{Name: "ESR"}},
	})
	if err != nil || otherOrg == "" {
		t.Fatalf("other org create failed: %v", err)
	}

	got, err := svc.GetByID(log, orderID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.AppointmentID != nil {
		t.Fatalf("expected nil appointment, got %v", got.AppointmentID)
	}
	if len(got.Tests) != 2 || got.Tests[0].Name != "CBC" || got.Tests[0].Status != labservice.TestStatusPending {
		t.Fatalf("unexpected order tests: %+v", got.Tests)
	}

	apptOrder, err := svc.GetByID(log, withAppt)
	if err != nil || apptOrder.AppointmentID == nil || *apptOrder.AppointmentID != "appt-9" {
		t.Fatalf("expected appointment on second order: %+v err=%v", apptOrder, err)
	}

	rows, total, err := svc.List(log, dto.ListLabOrdersRequest{
		OrganisationID: "org-1",
		PatientID:      "pat-1",
		Limit:          10,
		PageNo:         1,
	})
	if err != nil || total != 1 || len(rows) != 1 || rows[0].ID != orderID {
		t.Fatalf("list by patient failed: rows=%+v total=%d err=%v", rows, total, err)
	}

	testID := got.Tests[0].ID
	if err := svc.UpdateTestStatus(log, orderID, testID, labservice.TestStatusCompleted); err != nil {
		t.Fatalf("update first test: %v", err)
	}
	mid, err := svc.GetByID(log, orderID)
	if err != nil || mid.Status != labservice.OrderStatusPending {
		t.Fatalf("order should still be pending: %+v err=%v", mid, err)
	}

	if err := svc.UpdateTestStatus(log, orderID, got.Tests[1].ID, labservice.TestStatusCompleted); err != nil {
		t.Fatalf("update second test: %v", err)
	}
	done, err := svc.GetByID(log, orderID)
	if err != nil || done.Status != labservice.OrderStatusCompleted {
		t.Fatalf("order should be completed: %+v err=%v", done, err)
	}

	if err := svc.UpdateStatus(log, withAppt, labservice.OrderStatusCancelled); err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	cancelled, totalCancel, err := svc.List(log, dto.ListLabOrdersRequest{
		OrganisationID: "org-1",
		Status:         labservice.OrderStatusCancelled,
		Limit:          10,
		PageNo:         1,
	})
	if err != nil || totalCancel != 1 || cancelled[0].ID != withAppt {
		t.Fatalf("cancelled list failed: %+v total=%d err=%v", cancelled, totalCancel, err)
	}

	if err := svc.UpdateTestStatus(log, "missing", "t1", labservice.TestStatusCompleted); !errors.Is(err, wrapError.ErrLabOrderNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if _, err := svc.GetByID(log, "missing"); !errors.Is(err, wrapError.ErrLabOrderNotFound) {
		t.Fatalf("expected get not found, got %v", err)
	}
}

func TestLabOrderComplexHTTPAllRoutes(t *testing.T) {
	repo := newMemoryLabOrderRepo()
	svc := labservice.NewLabOrderService(repo)
	ctrl := labservice.NewLabOrderController(svc)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		g := app.Group("/lab-orders")
		g.Post("/create", ctrl.Create)
		g.Get("/list", ctrl.List)
		g.Get("/getbyID/:id", ctrl.GetByID)
		g.Patch("/update/:id/status", ctrl.UpdateStatus)
		g.Patch("/update/:id/tests/:testId/status", ctrl.UpdateTestStatus)
	})

	createResp, body := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/lab-orders/create",
		Body: `{
			"organisation_id":"org-1",
			"patient_id":"pat-1",
			"doctor_id":"doc-1",
			"diagnostic_centre_id":"dc-1",
			"tests":[{"name":"CBC","sub_test":"Hb"},{"name":"Lipid"}]
		}`,
	})
	controllertest.AssertStatus(t, createResp, fiber.StatusOK)
	var created struct {
		LabOrderID string `json:"lab_order_id"`
	}
	if err := json.Unmarshal(body, &created); err != nil || created.LabOrderID == "" {
		t.Fatalf("parse create: %v body=%s", err, body)
	}

	listResp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/lab-orders/list?organisation_id=org-1&status=pending&limit=5&page_no=1",
	})
	controllertest.AssertStatus(t, listResp, fiber.StatusOK)

	getResp, getBody := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/lab-orders/getbyID/" + created.LabOrderID,
	})
	controllertest.AssertStatus(t, getResp, fiber.StatusOK)
	var getOut struct {
		Data dto.LabOrderResponse `json:"data"`
	}
	if err := json.Unmarshal(getBody, &getOut); err != nil || len(getOut.Data.Tests) != 2 {
		t.Fatalf("get parse failed: %v body=%s", err, getBody)
	}

	testID := getOut.Data.Tests[0].ID
	testStatusResp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPatch,
		Path:   "/lab-orders/update/" + created.LabOrderID + "/tests/" + testID + "/status",
		Body:   `{"status":"completed"}`,
	})
	controllertest.AssertStatus(t, testStatusResp, fiber.StatusOK)

	orderStatusResp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPatch,
		Path:   "/lab-orders/update/" + created.LabOrderID + "/status",
		Body:   `{"status":"cancelled"}`,
	})
	controllertest.AssertStatus(t, orderStatusResp, fiber.StatusOK)

	missingResp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/lab-orders/getbyID/" + uuid.New().String(),
	})
	controllertest.AssertStatus(t, missingResp, fiber.StatusNotFound)

	badCreate, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/lab-orders/create",
		Body:   `{"organisation_id":"org-1"}`,
	})
	controllertest.AssertStatus(t, badCreate, fiber.StatusBadRequest)

	badList, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/lab-orders/list?patient_id=x",
	})
	controllertest.AssertStatus(t, badList, fiber.StatusBadRequest)
}
