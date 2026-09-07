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

const validSupplierBody = `{
	"user_id":"user-1",
	"organisation_id":"org-1",
	"name":"Pharma Supply Co",
	"payment_terms":"Net 30",
	"email_id":"supplier@example.com",
	"drug_license_number":"DL-123",
	"contact_number":"9876543210",
	"credit_limit":50000
}`

func TestCreateSupplier(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockSupplierServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing user_id",
			body:       `{"organisation_id":"org-1","name":"Co","payment_terms":"Net 30","email_id":"a@b.com","drug_license_number":"DL","contact_number":"9876543210","credit_limit":1}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing name",
			body:       `{"user_id":"u1","organisation_id":"org-1","payment_terms":"Net 30","email_id":"a@b.com","drug_license_number":"DL","contact_number":"9876543210","credit_limit":1}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service failure",
			body: validSupplierBody,
			setup: func(m *mocks.MockSupplierServicer) {
				m.EXPECT().CretateSupplier(gomock.Any(), gomock.Any()).Return("", wrapError.ErrSupplierCreateFailed)
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			body: validSupplierBody,
			setup: func(m *mocks.MockSupplierServicer) {
				m.EXPECT().CretateSupplier(gomock.Any(), gomock.Any()).Return("sup-1", nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockSupplierServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			supplierCtrl := medicine.NewSupplierController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/suppliers", supplierCtrl.CreateSupplier)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/suppliers",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetSupplierByID(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockSupplierServicer)
		wantStatus int
	}{
		{
			name:       "missing supplier_id",
			query:      "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "not found",
			query: "?supplier_id=sup-1",
			setup: func(m *mocks.MockSupplierServicer) {
				m.EXPECT().GetSupplierByID(gomock.Any(), gomock.Any()).Return(medicine.Supplier{}, wrapError.ErrSupplierNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name:  "success",
			query: "?supplier_id=sup-1",
			setup: func(m *mocks.MockSupplierServicer) {
				m.EXPECT().GetSupplierByID(gomock.Any(), gomock.Any()).Return(medicine.Supplier{ID: "sup-1", Name: "Pharma Supply Co"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockSupplierServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			supplierCtrl := medicine.NewSupplierController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/suppliers", supplierCtrl.GetSupplierByID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/suppliers" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetSupplierByOrgID(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockSupplierServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing organisation_id",
			body:       `{"limit":10,"page_no":1}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "success",
			body: `{"organisation_id":"org-1","limit":10,"page_no":1}`,
			setup: func(m *mocks.MockSupplierServicer) {
				m.EXPECT().GetSupplierByOrgID(gomock.Any(), gomock.Any()).Return([]dto.SupplierListItem{{ID: "sup-1", Name: "Pharma Supply Co"}}, int64(1), nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockSupplierServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			supplierCtrl := medicine.NewSupplierController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/suppliers/list", supplierCtrl.GetSupplierByOrgID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/suppliers/list",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestGetSupplierTotalCount(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockSupplierServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			query:      "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "success",
			query: "?organisation_id=org-1",
			setup: func(m *mocks.MockSupplierServicer) {
				m.EXPECT().GetTotalCount(gomock.Any(), gomock.Any()).Return(int64(5), nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockSupplierServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			supplierCtrl := medicine.NewSupplierController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/suppliers/count", supplierCtrl.GetTotalCount)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/suppliers/count" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
