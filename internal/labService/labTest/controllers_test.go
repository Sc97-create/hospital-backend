package labservice_test

import (
	"net/http"
	"testing"

	labservice "hospital-backend/internal/labService/labTest"
	"hospital-backend/internal/labService/labTest/dto"
	"hospital-backend/internal/labService/labTest/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

func TestLabOrderControllerCreate(t *testing.T) {
	validBody := `{
		"organisation_id":"org-1",
		"patient_id":"pat-1",
		"doctor_id":"doc-1",
		"diagnostic_centre_id":"dc-1",
		"tests":[{"name":"CBC","sub_test":"Hb"}]
	}`

	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockLabOrderServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing organisation_id",
			body:       `{"patient_id":"pat-1","doctor_id":"doc-1","diagnostic_centre_id":"dc-1","tests":[{"name":"CBC"}]}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing tests",
			body:       `{"organisation_id":"org-1","patient_id":"pat-1","doctor_id":"doc-1","diagnostic_centre_id":"dc-1"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "service failure",
			body: validBody,
			setup: func(m *mocks.MockLabOrderServicer) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return("", wrapError.ErrLabOrderCreateFailed)
			},
			wantStatus: fiber.StatusInternalServerError,
		},
		{
			name: "success",
			body: validBody,
			setup: func(m *mocks.MockLabOrderServicer) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return("lo-1", nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := mocks.NewMockLabOrderServicer(ctrl)
			if tt.setup != nil {
				tt.setup(mock)
			}
			labCtrl := labservice.NewLabOrderController(mock)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/lab-orders/create", labCtrl.Create)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/lab-orders/create",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestLabOrderControllerGetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockLabOrderServicer(ctrl)
	labCtrl := labservice.NewLabOrderController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/lab-orders/getbyID/:id", labCtrl.GetByID)
	})

	t.Run("not found", func(t *testing.T) {
		mock.EXPECT().GetByID(gomock.Any(), "missing").Return(dto.LabOrderResponse{}, wrapError.ErrLabOrderNotFound)
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodGet,
			Path:   "/lab-orders/getbyID/missing",
		})
		controllertest.AssertStatus(t, resp, fiber.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		mock.EXPECT().GetByID(gomock.Any(), "lo-1").Return(dto.LabOrderResponse{ID: "lo-1"}, nil)
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodGet,
			Path:   "/lab-orders/getbyID/lo-1",
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
	})
}

func TestLabOrderControllerList(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockLabOrderServicer(ctrl)
	labCtrl := labservice.NewLabOrderController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Get("/lab-orders/list", labCtrl.List)
	})

	t.Run("missing organisation_id", func(t *testing.T) {
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodGet,
			Path:   "/lab-orders/list",
		})
		controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		mock.EXPECT().List(gomock.Any(), gomock.Any()).Return([]dto.LabOrderListItem{{ID: "lo-1"}}, int64(1), nil)
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodGet,
			Path:   "/lab-orders/list?organisation_id=org-1&limit=10&page_no=1",
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
	})
}

func TestLabOrderControllerUpdateStatuses(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockLabOrderServicer(ctrl)
	labCtrl := labservice.NewLabOrderController(mock)
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Patch("/lab-orders/update/:id/status", labCtrl.UpdateStatus)
		app.Patch("/lab-orders/bulkupdatestatus", labCtrl.UpdateTestsStatus)
		app.Patch("/lab-orders/update/:id/tests/:testId/status", labCtrl.UpdateTestStatus)
	})

	t.Run("order status success", func(t *testing.T) {
		mock.EXPECT().UpdateStatus(gomock.Any(), "lo-1", "completed").Return(nil)
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodPatch,
			Path:   "/lab-orders/update/lo-1/status",
			Body:   `{"status":"completed"}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
	})

	t.Run("test status success", func(t *testing.T) {
		mock.EXPECT().UpdateTestStatus(gomock.Any(), "lo-1", "t1", "completed").Return(nil)
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodPatch,
			Path:   "/lab-orders/update/lo-1/tests/t1/status",
			Body:   `{"status":"completed"}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
	})

	t.Run("bulk tests status success", func(t *testing.T) {
		mock.EXPECT().UpdateTestsStatus(gomock.Any(), "lo-1", []string{"t1", "t2"}, "completed").Return(nil)
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodPatch,
			Path:   "/lab-orders/bulkupdatestatus",
			Body:   `{"lab_order_id":"lo-1","test_ids":["t1","t2"],"status":"completed"}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusOK)
	})

	t.Run("bulk tests status invalid body", func(t *testing.T) {
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodPatch,
			Path:   "/lab-orders/bulkupdatestatus",
			Body:   `{"status":"completed"}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
	})

	t.Run("status invalid body", func(t *testing.T) {
		resp, _ := controllertest.Do(t, app, controllertest.Request{
			Method: http.MethodPatch,
			Path:   "/lab-orders/update/lo-1/status",
			Body:   `{}`,
		})
		controllertest.AssertStatus(t, resp, fiber.StatusBadRequest)
	})
}
