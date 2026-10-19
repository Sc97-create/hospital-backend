package diagnosticcentre_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	diagnosticcentre "hospital-backend/internal/labService/diagnosticCentre"
	"hospital-backend/internal/labService/diagnosticCentre/dto"
	"hospital-backend/internal/testutil/controllertest"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type memoryDiagnosticCentreRepo struct {
	mu   sync.Mutex
	rows map[string]diagnosticcentre.DiagnosticCentre
}

func newMemoryRepo() *memoryDiagnosticCentreRepo {
	return &memoryDiagnosticCentreRepo{rows: map[string]diagnosticcentre.DiagnosticCentre{}}
}

func (m *memoryDiagnosticCentreRepo) Create(_ *zap.Logger, centre *diagnosticcentre.DiagnosticCentre) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.rows {
		if existing.OrganisationID == centre.OrganisationID && strings.EqualFold(existing.Name, centre.Name) {
			return errors.New("duplicate key")
		}
	}
	if centre.ID == "" {
		centre.ID = uuid.New().String()
	}
	m.rows[centre.ID] = *centre
	return nil
}

func (m *memoryDiagnosticCentreRepo) FindMany(_ *zap.Logger, query string, args ...any) ([]diagnosticcentre.DiagnosticCentre, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	orgID, _ := args[0].(string)
	statusFilter := ""
	search := ""
	limit := 10
	offset := 0
	argIdx := 1
	if strings.Contains(query, "status =") {
		statusFilter, _ = args[argIdx].(string)
		argIdx++
	}
	if strings.Contains(query, "ILIKE") {
		search, _ = args[argIdx].(string)
		search = strings.Trim(search, "%")
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

	out := make([]diagnosticcentre.DiagnosticCentre, 0)
	for _, row := range m.rows {
		if row.OrganisationID != orgID {
			continue
		}
		if statusFilter != "" && row.Status != statusFilter {
			continue
		}
		if search != "" {
			needle := strings.ToLower(search)
			haystack := strings.ToLower(row.Name + " " + row.Address + " " + row.ContactNumber + " " + row.Email)
			if !strings.Contains(haystack, needle) {
				continue
			}
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if offset > len(out) {
		return []diagnosticcentre.DiagnosticCentre{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (m *memoryDiagnosticCentreRepo) Count(_ *zap.Logger, query string, args ...any) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	orgID, _ := args[0].(string)
	statusFilter := ""
	search := ""
	argIdx := 1
	if strings.Contains(query, "status =") {
		statusFilter, _ = args[argIdx].(string)
		argIdx++
	}
	if strings.Contains(query, "ILIKE") {
		search, _ = args[argIdx].(string)
		search = strings.Trim(search, "%")
	}
	var total int64
	for _, row := range m.rows {
		if row.OrganisationID != orgID {
			continue
		}
		if statusFilter != "" && row.Status != statusFilter {
			continue
		}
		if search != "" {
			needle := strings.ToLower(search)
			haystack := strings.ToLower(row.Name + " " + row.Address + " " + row.ContactNumber + " " + row.Email)
			if !strings.Contains(haystack, needle) {
				continue
			}
		}
		total++
	}
	return total, nil
}

func (m *memoryDiagnosticCentreRepo) Exec(_ *zap.Logger, query string, args ...any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := strings.ToLower(query)
	if strings.Contains(q, "set name") {
		name, _ := args[0].(string)
		address, _ := args[1].(string)
		contact, _ := args[2].(string)
		email, _ := args[3].(string)
		updatedAt, _ := args[4].(time.Time)
		id, _ := args[5].(string)
		row, ok := m.rows[id]
		if !ok {
			return gorm.ErrRecordNotFound
		}
		row.Name = name
		row.Address = address
		row.ContactNumber = contact
		row.Email = email
		row.UpdatedAt = updatedAt
		m.rows[id] = row
		return nil
	}
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

func TestDiagnosticCentreComplexLifecycleAllAPIs(t *testing.T) {
	repo := newMemoryRepo()
	svc := diagnosticcentre.NewDiagnosticCentreService(repo)
	log := servicetest.NopLogger()

	cityID, err := svc.Create(log, dto.CreateDiagnosticCentreRequest{
		OrganisationID: "org-1",
		Name:           "  City Diagnostics  ",
		Address:        " 12 Main St ",
		ContactNumber:  " 9000011111 ",
		Email:          " city@lab.com ",
	})
	if err != nil || cityID == "" {
		t.Fatalf("create city: id=%q err=%v", cityID, err)
	}
	metroID, err := svc.Create(log, dto.CreateDiagnosticCentreRequest{
		OrganisationID: "org-1",
		Name:           "Metro Path Lab",
		Address:        "Ring Road",
		ContactNumber:  "9000022222",
		Email:          "metro@lab.com",
	})
	if err != nil {
		t.Fatalf("create metro: %v", err)
	}
	otherID, err := svc.Create(log, dto.CreateDiagnosticCentreRequest{
		OrganisationID: "org-2",
		Name:           "City Diagnostics",
		Address:        "Other City",
	})
	if err != nil {
		t.Fatalf("create other org: %v", err)
	}
	if otherID == cityID {
		t.Fatal("expected distinct ids across orgs")
	}

	_, err = svc.Create(log, dto.CreateDiagnosticCentreRequest{
		OrganisationID: "org-1",
		Name:           "City Diagnostics",
	})
	if !errors.Is(err, wrapError.ErrDiagnosticCentreCreateFailed) {
		t.Fatalf("expected duplicate create failed, got %v", err)
	}

	rows, total, err := svc.List(log, dto.ListDiagnosticCentresRequest{
		OrganisationID: "org-1",
		Search:         "city",
		Status:         diagnosticcentre.StatusActive,
		Limit:          10,
		PageNo:         1,
	})
	if err != nil || total != 1 || len(rows) != 1 || rows[0].ID != cityID {
		t.Fatalf("search list: rows=%+v total=%d err=%v", rows, total, err)
	}
	if rows[0].Name != "City Diagnostics" || rows[0].Address != "12 Main St" {
		t.Fatalf("expected trimmed fields, got %+v", rows[0])
	}

	page1, totalAll, err := svc.List(log, dto.ListDiagnosticCentresRequest{
		OrganisationID: "org-1",
		Limit:          1,
		PageNo:         1,
	})
	if err != nil || totalAll != 2 || len(page1) != 1 {
		t.Fatalf("page1: len=%d total=%d err=%v", len(page1), totalAll, err)
	}
	page2, _, err := svc.List(log, dto.ListDiagnosticCentresRequest{
		OrganisationID: "org-1",
		Limit:          1,
		PageNo:         2,
	})
	if err != nil || len(page2) != 1 || page1[0].ID == page2[0].ID {
		t.Fatalf("pagination failed: page1=%+v page2=%+v err=%v", page1, page2, err)
	}

	err = svc.Update(log, cityID, dto.UpdateDiagnosticCentreRequest{
		Name:          "City Care Diagnostics",
		Address:       "14 Main St",
		ContactNumber: "9000099999",
		Email:         "care@lab.com",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	listed, _, err := svc.List(log, dto.ListDiagnosticCentresRequest{
		OrganisationID: "org-1",
		Search:         "care",
		Limit:          10,
		PageNo:         1,
	})
	if err != nil || len(listed) != 1 || listed[0].Name != "City Care Diagnostics" || listed[0].Email != "care@lab.com" {
		t.Fatalf("after update list: %+v err=%v", listed, err)
	}

	if err := svc.UpdateStatus(log, cityID, "INACTIVE"); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	active, activeTotal, err := svc.List(log, dto.ListDiagnosticCentresRequest{
		OrganisationID: "org-1",
		Status:         diagnosticcentre.StatusActive,
		Limit:          10,
		PageNo:         1,
	})
	if err != nil || activeTotal != 1 || active[0].ID != metroID {
		t.Fatalf("active after deactivate: %+v total=%d err=%v", active, activeTotal, err)
	}
	inactive, inactiveTotal, err := svc.List(log, dto.ListDiagnosticCentresRequest{
		OrganisationID: "org-1",
		Status:         diagnosticcentre.StatusInactive,
		Search:         "care",
		Limit:          10,
		PageNo:         1,
	})
	if err != nil || inactiveTotal != 1 || inactive[0].ID != cityID {
		t.Fatalf("inactive filtered: %+v total=%d err=%v", inactive, inactiveTotal, err)
	}

	if err := svc.UpdateStatus(log, cityID, "active"); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	org2Rows, org2Total, err := svc.List(log, dto.ListDiagnosticCentresRequest{
		OrganisationID: "org-2",
		Limit:          10,
		PageNo:         1,
	})
	if err != nil || org2Total != 1 || org2Rows[0].ID != otherID {
		t.Fatalf("org isolation: %+v total=%d err=%v", org2Rows, org2Total, err)
	}

	if err := svc.Update(log, "missing", dto.UpdateDiagnosticCentreRequest{Name: "X"}); !errors.Is(err, wrapError.ErrDiagnosticCentreNotFound) {
		t.Fatalf("update missing: %v", err)
	}
	if err := svc.UpdateStatus(log, "missing", "inactive"); !errors.Is(err, wrapError.ErrDiagnosticCentreNotFound) {
		t.Fatalf("status missing: %v", err)
	}
}

func TestDiagnosticCentreComplexHTTPAllRoutes(t *testing.T) {
	repo := newMemoryRepo()
	svc := diagnosticcentre.NewDiagnosticCentreService(repo)
	ctrl := diagnosticcentre.NewDiagnosticCentreController(svc)

	app := controllertest.NewApp(t, func(app *fiber.App) {
		g := app.Group("/diagnostic-centres")
		g.Post("/", ctrl.Create)
		g.Get("/", ctrl.List)
		g.Put("/:id", ctrl.Update)
		g.Patch("/:id/status", ctrl.UpdateStatus)
	})

	resp, body := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/diagnostic-centres/",
		Body:   `{"organisation_id":"org-1","name":"Apollo Lab","address":"MG Road","contact_number":"98888","email":"apollo@lab.com"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
	var createResp map[string]any
	if err := json.Unmarshal(body, &createResp); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id, _ := createResp["diagnostic_centre_id"].(string)
	if id == "" {
		t.Fatalf("expected id in %s", body)
	}

	resp, body = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/diagnostic-centres/?organisation_id=org-1&search=apollo&status=active&limit=5&page_no=1",
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
	var listResp struct {
		Data  []dto.DiagnosticCentreResponse `json:"data"`
		Total int64                          `json:"total"`
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listResp.Total != 1 || listResp.Data[0].ID != id {
		t.Fatalf("unexpected list: %+v", listResp)
	}

	resp, _ = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPut,
		Path:   "/diagnostic-centres/" + id,
		Body:   `{"name":"Apollo Diagnostics","address":"MG Road 2","contact_number":"97777","email":"care@apollo.com"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)

	resp, _ = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPatch,
		Path:   "/diagnostic-centres/" + id + "/status",
		Body:   `{"status":"inactive"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)

	resp, body = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/diagnostic-centres/?organisation_id=org-1&status=inactive&limit=10&page_no=1",
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
	_ = json.Unmarshal(body, &listResp)
	if listResp.Total != 1 || listResp.Data[0].Status != diagnosticcentre.StatusInactive {
		t.Fatalf("expected inactive row, got %+v", listResp)
	}

	resp, _ = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPatch,
		Path:   "/diagnostic-centres/" + id + "/status",
		Body:   `{"status":"active"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)

	resp, _ = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/diagnostic-centres/",
		Body:   `{"organisation_id":"org-1"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)

	resp, _ = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/diagnostic-centres/?search=x",
	})
	controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)

	resp, _ = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPatch,
		Path:   "/diagnostic-centres/" + id + "/status",
		Body:   `{"status":"paused"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)

	resp, _ = controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPut,
		Path:   "/diagnostic-centres/" + uuid.New().String(),
		Body:   `{"name":"Ghost Lab"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusNotFound)
}
