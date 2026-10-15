package diagnosticcentre_test

import (
	"net/http"
	"testing"

	diagnosticcentre "hospital-backend/internal/labService/diagnosticCentre"
	"hospital-backend/internal/labService/diagnosticCentre/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

// TestDiagnosticCentreControllerRejectsBadPayloads ensures malformed / wrong-type
// input never reaches the service (so nothing wrong is persisted, and handlers
// return 400 instead of panicking).
func TestDiagnosticCentreControllerRejectsBadPayloads(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "empty body create", method: http.MethodPost, path: "/diagnostic-centres", body: ``},
		{name: "null body create", method: http.MethodPost, path: "/diagnostic-centres", body: `null`},
		{name: "array body create", method: http.MethodPost, path: "/diagnostic-centres", body: `[]`},
		{name: "organisation_id as number", method: http.MethodPost, path: "/diagnostic-centres", body: `{"organisation_id":123,"name":"Lab"}`},
		{name: "organisation_id as bool", method: http.MethodPost, path: "/diagnostic-centres", body: `{"organisation_id":true,"name":"Lab"}`},
		{name: "organisation_id as object", method: http.MethodPost, path: "/diagnostic-centres", body: `{"organisation_id":{"id":"org-1"},"name":"Lab"}`},
		{name: "name as number", method: http.MethodPost, path: "/diagnostic-centres", body: `{"organisation_id":"org-1","name":99}`},
		{name: "name as array", method: http.MethodPost, path: "/diagnostic-centres", body: `{"organisation_id":"org-1","name":["Lab"]}`},
		{name: "whitespace name", method: http.MethodPost, path: "/diagnostic-centres", body: `{"organisation_id":"org-1","name":"   "}`},
		{name: "update name as number", method: http.MethodPut, path: "/diagnostic-centres/dc-1", body: `{"name":1}`},
		{name: "update whitespace name", method: http.MethodPut, path: "/diagnostic-centres/dc-1", body: `{"name":"  "}`},
		{name: "status as number", method: http.MethodPatch, path: "/diagnostic-centres/dc-1/status", body: `{"status":1}`},
		{name: "status as bool", method: http.MethodPatch, path: "/diagnostic-centres/dc-1/status", body: `{"status":true}`},
		{name: "status empty", method: http.MethodPatch, path: "/diagnostic-centres/dc-1/status", body: `{"status":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockDiagnosticCentreServicer(ctrl)
			// No EXPECT — service must not be called for bad payloads.
			dcCtrl := diagnosticcentre.NewDiagnosticCentreController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/diagnostic-centres", dcCtrl.Create)
				app.Put("/diagnostic-centres/:id", dcCtrl.Update)
				app.Patch("/diagnostic-centres/:id/status", dcCtrl.UpdateStatus)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: tt.method,
				Path:   tt.path,
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
		})
	}
}

func TestDiagnosticCentreControllerInvalidStatusEnum(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockDiagnosticCentreServicer(ctrl)
	mock.EXPECT().UpdateStatus(gomock.Any(), "dc-1", "deleted").Return(wrapError.ErrInvalidRequest)

	dcCtrl := diagnosticcentre.NewDiagnosticCentreController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Patch("/diagnostic-centres/:id/status", dcCtrl.UpdateStatus)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPatch,
		Path:   "/diagnostic-centres/dc-1/status",
		Body:   `{"status":"deleted"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
}

func TestDiagnosticCentreControllerListRejectsBadQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockDiagnosticCentreServicer(ctrl)
	dcCtrl := diagnosticcentre.NewDiagnosticCentreController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/diagnostic-centres", dcCtrl.List)
	})

	t.Run("missing organisation_id", func(t *testing.T) {
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodGet,
			Path:   "/diagnostic-centres?limit=10&page_no=1",
		})
		controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
	})

	t.Run("whitespace organisation_id", func(t *testing.T) {
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodGet,
			Path:   "/diagnostic-centres?organisation_id=%20%20",
		})
		controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
	})
}
