package medicine_test

import (
	"net/http"
	"testing"

	"hospital-backend/internal/medicine"
	"hospital-backend/internal/medicine/dto"
	"hospital-backend/internal/medicine/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

const validPurchaseBody = `{
	"user_id":"user-1",
	"supplier_id":"sup-1",
	"organisation_id":"org-1",
	"invoice_no":"INV-001",
	"medicine_info":[{
		"medicine_id":"med-1",
		"purchase_qty_boxes":2,
		"units_per_box":10
	}]
}`

func TestGetByIDHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	medCtrl := medicine.NewMedicineController(mocks.NewMockMedicineServicer(ctrl))
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/medicines/:id", medCtrl.GetByIDHandler)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodGet,
		Path:   "/medicines/med-1",
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
}

func TestGetAllHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing limit",
			body:       `{"page_no":1}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing page_no",
			body:       `{"limit":10}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "success",
			body:       `{"limit":10,"page_no":1}`,
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			medCtrl := medicine.NewMedicineController(mocks.NewMockMedicineServicer(ctrl))
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/medicines/list", medCtrl.GetAllHandler)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/medicines/list",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestSearchMedicine(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockMedicineServicer)
		wantStatus int
	}{
		{
			name:       "missing name",
			query:      "?organisation_id=org-1",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing organisation_id",
			query:      "?name=para",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "search failure",
			query: "?name=para&organisation_id=org-1",
			setup: func(m *mocks.MockMedicineServicer) {
				m.EXPECT().SearchMedicine(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, wrapError.ErrMedicineSearchFailed)
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name:  "success",
			query: "?name=para&organisation_id=org-1",
			setup: func(m *mocks.MockMedicineServicer) {
				m.EXPECT().SearchMedicine(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dto.SearchMedicineItem{{ID: "med-1", Name: "Paracetamol"}}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockMedicineServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			medCtrl := medicine.NewMedicineController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/medicines/search", medCtrl.SearchMedicine)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/medicines/search" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestAddMedicine(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockMedicineServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing supplier_id",
			body:       `{"user_id":"u1","organisation_id":"org-1","invoice_no":"INV-1","medicine_info":[{"medicine_id":"m1","purchase_qty_boxes":1,"units_per_box":10}]}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "empty medicine_info",
			body:       `{"user_id":"u1","supplier_id":"s1","organisation_id":"org-1","invoice_no":"INV-1","medicine_info":[]}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "supplier not found",
			body: validPurchaseBody,
			setup: func(m *mocks.MockMedicineServicer) {
				m.EXPECT().CreateMedicine(gomock.Any(), gomock.Any()).Return(wrapError.ErrSupplierNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			body: validPurchaseBody,
			setup: func(m *mocks.MockMedicineServicer) {
				m.EXPECT().CreateMedicine(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockMedicineServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			medCtrl := medicine.NewMedicineController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/medicines/purchase", medCtrl.AddMedicine)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/medicines/purchase",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestAddMedicineNewItemValidation(t *testing.T) {
	body := `{
		"user_id":"user-1",
		"supplier_id":"sup-1",
		"organisation_id":"org-1",
		"invoice_no":"INV-001",
		"medicine_info":[{
			"purchase_qty_boxes":2,
			"units_per_box":10
		}]
	}`
	ctrl := gomock.NewController(t)
	medCtrl := medicine.NewMedicineController(mocks.NewMockMedicineServicer(ctrl))
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Post("/medicines/purchase", medCtrl.AddMedicine)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/medicines/purchase",
		Body:   body,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
}
