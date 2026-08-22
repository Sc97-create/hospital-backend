package prescription_test

import (
	"net/http"
	"testing"

	"hospital-backend/internal/prescription"
	"hospital-backend/internal/prescription/dto"
	"hospital-backend/internal/prescription/mocks"
	"hospital-backend/internal/testutil/controllertest"
	wrapError "hospital-backend/shared/error"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

const validCreatePrescriptionBody = `{
	"appointment_id":"appt-1",
	"organisation_id":"org-1",
	"prescribed_by":"doc-1",
	"medicine_array":[{"medicine_id":"med-1","duration":5,"duration_type":"days","quantity":10}]
}`

func TestCreatePrescription(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockPrescriptionServicer)
		wantStatus int
	}{
		{
			name:       "invalid json",
			body:       `{`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing appointment_id",
			body:       `{"organisation_id":"org-1","prescribed_by":"doc-1","medicine_array":[]}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "appointment not found",
			body: validCreatePrescriptionBody,
			setup: func(m *mocks.MockPrescriptionServicer) {
				m.EXPECT().CreatePrescription(gomock.Any(), gomock.Any()).Return("", wrapError.ErrAppointmentNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			body: validCreatePrescriptionBody,
			setup: func(m *mocks.MockPrescriptionServicer) {
				m.EXPECT().CreatePrescription(gomock.Any(), gomock.Any()).Return("rx-1", nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			pSvc := mocks.NewMockPrescriptionServicer(ctrl)
			if tt.setup != nil {
				tt.setup(pSvc)
			}
			rxCtrl := prescription.NewPrescriptionController(pSvc, mocks.NewMockPrescriptionItemServicer(ctrl))
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/prescriptions", rxCtrl.CreatePrescription)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/prescriptions",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindManyPrescriptions(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockPrescriptionServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			body:       `{"limit":10,"page_no":1}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "success",
			body: `{"organisation_id":"org-1","limit":10,"page_no":1}`,
			setup: func(m *mocks.MockPrescriptionServicer) {
				m.EXPECT().FindMany(gomock.Any(), gomock.Any()).Return([]dto.PrescriptionListItem{{ID: "rx-1"}}, int64(1), nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			pSvc := mocks.NewMockPrescriptionServicer(ctrl)
			if tt.setup != nil {
				tt.setup(pSvc)
			}
			rxCtrl := prescription.NewPrescriptionController(pSvc, mocks.NewMockPrescriptionItemServicer(ctrl))
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/prescriptions/list", rxCtrl.FindMany)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/prescriptions/list",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindByStatus(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockPrescriptionServicer)
		wantStatus int
	}{
		{
			name:       "missing organisation_id",
			query:      "?status=active",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:       "missing status",
			query:      "?organisation_id=org-1",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "success",
			query: "?organisation_id=org-1&status=active&limit=10&offset=0",
			setup: func(m *mocks.MockPrescriptionServicer) {
				m.EXPECT().FindByStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]dto.PrescriptionListItem{{ID: "rx-1"}}, int64(1), nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			pSvc := mocks.NewMockPrescriptionServicer(ctrl)
			if tt.setup != nil {
				tt.setup(pSvc)
			}
			rxCtrl := prescription.NewPrescriptionController(pSvc, mocks.NewMockPrescriptionItemServicer(ctrl))
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/prescriptions/status", rxCtrl.FindByStatus)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/prescriptions/status" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestAddPrescriptionItems(t *testing.T) {
	body := `{"prescription_id":"rx-1","prescribed_by":"doc-1","medicine_array":[{"medicine_id":"med-1"}]}`
	ctrl := gomock.NewController(t)
	pSvc := mocks.NewMockPrescriptionServicer(ctrl)
	pSvc.EXPECT().AddPrescriptionItems(gomock.Any(), gomock.Any()).Return(nil)
	rxCtrl := prescription.NewPrescriptionController(pSvc, mocks.NewMockPrescriptionItemServicer(ctrl))
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Post("/prescriptions/items", rxCtrl.AddPrescriptionItems)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/prescriptions/items",
		Body:   body,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
}

func TestUpdatePrescriptionItem(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*mocks.MockPrescriptionItemServicer)
		wantStatus int
	}{
		{
			name:       "missing prescription_item_id",
			body:       `{"medicine_id":"med-1"}`,
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name: "item not found",
			body: `{"prescription_item_id":"pi-1","medicine_id":"med-1"}`,
			setup: func(m *mocks.MockPrescriptionItemServicer) {
				m.EXPECT().UpdatePrescriptionItemByID(gomock.Any(), gomock.Any()).Return(wrapError.ErrPrescriptionItemNotFound)
			},
			wantStatus: fiber.StatusNotFound,
		},
		{
			name: "success",
			body: `{"prescription_item_id":"pi-1","medicine_id":"med-1"}`,
			setup: func(m *mocks.MockPrescriptionItemServicer) {
				m.EXPECT().UpdatePrescriptionItemByID(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			itemSvc := mocks.NewMockPrescriptionItemServicer(ctrl)
			if tt.setup != nil {
				tt.setup(itemSvc)
			}
			rxCtrl := prescription.NewPrescriptionController(mocks.NewMockPrescriptionServicer(ctrl), itemSvc)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Post("/prescriptions/item/update", rxCtrl.UpdatePrescriptionItem)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodPost,
				Path:   "/prescriptions/item/update",
				Body:   tt.body,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindPrescriptionByID(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockPrescriptionItemServicer)
		wantStatus int
	}{
		{
			name:       "missing prescription_id",
			query:      "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "success",
			query: "?prescription_id=rx-1",
			setup: func(m *mocks.MockPrescriptionItemServicer) {
				m.EXPECT().GetPrescriptionsByPIDWithLimit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]prescription.MixedPrescriptionItem{}, int64(0), nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			itemSvc := mocks.NewMockPrescriptionItemServicer(ctrl)
			if tt.setup != nil {
				tt.setup(itemSvc)
			}
			rxCtrl := prescription.NewPrescriptionController(mocks.NewMockPrescriptionServicer(ctrl), itemSvc)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/prescriptions/items", rxCtrl.FindPrescriptionByID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/prescriptions/items" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	body := `{"prescription_id":"rx-1","status":"completed"}`
	ctrl := gomock.NewController(t)
	pSvc := mocks.NewMockPrescriptionServicer(ctrl)
	pSvc.EXPECT().UpdateManualStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	rxCtrl := prescription.NewPrescriptionController(pSvc, mocks.NewMockPrescriptionItemServicer(ctrl))
	app := controllertest.NewApp(t, func(app *fiber.App) {
		app.Post("/prescriptions/status", rxCtrl.UpdateStatus)
	})
	resp, _ := controllertest.Do(t, app, controllertest.Request{
		Method: http.MethodPost,
		Path:   "/prescriptions/status",
		Body:   body,
	})
	controllertest.AssertStatus(t, resp, fiber.StatusOK)
}

func TestGetPrescriptionsByPatientID(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		setup      func(*mocks.MockPrescriptionServicer)
		wantStatus int
	}{
		{
			name:       "missing patient_id",
			query:      "",
			wantStatus: fiber.StatusBadRequest,
		},
		{
			name:  "success",
			query: "?patient_id=pat-1",
			setup: func(m *mocks.MockPrescriptionServicer) {
				m.EXPECT().GetPrescriptionsByPatientID(gomock.Any(), gomock.Any()).Return(dto.Response{Code: "200"}, nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			pSvc := mocks.NewMockPrescriptionServicer(ctrl)
			if tt.setup != nil {
				tt.setup(pSvc)
			}
			rxCtrl := prescription.NewPrescriptionController(pSvc, mocks.NewMockPrescriptionItemServicer(ctrl))
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/prescriptions/patient", rxCtrl.GetPrescriptionsByPatientID)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   "/prescriptions/patient" + tt.query,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestFindMedicineDetInfo(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*mocks.MockPrescriptionItemServicer)
		wantStatus int
	}{
		{
			name: "success",
			path: "/prescriptions/rx-1/medicines",
			setup: func(m *mocks.MockPrescriptionItemServicer) {
				m.EXPECT().GetMedicineInfo(gomock.Any(), gomock.Any()).Return([]prescription.MedicineDetInfo{}, int64(0), nil)
			},
			wantStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			itemSvc := mocks.NewMockPrescriptionItemServicer(ctrl)
			if tt.setup != nil {
				tt.setup(itemSvc)
			}
			rxCtrl := prescription.NewPrescriptionController(mocks.NewMockPrescriptionServicer(ctrl), itemSvc)
			app := controllertest.NewApp(t, func(app *fiber.App) {
				app.Get("/prescriptions/:prescription_id/medicines", rxCtrl.FindMedicineDetInfo)
			})
			resp, _ := controllertest.Do(t, app, controllertest.Request{
				Method: http.MethodGet,
				Path:   tt.path,
			})
			controllertest.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
