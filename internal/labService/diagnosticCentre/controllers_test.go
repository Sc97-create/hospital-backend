package diagnosticcentre_test

import (
	"net/http"
	"testing"

	diagnosticcentre "hospital-backend/internal/labService/diagnosticCentre"
	"hospital-backend/internal/labService/diagnosticCentre/dto"
	"hospital-backend/internal/labService/diagnosticCentre/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestDiagnosticCentreControllerCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockDiagnosticCentreServicer)
		wantStatus int
	}{
		{name: "invalid json", body: `{`, wantStatus: fiber.StatusBadRequest},
		{name: "missing organisation_id", body: `{"name":"City Lab"}`, wantStatus: fiber.StatusBadRequest},
		{name: "missing name", body: `{"organisation_id":"org-1"}`, wantStatus: fiber.StatusBadRequest},
		{
			name: "service failure",
			body: `{"organisation_id":"org-1","name":"City Lab"}`,
			setup: func(m *mocks.MockDiagnosticCentreServicer) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return("", wrapError.ErrDiagnosticCentreCreateFailed)
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			body: `{"organisation_id":"org-1","name":"City Lab","address":"Main","contact_number":"999","email":"a@b.com"}`,
			setup: func(m *mocks.MockDiagnosticCentreServicer) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return("dc-1", nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockDiagnosticCentreServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			dcCtrl := diagnosticcentre.NewDiagnosticCentreController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/diagnostic-centres", dcCtrl.Create)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/diagnostic-centres",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestDiagnosticCentreControllerList(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockDiagnosticCentreServicer(ctrl)
	dcCtrl := diagnosticcentre.NewDiagnosticCentreController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/diagnostic-centres", dcCtrl.List)
	})

	t.Run("missing organisation_id", func(t *testing.T) {
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodGet,
			Path:   "/diagnostic-centres",
		})
		controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		mock.EXPECT().List(gomock.Any(), gomock.Any()).Return([]dto.DiagnosticCentreResponse{{ID: "dc-1"}}, int64(1), nil)
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodGet,
			Path:   "/diagnostic-centres?organisation_id=org-1&limit=10&page_no=1",
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
	})
}

func TestDiagnosticCentreControllerUpdateAndStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockDiagnosticCentreServicer(ctrl)
	dcCtrl := diagnosticcentre.NewDiagnosticCentreController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Put("/diagnostic-centres/:id", dcCtrl.Update)
		app.Patch("/diagnostic-centres/:id/status", dcCtrl.UpdateStatus)
	})

	t.Run("update success", func(t *testing.T) {
		mock.EXPECT().Update(gomock.Any(), "dc-1", gomock.Any()).Return(nil)
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodPut,
			Path:   "/diagnostic-centres/dc-1",
			Body:   `{"name":"City Lab","address":"Main"}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
	})

	t.Run("status success", func(t *testing.T) {
		mock.EXPECT().UpdateStatus(gomock.Any(), "dc-1", "inactive").Return(nil)
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodPatch,
			Path:   "/diagnostic-centres/dc-1/status",
			Body:   `{"status":"inactive"}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
	})

	t.Run("status invalid body", func(t *testing.T) {
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodPatch,
			Path:   "/diagnostic-centres/dc-1/status",
			Body:   `{}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
	})
}
