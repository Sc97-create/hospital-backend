package labservice_test

import (
	"errors"
	"net/http"
	"testing"

	labservice "hospital-backend/internal/labService/labTest"
	"hospital-backend/internal/labService/labTest/dto"
	"hospital-backend/internal/labService/labTest/mocks"
	"hospital-backend/internal/testutil/controllertest"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestLabOrderControllerRejectsBadPayloads(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "empty body create", method: http.MethodPost, path: "/lab-orders/create", body: ``},
		{name: "organisation_id as number", method: http.MethodPost, path: "/lab-orders/create", body: `{"organisation_id":1,"patient_id":"p","doctor_id":"d","diagnostic_centre_id":"c","tests":[{"name":"CBC"}]}`},
		{name: "tests as object", method: http.MethodPost, path: "/lab-orders/create", body: `{"organisation_id":"org-1","patient_id":"p","doctor_id":"d","diagnostic_centre_id":"c","tests":{"name":"CBC"}}`},
		{name: "whitespace test name", method: http.MethodPost, path: "/lab-orders/create", body: `{"organisation_id":"org-1","patient_id":"p","doctor_id":"d","diagnostic_centre_id":"c","tests":[{"name":"  "}]}`},
		{name: "status as number", method: http.MethodPatch, path: "/lab-orders/update/lo-1/status", body: `{"status":0}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockLabOrderServicer(ctrl)
			labCtrl := labservice.NewLabOrderController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/lab-orders/create", labCtrl.Create)
				app.Patch("/lab-orders/update/:id/status", labCtrl.UpdateStatus)
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

func TestLabOrderControllerInvalidStatusEnum(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockLabOrderServicer(ctrl)
	mock.EXPECT().UpdateStatus(gomock.Any(), "lo-1", "archived").Return(wrapError.ErrInvalidRequest)
	labCtrl := labservice.NewLabOrderController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Patch("/lab-orders/update/:id/status", labCtrl.UpdateStatus)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPatch,
		Path:   "/lab-orders/update/lo-1/status",
		Body:   `{"status":"archived"}`,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
}

func TestLabOrderServiceRejectsBadPersistenceInputs(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockLabOrderRepository(ctrl)
	svc := labservice.NewLabOrderService(repo)
	log := servicetest.NopLogger()

	t.Run("create whitespace test name", func(t *testing.T) {
		_, err := svc.Create(log, dto.CreateLabOrderRequest{
			OrganisationID:     "org-1",
			PatientID:          "p",
			DoctorID:           "d",
			DiagnosticCentreID: "c",
			Tests:              []dto.CreateOrderedTestInput{{Name: "  "}},
		})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("status unknown enum no exec", func(t *testing.T) {
		for _, status := range []string{"", "active", "1", "true"} {
			err := svc.UpdateStatus(log, "lo-1", status)
			if !errors.Is(err, wrapError.ErrInvalidRequest) {
				t.Fatalf("status %q: expected invalid request, got %v", status, err)
			}
		}
	})

	t.Run("list unknown status no find", func(t *testing.T) {
		_, _, err := svc.List(log, dto.ListLabOrdersRequest{OrganisationID: "org-1", Status: "archived"})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})
}
